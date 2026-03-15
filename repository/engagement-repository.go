package repository

import (
	"bpl/config"

	"gorm.io/gorm"
)

type Engagement struct {
	Name   string `gorm:"primaryKey;not null"`
	Number int    `gorm:"not null"`
}

type EngagementRepository interface {
	SaveEngagement(engagement *Engagement) error
	UpdateEngagement(engagement *Engagement) error
	GetEngagementByName(name string) (*Engagement, error)
}

type EngagementRepositoryImpl struct {
	DB *gorm.DB
}

func NewEngagementRepository() EngagementRepository {
	return &EngagementRepositoryImpl{DB: config.DatabaseConnection()}
}

func (r *EngagementRepositoryImpl) SaveEngagement(engagement *Engagement) error {
	return r.DB.Create(&engagement).Error
}

func (r *EngagementRepositoryImpl) UpdateEngagement(engagement *Engagement) error {
	return r.DB.Save(engagement).Error
}

func (r *EngagementRepositoryImpl) GetEngagementByName(name string) (*Engagement, error) {
	engagement := &Engagement{}
	err := r.DB.Where("name = ?", name).First(&engagement).Error
	if err != nil {
		return nil, err
	}
	return engagement, nil
}
