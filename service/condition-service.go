package service

import (
	"bpl/repository"
)

type ConditionService struct {
	condition_repository *repository.ConditionRepository
	objective_repository *repository.ObjectiveRepository
}

func NewConditionService() *ConditionService {
	return &ConditionService{
		condition_repository: repository.NewConditionRepository(),
		objective_repository: repository.NewObjectiveRepository(),
	}
}

func (e *ConditionService) CreateCondition(condition *repository.Condition) (*repository.Condition, error) {
	return e.condition_repository.SaveCondition(condition)
}

func (e *ConditionService) DeleteCondition(conditionId int) error {
	return e.condition_repository.DeleteCondition(conditionId)
}

func (e *ConditionService) GetConditionsByObjectiveId(objectiveId int) ([]*repository.Condition, error) {
	objective, err := e.objective_repository.GetObjectiveById(objectiveId, "Conditions")
	if err != nil {
		return nil, err
	}
	return objective.Conditions, nil
}
