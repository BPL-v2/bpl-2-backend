package repository

import (
	"bpl/config"
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

type StashChangeRepository struct {
	DB *gorm.DB
}

type ChangeId struct {
	CurrentChangeId string `gorm:"not null"`
	NextChangeId    string `gorm:"not null"`
	EventId         int    `gorm:"primaryKey;references events(id)"`
}

type StashChange struct {
	Id        int    `gorm:"primaryKey;autoIncrement"`
	StashId   string `gorm:"not null"`
	EventId   int    `gorm:"index;not null;references events(id)"`
	Timestamp time.Time
}

func NewStashChangeRepository() *StashChangeRepository {
	return &StashChangeRepository{DB: config.DatabaseConnection()}
}

func (r *StashChangeRepository) CreateStashChangeIfNotExists(stashChange *StashChange) (*StashChange, error) {
	existing := &StashChange{}
	err := r.DB.First(existing, "stash_id = ? AND event_id = ?", stashChange.StashId, stashChange.EventId).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	if existing.Id != 0 {
		return existing, nil
	}
	err = r.DB.Create(stashChange).Error
	if err != nil {
		return nil, err
	}
	return stashChange, nil

}

func (r *StashChangeRepository) Save(stashChange *StashChange) error {
	return r.DB.Save(&stashChange).Error
}
func (r *StashChangeRepository) SaveStashChangesConditionally(message config.StashChangeMessage, eventId int, sendFunc func([]byte) error) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		currentId := &ChangeId{
			CurrentChangeId: message.ChangeId,
			NextChangeId:    message.NextChangeId,
			EventId:         eventId,
		}
		err := tx.Save(currentId).Error
		if err != nil {
			return err
		}
		data, err := json.Marshal(message)
		if err != nil {
			return err
		}
		return sendFunc(data)
	})
}

func (r *StashChangeRepository) GetChangeIdForEvet(event *Event) (ChangeId, error) {
	var changeId ChangeId
	err := r.DB.First(&changeId, "event_id = ?", event.Id).Error
	if err != nil {
		return ChangeId{}, err
	}
	return changeId, nil
}
