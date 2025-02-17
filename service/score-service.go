package service

import (
	"bpl/config"
	"bpl/scoring"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"
)

type ScoreMap map[string]*ScoreDifference

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
	LatestScores           map[int]ScoreMap
	eventService           *EventService
	scoringCategoryService *ScoringCategoryService
	objectiveService       *ObjectiveService
	db                     *gorm.DB
}

func NewScoreService() *ScoreService {
	eventService := NewEventService()
	scoringCategoryService := NewScoringCategoryService()
	objectiveService := NewObjectiveService()
	return &ScoreService{
		db:                     config.DatabaseConnection(),
		eventService:           eventService,
		scoringCategoryService: scoringCategoryService,
		objectiveService:       objectiveService,
		LatestScores:           make(map[int]ScoreMap),
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
	if scoreB.UserID != scoreA.UserID {
		fieldDiff = append(fieldDiff, "UserID")
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

func Diff(scoreMap map[string]*ScoreDifference, scores []*scoring.Score) (ScoreMap, ScoreMap) {
	newMap := make(ScoreMap)
	diffMap := make(ScoreMap)
	for _, score := range scores {
		id := score.Identifier()
		scorediff := GetScoreDifference(scoreMap[id], score)
		newMap[id] = scorediff
		if scorediff.DiffType != Unchanged {
			diffMap[id] = scorediff
		}
	}
	for id, oldScore := range scoreMap {
		if _, ok := newMap[id]; !ok {
			diffMap[id] = &ScoreDifference{Score: oldScore.Score, DiffType: Removed}
		}
	}
	return newMap, diffMap
}

func (s *ScoreService) GetNewDiff(eventID int) (ScoreMap, error) {
	t := time.Now()
	newScores, err := s.calcScores(eventID)
	if err != nil {
		return nil, err
	}
	oldScore := s.LatestScores[eventID]
	newScoreMap, diff := Diff(oldScore, newScores)
	s.LatestScores[eventID] = newScoreMap
	log.Printf("Calculated scores for event %d in %d milliseconds", eventID, time.Since(t).Milliseconds())
	if len(diff) == 0 {
		log.Printf("No changes in scores")
		return nil, fmt.Errorf("no changes in scores")
	}
	return diff, nil
}

func (s *ScoreService) calcScores(eventId int) (score []*scoring.Score, err error) {

	event, err := s.eventService.GetEventById(eventId, "Teams", "Teams.Users")
	if err != nil {
		return nil, err
	}
	rules, err := s.scoringCategoryService.GetRulesForEvent(event.ID, "Objectives", "Objectives.Conditions", "ScoringPreset", "Objectives.ScoringPreset")
	if err != nil {
		return nil, err
	}
	objectives, err := s.objectiveService.GetObjectivesByEventId(event.ID)
	if err != nil {
		return nil, err
	}

	matches, err := scoring.AggregateMatches(s.db, event, objectives)
	if err != nil {
		return nil, err
	}
	scores, err := scoring.EvaluateAggregations(rules, matches)
	if err != nil {
		return nil, err
	}
	return scores, nil
}
