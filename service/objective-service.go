package service

import (
	"bpl/parser"
	"bpl/repository"
	"bpl/utils"
)

type ObjectiveService struct {
	objectiveRepository       *repository.ObjectiveRepository
	conditionRepository       *repository.ConditionRepository
	scoringCategoryRepository *repository.ScoringCategoryRepository
}

func NewObjectiveService() *ObjectiveService {
	return &ObjectiveService{
		objectiveRepository:       repository.NewObjectiveRepository(),
		conditionRepository:       repository.NewConditionRepository(),
		scoringCategoryRepository: repository.NewScoringCategoryRepository(),
	}
}

func (e *ObjectiveService) CreateObjective(objective *repository.Objective) (*repository.Objective, error) {
	var err error
	// saving conditions separately is necessary for some weird reason, otherwise condition updates will not be saved
	if objective.Id != 0 {
		for _, condition := range objective.Conditions {
			condition.ObjectiveId = objective.Id
			res := e.objectiveRepository.DB.Save(condition)
			if res.Error != nil {
				return nil, res.Error
			}
		}
	}
	objective, err = e.objectiveRepository.SaveObjective(objective)
	if err != nil {
		return nil, err
	}
	return objective, nil
}

func (e *ObjectiveService) DeleteObjective(objectiveId int) error {
	return e.objectiveRepository.DeleteObjective(objectiveId)
}

func (e *ObjectiveService) GetObjectiveById(objectiveId int) (*repository.Objective, error) {
	return e.objectiveRepository.GetObjectiveById(objectiveId, "Conditions")
}

func (e *ObjectiveService) GetObjectivesByEventId(eventId int) ([]*repository.Objective, error) {
	category, err := e.scoringCategoryRepository.GetRulesForEvent(eventId, "Objectives", "Objectives.Conditions")
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

func (e *ObjectiveService) GetParser(eventId int) (*parser.ItemChecker, error) {
	cats, err := e.scoringCategoryRepository.GetCategoriesForEvent(eventId, "Objectives", "Objectives.Conditions")
	if err != nil {
		return nil, err
	}
	objectives := utils.FlatMap(cats, func(cat *repository.ScoringCategory) []*repository.Objective {
		return cat.Objectives
	})
	return parser.NewItemChecker(objectives)
}

func (e *ObjectiveService) StartSync(objectiveIds []int) error {
	return e.objectiveRepository.StartSync(objectiveIds)
}

func (e *ObjectiveService) SetSynced(objectiveIds []int) error {
	return e.objectiveRepository.FinishSync(objectiveIds)
}
