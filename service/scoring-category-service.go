package service

import (
	"bpl/repository"
	"bpl/utils"
)

type ScoringCategoryService struct {
	rulesRepository *repository.ScoringCategoryRepository
	eventRepository *repository.EventRepository
}

func NewScoringCategoryService() *ScoringCategoryService {
	return &ScoringCategoryService{
		rulesRepository: repository.NewScoringCategoryRepository(),
		eventRepository: repository.NewEventRepository(),
	}
}

func (e *ScoringCategoryService) GetCategoryById(categoryId int, preloads ...string) (*repository.ScoringCategory, error) {
	category, err := e.rulesRepository.GetNestedCategoriesForCategory(categoryId, preloads...)
	if err != nil {
		return nil, err
	}
	return category, nil
}

func (e *ScoringCategoryService) GetRulesForEvent(eventId int, preloads ...string) (*repository.ScoringCategory, error) {
	return e.rulesRepository.GetNestedCategoriesForEvent(eventId, preloads...)
}

func (e *ScoringCategoryService) CreateCategory(category *repository.ScoringCategory) (*repository.ScoringCategory, error) {
	category, err := e.rulesRepository.SaveCategory(category)
	if err != nil {
		return nil, err
	}
	return category, nil
}

func (e *ScoringCategoryService) UpdateCategory(categoryUpdate *repository.ScoringCategory) (*repository.ScoringCategory, error) {
	category, err := e.rulesRepository.GetCategoryById(categoryUpdate.Id)
	if err != nil {
		return nil, err
	}
	if categoryUpdate.Name != "" {
		category.Name = categoryUpdate.Name
	}
	return e.rulesRepository.SaveCategory(category)
}

func (e *ScoringCategoryService) DeleteCategory(category *repository.ScoringCategory) error {
	return e.rulesRepository.DeleteCategory(category)
}

func (e *ScoringCategoryService) DuplicateScoringCategories(oldEventId int, newEventId int, scoringPresetMap map[int]int) (*repository.ScoringCategory, error) {
	scoringCategories, err := e.rulesRepository.GetNestedCategoriesForEvent(oldEventId, "Objectives", "Objectives.Conditions")
	if err != nil {
		return nil, err
	}
	newCategory := StripCategory(scoringCategories, scoringPresetMap, newEventId)
	return e.rulesRepository.SaveCategory(newCategory)
}

func StripCategory(category *repository.ScoringCategory, scoringPresetMap map[int]int, newEventId int) *repository.ScoringCategory {
	newCategory := &repository.ScoringCategory{
		Name: category.Name,
		SubCategories: utils.Map(category.SubCategories, func(c *repository.ScoringCategory) *repository.ScoringCategory {
			return StripCategory(c, scoringPresetMap, newEventId)
		}),
		ParentId:      nil,
		EventId:       newEventId,
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
