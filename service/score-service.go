package service

import (
	"bpl/config"
	"bpl/scoring"
	"fmt"

	"gorm.io/gorm"
)

type ScoreMap map[int]map[int]*ScoreDifference

func (s ScoreMap) setDiff(score *scoring.Score, diff *ScoreDifference) {
	if _, ok := s[score.TeamId]; !ok {
		s[score.TeamId] = make(map[int]*ScoreDifference)
	}
	s[score.TeamId][score.Id] = diff
}

func (s ScoreMap) GetSimpleScore() map[int]int {
	scores := make(map[int]int)
	for _, teamScore := range s {
		for _, scoreDiff := range teamScore {
			scores[scoreDiff.Score.TeamId] += scoreDiff.Score.Points
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
	LatestScores     map[int]ScoreMap
	eventService     *EventService
	objectiveService *ObjectiveService
	db               *gorm.DB
}

func NewScoreService() *ScoreService {
	eventService := NewEventService()
	objectiveService := NewObjectiveService()
	return &ScoreService{
		db:               config.DatabaseConnection(),
		eventService:     eventService,
		objectiveService: objectiveService,
		LatestScores:     make(map[int]ScoreMap),
	}
}

func GetScoreDifference(prevDiff *ScoreDifference, scoreA *scoring.Score) *ScoreDifference {
	if prevDiff == nil {
		return &ScoreDifference{Score: scoreA, DiffType: Added}
	}
	scoreB := prevDiff.Score
	fieldDiff := make([]string, 0)
	if scoreB.Points != scoreA.Points {
		fieldDiff = append(fieldDiff, "Points")
	}
	if scoreB.UserId != scoreA.UserId {
		fieldDiff = append(fieldDiff, "UserId")
	}
	if scoreB.Rank != scoreA.Rank {
		fieldDiff = append(fieldDiff, "Rank")
	}
	if scoreB.Number != scoreA.Number {
		fieldDiff = append(fieldDiff, "Number")
	}
	if scoreB.Finished != scoreA.Finished {
		fieldDiff = append(fieldDiff, "Finished")
	}
	if len(fieldDiff) == 0 {
		return &ScoreDifference{Score: scoreA, DiffType: Unchanged}
	}
	return &ScoreDifference{Score: scoreA, FieldDiff: fieldDiff, DiffType: Changed}
}

func Diff(scoreMap ScoreMap, scores []*scoring.Score) (ScoreMap, ScoreMap) {
	newMap := make(ScoreMap)
	diffMap := make(ScoreMap)
	for _, score := range scores {
		scorediff := GetScoreDifference(scoreMap[score.TeamId][score.Id], score)
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
	newScores, err := s.calcScores(eventId)
	if err != nil {
		fmt.Println("Error calculating scores:", err)
		return nil, err
	}
	oldScore := s.LatestScores[eventId]
	newScoreMap, diff := Diff(oldScore, newScores)
	s.LatestScores[eventId] = newScoreMap
	if len(diff) == 0 {
		return nil, fmt.Errorf("no changes in scores")
	}
	return diff, nil
}

func (s *ScoreService) calcScores(eventId int) (score []*scoring.Score, err error) {

	event, err := s.eventService.GetEventById(eventId, "Teams", "Teams.Users")
	if err != nil {
		return nil, err
	}

	rootObjective, err := s.objectiveService.GetObjectiveTreeForEvent(event.Id, "ScoringPreset", "Conditions")
	if err != nil {
		return nil, err
	}

	matches, err := scoring.AggregateMatches(s.db, event, rootObjective.FlatMap())
	if err != nil {
		return nil, err
	}
	scores, err := scoring.EvaluateAggregations(rootObjective, matches)
	if err != nil {
		return nil, err
	}
	return scores, nil
}
