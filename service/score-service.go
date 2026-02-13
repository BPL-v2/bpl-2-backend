package service

import (
	"bpl/client"
	"bpl/config"
	"bpl/repository"
	"bpl/scoring"
	"bpl/utils"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"gorm.io/gorm"
)

// Trie data structure for efficient string matching
type TrieNode struct {
	children map[rune]*TrieNode
	objId    *int // Only set if this is end of an objective name
}

func buildTrie(objectiveNameMap map[int]string) *TrieNode {
	root := &TrieNode{children: make(map[rune]*TrieNode)}
	for objId, objName := range objectiveNameMap {
		node := root
		for _, char := range objName {
			if node.children[char] == nil {
				node.children[char] = &TrieNode{children: make(map[rune]*TrieNode)}
			}
			node = node.children[char]
		}
		node.objId = &objId
	}
	return root
}

func findObjectiveId(itemName string, root *TrieNode) *int {
	// Try all possible starting positions in the item name
	for i := 0; i < len(itemName); i++ {
		node := root
		// Try to match from position i
		for j := i; j < len(itemName); j++ {
			char := rune(itemName[j])
			if node.children[char] == nil {
				break // No match possible from this position
			}
			node = node.children[char]
			if node.objId != nil {
				return node.objId // Found a complete match
			}
		}
	}
	return nil
}

type ScoreMap map[int]map[int]*ScoreDifference

func (s ScoreMap) setDiff(score *scoring.Score, diff *ScoreDifference) {
	if _, ok := s[score.TeamId]; !ok {
		s[score.TeamId] = make(map[int]*ScoreDifference)
	}
	s[score.TeamId][score.ObjectiveId] = diff
}

func (s ScoreMap) GetSimpleScore() map[int]int {
	scores := make(map[int]int)
	for _, teamScore := range s {
		for _, scoreDiff := range teamScore {
			scores[scoreDiff.Score.TeamId] += scoreDiff.Score.Points()
		}
	}
	return scores
}

type Difftype string

const (
	Added     Difftype = "Added"
	Removed   Difftype = "Removed"
	Changed   Difftype = "Changed"
	Unchanged Difftype = "Unchanged"
)

type ScoreDifference struct {
	Score     *scoring.Score
	FieldDiff []string
	DiffType  Difftype
}

type ScoreService struct {
	LatestScores      map[int]ScoreMap
	eventService      *EventService
	objectiveService  *ObjectiveService
	guildStashService *GuildStashService
	cachedDataService *CachedDataService
	userService       *UserService
	db                *gorm.DB
	// Mutex to protect concurrent access to calculation state
	calculationMutex sync.Mutex
	calculating      map[int]chan ScoreMap // Track which events are currently being calculated with result channels
}

func NewScoreService(PoEClient *client.PoEClient) *ScoreService {
	eventService := NewEventService()
	objectiveService := NewObjectiveService()
	return &ScoreService{
		db:                config.DatabaseConnection(),
		eventService:      eventService,
		objectiveService:  objectiveService,
		guildStashService: NewGuildStashService(PoEClient),
		cachedDataService: NewCachedDataService(),
		userService:       NewUserService(),
		LatestScores:      make(map[int]ScoreMap),
		calculating:       make(map[int]chan ScoreMap),
	}
}

func GetScoreDifference(prevDiff *ScoreDifference, scoreA *scoring.Score) *ScoreDifference {
	if prevDiff == nil {
		return &ScoreDifference{Score: scoreA, DiffType: Added}
	}
	scoreB := prevDiff.Score
	fieldDiff := make(map[string]bool)
	for presetId, completionA := range scoreA.PresetCompletions {
		completionB, ok := scoreB.PresetCompletions[presetId]
		if !ok {
			continue
		}
		if completionB.Points != completionA.Points {
			fieldDiff["Points"] = true
		}
		if completionB.Rank != completionA.Rank {
			fieldDiff["Rank"] = true
		}
		if completionB.Number != completionA.Number {
			fieldDiff["Number"] = true
		}
		if completionB.Finished != completionA.Finished {
			fieldDiff["Finished"] = true
		}
	}
	if len(fieldDiff) == 0 {
		return &ScoreDifference{Score: scoreA, DiffType: Unchanged}
	}
	return &ScoreDifference{Score: scoreA, FieldDiff: utils.Keys(fieldDiff), DiffType: Changed}
}

func Diff(scoreMap ScoreMap, scores []*scoring.Score) (ScoreMap, ScoreMap) {
	newMap := make(ScoreMap)
	diffMap := make(ScoreMap)
	for _, score := range scores {
		scorediff := GetScoreDifference(scoreMap[score.TeamId][score.ObjectiveId], score)
		newMap.setDiff(score, scorediff)
		if scorediff.DiffType != Unchanged {
			diffMap.setDiff(score, scorediff)
		}
	}
	for teamId, oldTeamScore := range scoreMap {
		for objectiveId, scoreDiff := range oldTeamScore {
			if _, ok := newMap[teamId][objectiveId]; !ok {
				diffMap.setDiff(scoreDiff.Score, &ScoreDifference{
					Score:    scoreDiff.Score,
					DiffType: Removed,
				})
			}
		}
	}
	return newMap, diffMap
}

