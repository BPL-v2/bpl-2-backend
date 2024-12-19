package service

import (
	"bpl/repository"
	"time"

	"gorm.io/gorm"
)

type ObjectiveMatchService struct {
	objective_match_repository *repository.ObjectiveMatchRepository
}

func NewObjectiveMatchService(db *gorm.DB) *ObjectiveMatchService {
	return &ObjectiveMatchService{
		objective_match_repository: repository.NewObjectiveMatchRepository(db),
	}
}

func (e *ObjectiveMatchService) SaveItemMatches(matches map[int]int, userId int, changeId int64, stashId string) error {
	objectiveMatches := make([]*repository.ObjectiveMatch, 0)
	now := time.Now()
	for objectiveId, number := range matches {
		objectiveMatch := &repository.ObjectiveMatch{
			ObjectiveID: objectiveId,
			Timestamp:   now,
			Number:      number,
			UserID:      &userId,
			StashId:     &stashId,
			ChangeId:    &changeId,
		}
		objectiveMatches = append(objectiveMatches, objectiveMatch)
	}
	return e.objective_match_repository.SaveMatches(objectiveMatches)
}

func (e *ObjectiveMatchService) SaveStashChange(stashId string, changeId int64) error {
	stashChange := &repository.StashChange{
		StashID:   stashId,
		ChangeID:  changeId,
		Timestamp: time.Now(),
	}
	return e.objective_match_repository.SaveStashChange(stashChange)
}
