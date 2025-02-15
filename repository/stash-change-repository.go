package repository

import (
	"bpl/config"
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

func NewStashChangeRepository() *StashChangeRepository {
	return &StashChangeRepository{DB: config.DatabaseConnection()}
}

func (r *StashChangeRepository) SaveStashChangesConditionally(stashChanges []*StashChange, condFunc func() error) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		result := r.DB.CreateInBatches(stashChanges, 1000)
		if result.Error != nil {
			return result.Error
		}
		err := condFunc()
		if err != nil {
			return err
		}
		return nil
	})
}
func (r *StashChangeRepository) GetLatestStashChangeForEvent(event *Event) (stashChange *StashChange, err error) {
	result := r.DB.Where("event_id = ?", event.ID).Order("int_change_id desc").First(&stashChange)
	if result.Error != nil {
		return nil, result.Error
	}
	return stashChange, nil
}
