package service

import (
	"bpl/repository"
	"time"
)

type ObjectiveMatchService struct {
	objective_match_repository *repository.ObjectiveMatchRepository
}

func NewObjectiveMatchService() *ObjectiveMatchService {
	return &ObjectiveMatchService{
		objective_match_repository: repository.NewObjectiveMatchRepository(),
	}
}

func (e *ObjectiveMatchService) CreateMatches(matches map[int]int, userId int, stashChangeID int, eventId int, timestamp time.Time) []*repository.ObjectiveMatch {
	objectiveMatches := make([]*repository.ObjectiveMatch, 0)
	for objectiveId, number := range matches {
		objectiveMatch := &repository.ObjectiveMatch{
			ObjectiveID:   objectiveId,
			Timestamp:     timestamp,
			Number:        number,
			UserID:        userId,
			EventId:       eventId,
			StashChangeID: &stashChangeID,
		}
		objectiveMatches = append(objectiveMatches, objectiveMatch)
	}
	return objectiveMatches
}

func (e *ObjectiveMatchService) SaveMatches(matches []*repository.ObjectiveMatch, desyncedObjectIDs []int) error {
	if len(desyncedObjectIDs) > 0 {
		return e.objective_match_repository.OverwriteMatches(matches, desyncedObjectIDs)
	}
	return e.objective_match_repository.SaveMatches(matches)
}

func (e *ObjectiveMatchService) GetKafkaConsumer(eventId int) (*repository.KafkaConsumer, error) {
	return e.objective_match_repository.GetKafkaConsumer(eventId)
}

func (e *ObjectiveMatchService) SaveKafkaConsumerId(consumer *repository.KafkaConsumer) error {
	return e.objective_match_repository.SaveKafkaConsumer(consumer)
}
