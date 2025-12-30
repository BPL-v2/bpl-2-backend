package repository

import (
	"bpl/config"

	"gorm.io/gorm"
)

type Engagement struct {
	Name   string `gorm:"primaryKey;not null"`
	Number int    `gorm:"not null"`
}

type EngagementRepository struct {
	DB *gorm.DB
}

func NewEngagementRepository() *EngagementRepository {
	return &EngagementRepository{DB: config.DatabaseConnection()}
}

func (r *EngagementRepository) SaveEngagement(engagement *Engagement) error {
	return r.DB.Create(&engagement).Error
}

func (r *EngagementRepository) GetEngagementByName(name string) (*Engagement, error) {
	engagement := &Engagement{}
	err := r.DB.Where("name = ?", name).First(&engagement).Error
	if err != nil {
		return nil, err
	}
	return engagement, nil
}
