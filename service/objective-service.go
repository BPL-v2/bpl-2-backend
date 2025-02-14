package service

import (
	"bpl/parser"
	"bpl/repository"
	util "bpl/utils"
)

type ObjectiveService struct {
	objective_repository        *repository.ObjectiveRepository
	condition_repository        *repository.ConditionRepository
	scoring_category_repository *repository.ScoringCategoryRepository
}

func NewObjectiveService() *ObjectiveService {
	return &ObjectiveService{
		objective_repository:        repository.NewObjectiveRepository(),
		condition_repository:        repository.NewConditionRepository(),
		scoring_category_repository: repository.NewScoringCategoryRepository(),
	}
}

func (e *ObjectiveService) CreateObjective(objective *repository.Objective) (*repository.Objective, error) {
	var err error
	// saving conditions separately is necessary for some weird reason, otherwise condition updates will not be saved
	if objective.ID != 0 {
		for _, condition := range objective.Conditions {
			condition.ObjectiveID = objective.ID
			res := e.objective_repository.DB.Save(condition)
			if res.Error != nil {
				return nil, res.Error
			}
		}
	}
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
	return e.objective_repository.GetObjectiveById(objectiveId, "Conditions")
}

func (e *ObjectiveService) GetObjectivesByEventId(eventID int) ([]*repository.Objective, error) {
	category, err := e.scoring_category_repository.GetRulesForEvent(eventID, "Objectives", "Objectives.Conditions")
	if err != nil {
		return nil, err
	}
	objectives := make([]*repository.Objective, 0)
	extractObjectives(category, &objectives)
	return objectives, nil
}

func extractObjectives(category *repository.ScoringCategory, objectives *[]*repository.Objective) {
	for _, subCategory := range category.SubCategories {
		*objectives = append(*objectives, subCategory.Objectives...)
		extractObjectives(subCategory, objectives)
	}
}

func (e *ObjectiveService) UpdateObjective(objectiveId int, updateObjective *repository.Objective) (*repository.Objective, error) {
	objective, err := e.objective_repository.GetObjectiveById(objectiveId)
	if err != nil {
		return nil, err
	}
	if updateObjective.Name != "" {
		objective.Name = updateObjective.Name
	}
	if updateObjective.RequiredAmount != 0 {
		objective.RequiredAmount = updateObjective.RequiredAmount
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

func (e *ObjectiveService) StartSync(objectiveIds []int) error {
	return e.objective_repository.StartSync(objectiveIds)
}

func (e *ObjectiveService) SetSynced(objectiveIds []int) error {
	return e.objective_repository.FinishSync(objectiveIds)
}
