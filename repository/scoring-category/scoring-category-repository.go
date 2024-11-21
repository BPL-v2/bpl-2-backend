package scoring_category_repository

import (
	model "bpl/model/dbmodel"

	"gorm.io/gorm"
)

func Save(db *gorm.DB, category *model.ScoringCategory) (*model.ScoringCategory, error) {
	result := db.Create(category)
	if result.Error != nil {
		return nil, result.Error
	}
	return category, nil
}
