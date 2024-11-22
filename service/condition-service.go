package service

import (
	"bpl/parser"
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
	objective, err := e.objective_repository.GetObjectiveById(condition.ObjectiveID, "Conditions")
	if err != nil {
		return nil, err
	}
	objective.Conditions = append(objective.Conditions, condition)
	err = parser.ValidateConditions(objective.Conditions)
	if err != nil {
		return nil, err
	}
	objective, err = e.objective_repository.SaveObjective(objective)
	if err != nil {
		return nil, err
	}
	condition.ObjectiveID = objective.ID
	return condition, nil
}

func (e *ConditionService) UpdateCondition(conditionId int, updateCondition *repository.Condition) (*repository.Condition, error) {
	condition, err := e.condition_repository.GetConditionById(conditionId)
	if err != nil {
		return nil, err
	}
	if updateCondition.Field != "" {
		condition.Field = updateCondition.Field
	}
	if updateCondition.Operator != "" {
		condition.Operator = updateCondition.Operator
	}
	if updateCondition.Value != "" {
		condition.Value = updateCondition.Value
	}
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
