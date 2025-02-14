package service

import (
	"bpl/repository"
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

func (e *ScoringCategoryService) DeleteCategory(categoryId int) error {
	return e.rules_repository.DeleteCategory(categoryId)
}
