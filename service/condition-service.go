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
