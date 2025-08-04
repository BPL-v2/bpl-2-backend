package service

import (
	"bpl/repository"
)

type ObjectiveMatchService struct {
	objectiveMatchRepository *repository.ObjectiveMatchRepository
	stashchangeRepository    *repository.StashChangeRepository
}

func NewObjectiveMatchService() *ObjectiveMatchService {
	return &ObjectiveMatchService{
		objectiveMatchRepository: repository.NewObjectiveMatchRepository(),
		stashchangeRepository:    repository.NewStashChangeRepository(),
	}
}

func (e *ObjectiveMatchService) CreateItemMatches(matches map[int]int, user *repository.TeamUserWithPoEToken, stashChange *repository.StashChange) []*repository.ObjectiveMatch {
	stashChange, err := e.stashchangeRepository.CreateStashChangeIfNotExists(stashChange)
	if err != nil {
		return nil
	}
	objectiveMatches := make([]*repository.ObjectiveMatch, 0)
	for objectiveId, number := range matches {
		objectiveMatch := &repository.ObjectiveMatch{
			ObjectiveId:   objectiveId,
			Timestamp:     stashChange.Timestamp,
			Number:        number,
			UserId:        user.UserId,
			TeamId:        user.TeamId,
			StashChangeId: &stashChange.Id,
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
