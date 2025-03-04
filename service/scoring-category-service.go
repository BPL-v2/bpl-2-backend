package service

import (
	"bpl/repository"
	"bpl/utils"
)

type ScoringCategoryService struct {
	rules_repository *repository.ScoringCategoryRepository
	event_repository *repository.EventRepository
}

func NewScoringCategoryService() *ScoringCategoryService {
	return &ScoringCategoryService{
		rules_repository: repository.NewScoringCategoryRepository(),
		event_repository: repository.NewEventRepository(),
	}
}

func (e *ScoringCategoryService) GetCategoryById(categoryId int, preloads ...string) (*repository.ScoringCategory, error) {
	category, err := e.rules_repository.GetNestedCategories(categoryId, preloads...)
	if err != nil {
		return nil, err
	}
	return category, nil
}

func (e *ScoringCategoryService) GetRulesForEvent(eventId int, preloads ...string) (*repository.ScoringCategory, error) {
	event, err := e.event_repository.GetEventById(eventId)
	if err != nil {
		return nil, err
	}

	return e.rules_repository.GetNestedCategories(event.ScoringCategoryID, preloads...)
}

func (e *ScoringCategoryService) CreateCategory(category *repository.ScoringCategory) (*repository.ScoringCategory, error) {
	category, err := e.rules_repository.SaveCategory(category)
	if err != nil {
		return nil, err
	}
	return category, nil
}

func (e *ScoringCategoryService) UpdateCategory(categoryUpdate *repository.ScoringCategory) (*repository.ScoringCategory, error) {
	category, err := e.rules_repository.GetCategoryById(categoryUpdate.ID)
	if err != nil {
		return nil, err
	}
	if categoryUpdate.Name != "" {
		category.Name = categoryUpdate.Name
	}
	return e.rules_repository.SaveCategory(category)
}

func (e *ScoringCategoryService) DeleteCategoryById(categoryId int) error {
	return e.rules_repository.DeleteCategoryById(categoryId)
}

func (e *ScoringCategoryService) DuplicateScoringCategories(eventID int, scoringPresetMap map[int]int) (*repository.ScoringCategory, error) {
	event, err := e.event_repository.GetEventById(eventID)
	if err != nil {
		return nil, err
	}
	scoringCategories, err := e.rules_repository.GetNestedCategories(event.ScoringCategoryID, "Objectives", "Objectives.Conditions")
	if err != nil {
		return nil, err
	}
	newCategory := StripCategory(scoringCategories, scoringPresetMap)
	e.rules_repository.SaveCategory(newCategory)
	return newCategory, nil
}

func StripCategory(category *repository.ScoringCategory, scoringPresetMap map[int]int) *repository.ScoringCategory {
	newCategory := &repository.ScoringCategory{
		Name: category.Name,
		SubCategories: utils.Map(category.SubCategories, func(c *repository.ScoringCategory) *repository.ScoringCategory {
			return StripCategory(c, scoringPresetMap)
		}),
		ScoringPreset: category.ScoringPreset,
		Objectives: utils.Map(category.Objectives, func(o *repository.Objective) *repository.Objective {
			newObjective := &repository.Objective{
				Name:           o.Name,
				Extra:          o.Extra,
				RequiredAmount: o.RequiredAmount,
				Conditions: utils.Map(o.Conditions, func(c *repository.Condition) *repository.Condition {
					return &repository.Condition{
						Field:    c.Field,
						Operator: c.Operator,
						Value:    c.Value,
					}
				}),
				ObjectiveType: o.ObjectiveType,
				NumberField:   o.NumberField,
				Aggregation:   o.Aggregation,
				ValidFrom:     o.ValidFrom,
				ValidTo:       o.ValidTo,
				ScoringId:     o.ScoringId,
				ScoringPreset: o.ScoringPreset,
				SyncStatus:    o.SyncStatus,
			}
			if o.ScoringId != nil {
				if newId, ok := scoringPresetMap[*o.ScoringId]; ok {
					newObjective.ScoringId = &newId
				}
			}
			return newObjective

		}),
	}
	if category.ScoringId != nil {
		if newId, ok := scoringPresetMap[*category.ScoringId]; ok {
			newCategory.ScoringId = &newId
		}
	}
	return newCategory
}
