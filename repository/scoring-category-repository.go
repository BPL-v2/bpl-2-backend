package repository

import (
	"bpl/config"
	"bpl/utils"
	"fmt"

	"gorm.io/gorm"
)

type ScoringCategory struct {
	Id            int                `gorm:"primaryKey foreignKey:CategoryId references:Id on:objectives"`
	Name          string             `gorm:"not null"`
	EventId       int                `gorm:"not null;references:events(id)"`
	ParentId      *int               `gorm:"null;references:scoring_category(id)"`
	ScoringId     *int               `gorm:"null;references:scoring_presets(id)"`
	SubCategories []*ScoringCategory `gorm:"foreignKey:ParentId;constraint:OnDelete:CASCADE"`
	Objectives    []*Objective       `gorm:"foreignKey:CategoryId;constraint:OnDelete:CASCADE"`
	ScoringPreset *ScoringPreset     `gorm:"foreignKey:ScoringId;references:Id"`
}

type ScoringCategoryRepository struct {
	DB *gorm.DB
}

func NewScoringCategoryRepository() *ScoringCategoryRepository {
	return &ScoringCategoryRepository{DB: config.DatabaseConnection()}
}

func (r *ScoringCategoryRepository) GetRulesForEvent(eventId int, preloads ...string) (*ScoringCategory, error) {
	var event Event
	result := r.DB.First(&event, "id = ?", eventId)
	if result.Error != nil {
		return nil, result.Error
	}
	return r.GetNestedCategoriesForEvent(event.Id, preloads...)
}

func addSubCategories(category *ScoringCategory, categories []*ScoringCategory) {
	for _, cat := range categories {
		if cat.ParentId != nil {
			if *cat.ParentId == category.Id {
				addSubCategories(cat, categories)
				category.SubCategories = append(category.SubCategories, cat)
			}
		}
	}
}

func (r *ScoringCategoryRepository) GetCategoriesForEvent(eventId int, preloads ...string) (categories []*ScoringCategory, err error) {
	query := r.DB
	for _, preload := range preloads {
		query = query.Preload(preload)
	}
	err = query.Find(&categories, "event_id = ?", eventId).Error
	if err != nil {
		return nil, err
	}
	return categories, nil
}

func (r *ScoringCategoryRepository) GetNestedCategoriesForEvent(eventId int, preloads ...string) (*ScoringCategory, error) {
	categories, err := r.GetCategoriesForEvent(eventId, preloads...)
	if err != nil {
		return nil, err
	}
	rootCategory, found := utils.Find(categories, func(cat *ScoringCategory) bool {
		return cat.ParentId == nil
	})
	if !found {
		return nil, fmt.Errorf("no root category found for event %d", eventId)
	}
	addSubCategories(rootCategory, categories)
	return rootCategory, nil
}

func (r *ScoringCategoryRepository) GetNestedCategoriesForCategory(categoryId int, preloads ...string) (*ScoringCategory, error) {
	// First get all ids of the categories involved in the tree structure and their parent ids
	rootCategory, err := r.GetCategoryById(categoryId, preloads...)
	if err != nil {
		return nil, err
	}
	categories, err := r.GetCategoriesForEvent(rootCategory.EventId, preloads...)
	if err != nil {
		return nil, err
	}
	addSubCategories(rootCategory, categories)
	return rootCategory, nil
}

type CategoryRelation struct {
	ChildId  int
	ParentId int
}

func (r *ScoringCategoryRepository) GetCategoryById(categoryId int, preloads ...string) (*ScoringCategory, error) {

	var category ScoringCategory
	query := r.DB
	for _, preload := range preloads {
		query = query.Preload(preload)
	}
	result := query.First(&category, "id = ?", categoryId)
	if result.Error != nil {
		return nil, result.Error
	}
	return &category, nil
}

func (r *ScoringCategoryRepository) SaveCategory(category *ScoringCategory) (*ScoringCategory, error) {
	result := r.DB.Save(category)
	if result.Error != nil {
		return nil, result.Error
	}
	return category, nil
}

func (r *ScoringCategoryRepository) DeleteCategory(category *ScoringCategory) error {
	for _, objective := range category.Objectives {
		for _, condition := range objective.Conditions {
			if err := r.DB.Delete(&condition).Error; err != nil {
				return err
			}
		}

		if err := r.DB.Delete(&objective).Error; err != nil {
			return err
		}
	}
	result := r.DB.Delete(&category)
	return result.Error
}

func (r *ScoringCategoryRepository) DeleteCategoriesForEvent(eventId int) error {
	categories, err := r.GetCategoriesForEvent(eventId, "Objectives", "Objectives.Conditions")
	if err != nil {
		return err
	}
	for _, category := range categories {
		if err := r.DeleteCategory(category); err != nil {
			return err
		}
	}
	return nil
}