func (s *ScoreService) GetNewDiff(eventId int) (ScoreMap, error) {
	// Check if calculation is already in progress for this event
	s.calculationMutex.Lock()
	if resultChan, exists := s.calculating[eventId]; exists {
		// Calculation is in progress, wait for the result
		s.calculationMutex.Unlock()
		result := <-resultChan
		return result, nil
	}

	// Create a channel to communicate the result to other waiting goroutines
	resultChan := make(chan ScoreMap, 1)
	defer close(resultChan)
	s.calculating[eventId] = resultChan
	s.calculationMutex.Unlock()

	// Ensure we clean up the calculation flag when done
	defer func() {
		s.calculationMutex.Lock()
		delete(s.calculating, eventId)
		s.calculationMutex.Unlock()
	}()

	newScores, err := s.calcScores(eventId)
	if err != nil {
		// Send empty result to notify waiting goroutines of the error
		return nil, err
	}

	oldScore := s.LatestScores[eventId]
	newScoreMap, diff := Diff(oldScore, newScores)
	s.LatestScores[eventId] = newScoreMap

	if len(diff) == 0 {
		// Send empty result to notify waiting goroutines
		return nil, fmt.Errorf("no changes in scores")
	}

	byteData, err := json.Marshal(newScoreMap)
	if err != nil {
		return nil, err
	}

	err = s.cachedDataService.SaveScore(eventId, byteData)
	if err != nil {
		return nil, err
	}

	// Send the result to all waiting goroutines
	resultChan <- diff
	return diff, nil
}

func (s *ScoreService) calcScores(eventId int) (score []*scoring.Score, err error) {
	event, err := s.eventService.GetEventById(eventId, "Teams", "Teams.Users")
	if err != nil {
		return nil, err
	}
	rootObjective, err := s.objectiveService.GetObjectiveTreeForEvent(event.Id, "ScoringPresets")
	if err != nil {
		return nil, err
	}
	matches := scoring.AggregateMatches(s.db, event, rootObjective.FlatMap())
	teamScores := make(map[int]map[int]*scoring.Score)
	for _, team := range event.Teams {
		teamScores[team.Id] = make(map[int]*scoring.Score)
		for _, obj := range rootObjective.FlatMap() {
			presetCompletions := make(map[int]*scoring.PresetCompletion)
			for _, preset := range obj.ScoringPresets {
				presetCompletions[preset.Id] = &scoring.PresetCompletion{
					ObjectiveId: obj.Id,
				}
			}
			teamScores[team.Id][obj.Id] = &scoring.Score{
				ObjectiveId:       obj.Id,
				TeamId:            team.Id,
				HideProgress:      obj.HideProgress,
				PresetCompletions: presetCompletions,
			}
		}
	}
	err = scoring.EvaluateAggregations(rootObjective, matches, teamScores)
	if err != nil {
		return nil, err
	}
	scores := make([]*scoring.Score, 0)
	for _, teamScore := range teamScores {
		for _, score := range teamScore {
			scores = append(scores, score)
		}
	}
	overrides, _ := s.GetPlayerAttributionsFromGuildstash(event, rootObjective)
	for _, score := range scores {
		override, ok := overrides[score.ObjectiveId][score.TeamId]
		if !ok || score.Timestamp().Before(override.Timestamp) {
			continue
		}
		for _, completion := range score.PresetCompletions {
			completion.UserId = override.UserId
			completion.Timestamp = override.Timestamp
		}
	}

	return scores, nil
}

func (s *ScoreService) GetCurrentScore(eventId int) (ScoreMap, error) {
	if s.LatestScores[eventId] != nil {
		return s.LatestScores[eventId], nil
	}
	cached, err := s.cachedDataService.GetLatestScore(eventId)
	if err == nil {
		score := make(ScoreMap)
		if err := json.Unmarshal(cached, &score); err == nil {
			s.LatestScores[eventId] = score
			return score, nil
		}
	}
	return s.GetNewDiff(eventId)
}

// IsCalculating returns true if a score calculation is currently in progress for the given event
func (s *ScoreService) IsCalculating(eventId int) bool {
	s.calculationMutex.Lock()
	defer s.calculationMutex.Unlock()
	_, exists := s.calculating[eventId]
	return exists
}

type PlayerOverwrite struct {
	UserId    int
	Timestamp time.Time
}

type TeamAttributionOverwrite = map[int]PlayerOverwrite
type AttributionOverwrites = map[int]TeamAttributionOverwrite

func (s *ScoreService) GetPlayerAttributionsFromGuildstash(event *repository.Event, objectiveTree *repository.Objective) (AttributionOverwrites, error) {
	overwrites := make(AttributionOverwrites)
	objectiveNameMap := make(map[int]string)
	// we can only definitively identify objectives that have their name or base type as their only condition
	for _, objective := range objectiveTree.FlatMap() {
		if len(objective.Conditions) == 1 {
			cond := objective.Conditions[0]
			if cond.Operator == repository.EQ &&
				(cond.Field == repository.BASE_TYPE || cond.Field == repository.NAME || cond.Field == repository.TYPE_LINE) {
				objectiveNameMap[objective.Id] = cond.Value
			}
		}
	}
	deposits, err := s.guildStashService.GetEarliestDeposits(event)
	if err != nil {
		return overwrites, err
	}

	// Build trie for efficient substring matching
	trie := buildTrie(objectiveNameMap)

	for _, deposit := range deposits {
		if objId := findObjectiveId(deposit.ItemName, trie); objId != nil {
			if overwrites[*objId] == nil {
				overwrites[*objId] = make(TeamAttributionOverwrite)
			}
			overwrites[*objId][deposit.TeamId] = PlayerOverwrite{
				UserId:    deposit.UserId,
				Timestamp: deposit.Timestamp,
			}

		}
	}
	return overwrites, nil
}
