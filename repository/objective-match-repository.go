package repository

import (
	"time"

	"gorm.io/gorm"
)

type ObjectiveMatch struct {
	ID          int       `gorm:"primaryKey"`
	ObjectiveID int       `gorm:"not null"`
	Timestamp   time.Time `gorm:"not null"`
	Number      int       `gorm:"not null"`
	UserID      *int      `gorm:"null"`
	StashId     *string   `gorm:"null"` // Only relevant for item objectives
	ChangeId    *int64    `gorm:"null"` // Only relevant for item objectives
}

type StashChange struct {
	StashID   string `gorm:"primaryKey;not null"`
	ChangeID  int64  `gorm:"primaryKey;not null"`
	EventID   int    `gorm:"not null;references events(id)"`
	Timestamp time.Time
}

type ObjectiveMatchRepository struct {
	DB *gorm.DB
}

func NewObjectiveMatchRepository(db *gorm.DB) *ObjectiveMatchRepository {
	return &ObjectiveMatchRepository{DB: db}
}

func (r *ObjectiveMatchRepository) SaveMatches(objectiveMatches []*ObjectiveMatch) error {
	result := r.DB.CreateInBatches(objectiveMatches, 1000)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (r *ObjectiveMatchRepository) SaveStashChange(stashChange *StashChange) error {
	result := r.DB.Create(stashChange)
	if result.Error != nil {
		return result.Error
	}
	return nil
}
