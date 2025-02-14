package repository

import (
	"time"

	"gorm.io/gorm"
)

type StashChangeRepository struct {
	DB *gorm.DB
}
type StashChange struct {
	StashID      string `gorm:"primaryKey;not null"`
	IntChangeID  int64  `gorm:"primaryKey;not null"`
	NextChangeID string `gorm:" not null"`
	EventID      int    `gorm:"index;not null;references events(id)"`
	Timestamp    time.Time
}

func NewStashChangeRepository(db *gorm.DB) *StashChangeRepository {
	return &StashChangeRepository{DB: db}
}

func (r *StashChangeRepository) SaveStashChange(stashChange *StashChange) error {
	result := r.DB.Create(stashChange)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (r *StashChangeRepository) GetLatestStashChangeForEvent(event *Event) (stashChange *StashChange, err error) {
	result := r.DB.Where("event_id = ?", event.ID).Order("int_change_id desc").First(&stashChange)
	if result.Error != nil {
		return nil, result.Error
	}
	return stashChange, nil
}
