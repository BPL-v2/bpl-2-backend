package service

import (
	"bpl/repository"

	"gorm.io/gorm"
)

type ScoringCategoryService struct {
	rules_repository *repository.ScoringCategoryRepository
	event_repository *repository.EventRepository
}

func NewScoringCategoryService(db *gorm.DB) *ScoringCategoryService {
	return &ScoringCategoryService{
		rules_repository: repository.NewScoringCategoryRepository(db),
		event_repository: repository.NewEventRepository(db),
	}
}

func (e *ScoringCategoryService) GetCategoryById(categoryId int) (*repository.ScoringCategory, error) {
	category, err := e.rules_repository.GetNestedCategories(categoryId)
	if err != nil {
		return nil, err
	}
	return category, nil
}

func (e *ScoringCategoryService) GetRulesForEvent(eventId int) (*repository.ScoringCategory, error) {
	event, err := e.event_repository.GetEventById(eventId)
	if err != nil {
		return nil, err
	}
	return e.rules_repository.GetNestedCategories(event.ScoringCategoryID)
}

func (e *ScoringCategoryService) CreateCategory(parentId int, category *repository.ScoringCategory) (*repository.ScoringCategory, error) {
	parentCategory, err := e.rules_repository.GetCategoryById(parentId)
	if err != nil {
		return nil, err
	}
	category.ParentID = &parentCategory.ID
	category, err = e.rules_repository.SaveCategory(category)
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
	if categoryUpdate.Inheritance != "" {
		category.Inheritance = categoryUpdate.Inheritance
	}
	return e.rules_repository.SaveCategory(category)
}

func (e *ScoringCategoryService) DeleteCategory(categoryId int) error {
	return e.rules_repository.DeleteCategory(categoryId)
}
