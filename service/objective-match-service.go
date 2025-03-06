package service

import (
	"bpl/repository"
	"time"
)

type ObjectiveMatchService struct {
	objectiveMatchRepository *repository.ObjectiveMatchRepository
}

func NewObjectiveMatchService() *ObjectiveMatchService {
	return &ObjectiveMatchService{
		objectiveMatchRepository: repository.NewObjectiveMatchRepository(),
	}
}

func (e *ObjectiveMatchService) CreateItemMatches(matches map[int]int, userId int, stashChangeId int, eventId int, timestamp time.Time) []*repository.ObjectiveMatch {
	objectiveMatches := make([]*repository.ObjectiveMatch, 0)
	for objectiveId, number := range matches {
		objectiveMatch := &repository.ObjectiveMatch{
			ObjectiveId:   objectiveId,
			Timestamp:     timestamp,
			Number:        number,
			UserId:        userId,
			EventId:       eventId,
			StashChangeId: &stashChangeId,
		}
		objectiveMatches = append(objectiveMatches, objectiveMatch)
	}
	return objectiveMatches
}

func (e *ObjectiveMatchService) SaveMatches(matches []*repository.ObjectiveMatch, desyncedObjectIds []int) error {
	if len(desyncedObjectIds) > 0 {
		return e.objectiveMatchRepository.OverwriteMatches(matches, desyncedObjectIds)
	}
	return e.objectiveMatchRepository.SaveMatches(matches)
}

func (e *ObjectiveMatchService) GetKafkaConsumer(eventId int) (*repository.KafkaConsumer, error) {
	return e.objectiveMatchRepository.GetKafkaConsumer(eventId)
}

func (e *ObjectiveMatchService) SaveKafkaConsumerId(consumer *repository.KafkaConsumer) error {
	return e.objectiveMatchRepository.SaveKafkaConsumer(consumer)
}
