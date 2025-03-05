package repository

import (
	"bpl/config"

	"gorm.io/gorm"
)

type ScoringCategory struct {
	Id        int    `gorm:"primaryKey foreignKey:CategoryId references:Id on:objectives"`
	Name      string `gorm:"not null"`
	ParentId  *int   `gorm:"null;references:scoring_category(id)"`
	ScoringId *int   `gorm:"null;references:scoring_presets(id)"`

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
	return r.GetNestedCategories(event.ScoringCategoryId, preloads...)
}

func (r *ScoringCategoryRepository) GetNestedCategories(categoryId int, preloads ...string) (*ScoringCategory, error) {
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
	query := r.DB
	for _, preload := range preloads {
		query = query.Preload(preload)
	}
	result := query.Where("id IN ?", uniques).Find(&scoringCategories)
	if result.Error != nil {
		return nil, result.Error
	}
	categoryMap := make(map[int]*ScoringCategory)
	for _, category := range scoringCategories {
		categoryMap[category.Id] = &category
	}

	for _, category := range categoryMap {
		for _, relation := range relations {
			if relation.ParentId == category.Id {
				category.SubCategories = append(category.SubCategories, categoryMap[relation.ChildId])
			}
		}
	}

	category := categoryMap[categoryId]
	return category, nil
}

type CategoryRelation struct {
	ChildId  int
	ParentId int
}

func (r *ScoringCategoryRepository) GetTreeStructure(parentId int) ([]CategoryRelation, error) {
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
                Relations relation ON category.parent_id = relation.Id
        )
        SELECT
			id as child_id,
            parent_id
        FROM
        	Relations;
    `

	if err := r.DB.Raw(query, parentId).Scan(&categories).Error; err != nil {
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
	result := r.DB.Save(category)
	if result.Error != nil {
		return nil, result.Error
	}
	return category, nil
}

func (r *ScoringCategoryRepository) DeleteCategoryById(categoryId int) error {
	category, err := r.GetNestedCategories(categoryId, "Objectives", "Objectives.Conditions")
	if err != nil {
		return err
	}
	for _, subCategory := range category.SubCategories {
		if err := r.DeleteCategory(subCategory); err != nil {
			return err
		}
	}
	return r.DeleteCategory(category)
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
