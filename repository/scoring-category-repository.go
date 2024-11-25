package repository

import (
	"gorm.io/gorm"
)

type ScoringCategory struct {
	ID             int                      `gorm:"primaryKey foreignKey:CategoryID references:ID on:objectives"`
	Name           string                   `gorm:"not null"`
	Inheritance    ScoringMethodInheritance `gorm:"type:bpl2.scoring_method_inheritance"`
	ParentID       *int                     `gorm:"null"`
	ScoringMethods []*ScoringMethod         `gorm:"foreignKey:CategoryID;constraint:OnDelete:CASCADE"`
	SubCategories  []*ScoringCategory       `gorm:"foreignKey:ParentID;constraint:OnDelete:CASCADE"`
	Objectives     []*Objective             `gorm:"foreignKey:CategoryID;constraint:OnDelete:CASCADE"`
	Event          Event                    `gorm:"foreignKey:ScoringCategoryID"`
}

type ScoringMethod struct {
	CategoryID int               `gorm:"primaryKey"`
	Type       ScoringMethodType `gorm:"primaryKey;type:bpl2.scoring_method_type"`
	Points     []int             `gorm:"type:integer[]"`
}

type ScoringMethodType string

const (
	PRESENCE          ScoringMethodType = "PRESENCE"
	RANKED            ScoringMethodType = "RANKED"
	RELATIVE_PRESENCE ScoringMethodType = "RELATIVE_PRESENCE"
)

type ScoringMethodInheritance string

const (
	OVERWRITE ScoringMethodInheritance = "OVERWRITE"
	INHERIT   ScoringMethodInheritance = "INHERIT"
	EXTEND    ScoringMethodInheritance = "EXTEND"
)

type ScoringCategoryRepository struct {
	DB *gorm.DB
}

func NewScoringCategoryRepository(db *gorm.DB) *ScoringCategoryRepository {
	return &ScoringCategoryRepository{DB: db}
}

func (r *ScoringCategoryRepository) GetRulesForEvent(eventId int) (*ScoringCategory, error) {
	var event Event
	result := r.DB.First(&event, "id = ?", eventId)
	if result.Error != nil {
		return nil, result.Error
	}
	return r.GetNestedCategories(event.ScoringCategoryID)
}

func (r *ScoringCategoryRepository) GetNestedCategories(categoryId int) (*ScoringCategory, error) {
	// First get all ids of the categories involved in the tree structure and their parent ids
	relations, err := r.GetTreeStructure(categoryId)
	if err != nil {
		return nil, err
	}
	var scoringCategories []ScoringCategory
	ids := make(map[int]bool)
	// manually add the root category id as it is not necessarily included in the relations as it has no parent
	ids[categoryId] = true
	for _, relation := range relations {
		ids[relation.ChildId] = true
	}
	uniques := make([]int, 0, len(ids))
	uniques = append(uniques, categoryId)
	for id := range ids {
		uniques = append(uniques, id)
	}

	query := r.DB.Preload("ScoringMethods").Preload("Objectives").Preload("Objectives.Conditions").Where("id IN ?", uniques)
	result := query.Find(&scoringCategories)
	if result.Error != nil {
		return nil, result.Error
	}
	categoryMap := make(map[int]*ScoringCategory)
	for _, category := range scoringCategories {
		categoryMap[category.ID] = &category
	}

	for _, category := range categoryMap {
		for _, relation := range relations {
			if relation.ParentID == category.ID {
				category.SubCategories = append(category.SubCategories, categoryMap[relation.ChildId])
			}
		}
	}

	category := categoryMap[categoryId]
	return category, nil
}

type CategoryRelation struct {
	ChildId  int
	ParentID int
}

func (r *ScoringCategoryRepository) GetTreeStructure(parentID int) ([]CategoryRelation, error) {
	var categories []CategoryRelation
	query := `
        WITH RECURSIVE Relations AS (
            SELECT
                id,
                parent_id
            FROM
                bpl2.scoring_categories
            WHERE
                parent_id = ?

            UNION ALL

            SELECT
                category.id,
                category.parent_id
            FROM
                bpl2.scoring_categories category
            INNER JOIN
                Relations relation ON category.parent_id = relation.ID
        )
        SELECT
			id as child_id,
            parent_id
        FROM
        	Relations;
    `

	if err := r.DB.Raw(query, parentID).Scan(&categories).Error; err != nil {
		return nil, err
	}

	return categories, nil
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
	result := r.DB.Create(category)
	if result.Error != nil {
		return nil, result.Error
	}
	return category, nil
}

func (r *ScoringCategoryRepository) DeleteCategory(categoryId int) error {
	result := r.DB.Delete(&ScoringCategory{}, "id = ?", categoryId)
	return result.Error
}
