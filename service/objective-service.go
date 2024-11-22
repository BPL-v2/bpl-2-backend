package service

import (
	"bpl/parser"
	"bpl/repository"

	"gorm.io/gorm"
)

type ObjectiveService struct {
	rules_repository            *repository.ObjectiveRepository
	scoring_category_repository *repository.ScoringCategoryRepository
}

func NewObjectiveService(db *gorm.DB) *ObjectiveService {
	return &ObjectiveService{
		rules_repository:            repository.NewObjectiveRepository(db),
		scoring_category_repository: repository.NewScoringCategoryRepository(db),
	}
}

func (e *ObjectiveService) CreateObjective(categoryId int, objective *repository.Objective) (*repository.Objective, error) {
	category, err := e.scoring_category_repository.GetCategoryById(categoryId)
	if err != nil {
		return nil, err
	}
	err = parser.ValidateConditions(objective.Conditions)
	if err != nil {
		return nil, err
	}
	objective.CategoryID = category.ID
	objective, err = e.rules_repository.SaveObjective(objective)
	if err != nil {
		return nil, err
	}
	return objective, nil
}

func (e *ObjectiveService) DeleteObjective(objectiveId int) error {
	return e.rules_repository.DeleteObjective(objectiveId)
}

func (e *ObjectiveService) GetObjectivesByCategoryId(categoryId int) ([]*repository.Objective, error) {
	return e.rules_repository.GetObjectivesByCategoryId(categoryId)
}

func (e *ObjectiveService) GetObjectiveById(objectiveId int) (*repository.Objective, error) {
	return e.rules_repository.GetObjectiveById(objectiveId)
}

func (e *ObjectiveService) UpdateObjective(objectiveId int, updateObjective *repository.Objective) (*repository.Objective, error) {
	objective, err := e.rules_repository.GetObjectiveById(objectiveId)
	if err != nil {
		return nil, err
	}
	if updateObjective.Name != "" {
		objective.Name = updateObjective.Name
	}
	if updateObjective.RequiredNumber != 0 {
		objective.RequiredNumber = updateObjective.RequiredNumber
	}
	if updateObjective.ObjectiveType != "" {
		objective.ObjectiveType = updateObjective.ObjectiveType
	}
	if updateObjective.ValidFrom != nil {
		objective.ValidFrom = updateObjective.ValidFrom
	}
	if updateObjective.ValidTo != nil {
		objective.ValidTo = updateObjective.ValidTo
	}
	return e.rules_repository.SaveObjective(objective)
}
