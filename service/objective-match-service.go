package service

import (
	"bpl/repository"
)

type ObjectiveMatchService interface {
	CreateItemMatches(matches map[int]int, userId *int, teamId int, stashChange *repository.StashChange) []*repository.ObjectiveMatch
	SaveMatches(matches []*repository.ObjectiveMatch, desyncedObjectIds []int) error
	GetKafkaConsumer(eventId int) (*repository.KafkaConsumer, error)
	SaveKafkaConsumerId(consumer *repository.KafkaConsumer) error
	GetValidationsByEventId(eventId int) ([]*repository.ObjectiveValidation, error)
}

type ObjectiveMatchServiceImpl struct {
	objectiveMatchRepository repository.ObjectiveMatchRepository
	stashchangeRepository    repository.StashChangeRepository
}

func NewObjectiveMatchService() ObjectiveMatchService {
	return &ObjectiveMatchServiceImpl{
		objectiveMatchRepository: repository.NewObjectiveMatchRepository(),
		stashchangeRepository:    repository.NewStashChangeRepository(),
	}
}

func (e *ObjectiveMatchServiceImpl) CreateItemMatches(matches map[int]int, userId *int, teamId int, stashChange *repository.StashChange) []*repository.ObjectiveMatch {
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
			TeamId:        teamId,
			UserId:        userId,
			StashChangeId: &stashChange.Id,
		}
		objectiveMatches = append(objectiveMatches, objectiveMatch)
	}
	return objectiveMatches
}

func (e *ObjectiveMatchServiceImpl) SaveMatches(matches []*repository.ObjectiveMatch, desyncedObjectIds []int) error {
	if len(desyncedObjectIds) > 0 {
		return e.objectiveMatchRepository.OverwriteMatches(matches, desyncedObjectIds)
	}
	return e.objectiveMatchRepository.SaveMatches(matches)
}

func (e *ObjectiveMatchServiceImpl) GetKafkaConsumer(eventId int) (*repository.KafkaConsumer, error) {
	return e.objectiveMatchRepository.GetKafkaConsumer(eventId)
}

func (e *ObjectiveMatchServiceImpl) SaveKafkaConsumerId(consumer *repository.KafkaConsumer) error {
	return e.objectiveMatchRepository.SaveKafkaConsumer(consumer)
}

func (e *ObjectiveMatchServiceImpl) GetValidationsByEventId(eventId int) ([]*repository.ObjectiveValidation, error) {
	return e.objectiveMatchRepository.GetValidationsByEventId(eventId)
}
