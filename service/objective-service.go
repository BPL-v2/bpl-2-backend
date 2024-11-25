package service

import (
	"bpl/parser"
	"bpl/repository"
	util "bpl/utils"

	"gorm.io/gorm"
)

type ObjectiveService struct {
	objective_repository        *repository.ObjectiveRepository
	scoring_category_repository *repository.ScoringCategoryRepository
}

func NewObjectiveService(db *gorm.DB) *ObjectiveService {
	return &ObjectiveService{
		objective_repository:        repository.NewObjectiveRepository(db),
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
	objective, err = e.objective_repository.SaveObjective(objective)
	if err != nil {
		return nil, err
	}
	return objective, nil
}

func (e *ObjectiveService) DeleteObjective(objectiveId int) error {
	return e.objective_repository.DeleteObjective(objectiveId)
}

func (e *ObjectiveService) GetObjectivesByCategoryId(categoryId int) ([]*repository.Objective, error) {
	return e.objective_repository.GetObjectivesByCategoryId(categoryId)
}

func (e *ObjectiveService) GetObjectiveById(objectiveId int) (*repository.Objective, error) {
	return e.objective_repository.GetObjectiveById(objectiveId)
}

func (e *ObjectiveService) UpdateObjective(objectiveId int, updateObjective *repository.Objective) (*repository.Objective, error) {
	objective, err := e.objective_repository.GetObjectiveById(objectiveId)
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
	return e.objective_repository.SaveObjective(objective)
}

func (e *ObjectiveService) GetParser(categoryId int) (*parser.ItemChecker, error) {
	relations, err := e.scoring_category_repository.GetTreeStructure(categoryId)
	if err != nil {
		return nil, err
	}

	categoryIds := make([]int, len(relations))
	categoryIds = append(categoryIds, categoryId)
	for _, relation := range relations {
		categoryIds = append(categoryIds, relation.ChildId)
	}
	objectives, _ := e.objective_repository.GetObjectivesByCategoryIds(util.Uniques(categoryIds))
	return parser.NewItemChecker(objectives)
}
