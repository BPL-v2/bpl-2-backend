package service

import (
	"bpl/repository"
)

type ConditionService struct {
	conditionRepository *repository.ConditionRepository
	objectiveRepository *repository.ObjectiveRepository
}

func NewConditionService() *ConditionService {
	return &ConditionService{
		conditionRepository: repository.NewConditionRepository(),
		objectiveRepository: repository.NewObjectiveRepository(),
	}
}

func (e *ConditionService) CreateCondition(condition *repository.Condition) (*repository.Condition, error) {
	return e.conditionRepository.SaveCondition(condition)
}

func (e *ConditionService) DeleteCondition(conditionId int) error {
	return e.conditionRepository.DeleteCondition(conditionId)
}

func (e *ConditionService) GetConditionsByObjectiveId(objectiveId int) ([]*repository.Condition, error) {
	objective, err := e.objectiveRepository.GetObjectiveById(objectiveId, "Conditions")
	if err != nil {
		return nil, err
	}
	return objective.Conditions, nil
}
