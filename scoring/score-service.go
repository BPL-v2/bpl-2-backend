// should be in service package, but would lead to circular imports

package scoring

import (
	"bpl/service"
	"fmt"
	"log"
	"strconv"
	"time"

	"gorm.io/gorm"
)

func (s *Score) Identifier() string {
	if s.Type == OBJECTIVE {
		return "O-" + strconv.Itoa(s.ID) + "-" + strconv.Itoa(s.TeamID)
	} else {
		return "C-" + strconv.Itoa(s.ID) + "-" + strconv.Itoa(s.TeamID)
	}
}

type ScoreMap map[string]*ScoreDifference

type Difftype string

const (
	Added     Difftype = "Added"
	Removed   Difftype = "Removed"
	Changed   Difftype = "Changed"
	Unchanged Difftype = "Unchanged"
)

type ScoreDifference struct {
	Score     *Score
	FieldDiff []string
	DiffType  Difftype
}

type ScoreService struct {
	LatestScores           map[int]ScoreMap
	eventService           *service.EventService
	scoringCategoryService *service.ScoringCategoryService
	objectiveService       *service.ObjectiveService
	db                     *gorm.DB
}

func NewScoreService(db *gorm.DB) *ScoreService {
	eventService := service.NewEventService(db)
	scoringCategoryService := service.NewScoringCategoryService(db)
	objectiveService := service.NewObjectiveService(db)
	return &ScoreService{
		db:                     db,
		eventService:           eventService,
		scoringCategoryService: scoringCategoryService,
		objectiveService:       objectiveService,
		LatestScores:           make(map[int]ScoreMap),
	}
}

func GetScoreDifference(prevDiff *ScoreDifference, scoreB *Score) *ScoreDifference {
	if prevDiff == nil {
		return &ScoreDifference{Score: scoreB, DiffType: Added}
	}
	scoreA := prevDiff.Score
	fieldDiff := make([]string, 0)
	if scoreA.Points != scoreB.Points {
		fieldDiff = append(fieldDiff, "Points")
	}
	if scoreA.UserID != scoreB.UserID {
		fieldDiff = append(fieldDiff, "UserID")
	}
	if scoreA.Rank != scoreB.Rank {
		fieldDiff = append(fieldDiff, "Rank")
	}
	if scoreA.Number != scoreB.Number {
		fieldDiff = append(fieldDiff, "Number")
	}
	if scoreA.Finished != scoreB.Finished {
		fieldDiff = append(fieldDiff, "Finished")
	}
	if len(fieldDiff) == 0 {
		return &ScoreDifference{Score: scoreB, DiffType: Unchanged}
	}
	return &ScoreDifference{Score: scoreB, FieldDiff: fieldDiff, DiffType: Changed}
}

func Diff(scoreMap map[string]*ScoreDifference, scores []*Score) (ScoreMap, ScoreMap) {
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

func (s *ScoreService) calcScores(eventId int) (score []*Score, err error) {

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

	matches, err := AggregateMatches(s.db, event, objectives)
	if err != nil {
		return nil, err
	}
	scores, err := EvaluateAggregations(rules, matches)
	if err != nil {
		return nil, err
	}
	return scores, nil
}
