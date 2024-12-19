package service

import (
	"bpl/repository"

	"gorm.io/gorm"
)

type ConditionService struct {
	condition_repository *repository.ConditionRepository
	objective_repository *repository.ObjectiveRepository
}

func NewConditionService(db *gorm.DB) *ConditionService {
	return &ConditionService{
		condition_repository: repository.NewConditionRepository(db),
		objective_repository: repository.NewObjectiveRepository(db),
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
