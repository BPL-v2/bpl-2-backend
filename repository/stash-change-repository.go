package repository

import (
	"bpl/client"
	"bpl/config"
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

type StashChangeMessage struct {
	Stashes      []client.PublicStashChange
	ChangeId     string
	NextChangeId string
	Timestamp    time.Time
}

type StashChangeRepository struct {
	DB *gorm.DB
}

type ChangeId struct {
	CurrentChangeId string    `gorm:"not null"`
	NextChangeId    string    `gorm:"not null"`
	EventId         int       `gorm:"primaryKey;references events(id)"`
	Timestamp       time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
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

func (r *StashChangeRepository) GetLatestTimestamp(eventId int) (time.Time, error) {
	var latest StashChange
	err := r.DB.Order("timestamp DESC").First(&latest, "event_id = ?", eventId).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return time.Time{}, nil // No records found
		}
		return time.Time{}, err // Other error
	}
	return latest.Timestamp, nil
}

func (r *StashChangeRepository) CreateStashChangeIfNotExists(stashChange *StashChange) (*StashChange, error) {
	existing := &StashChange{}
	err := r.DB.First(existing, "stash_id = ? AND event_id = ? AND timestamp = ?", stashChange.StashId, stashChange.EventId, stashChange.Timestamp).Error
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
func (r *StashChangeRepository) SaveStashChangesConditionally(message StashChangeMessage, eventId int, sendFunc func([]byte) error) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		currentId := &ChangeId{
			CurrentChangeId: message.ChangeId,
			NextChangeId:    message.NextChangeId,
			EventId:         eventId,
			Timestamp:       message.Timestamp,
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

func (r *StashChangeRepository) GetChangeIdForEvent(event *Event) (ChangeId, error) {
	var changeId ChangeId
	err := r.DB.First(&changeId, "event_id = ?", event.Id).Error
	if err != nil {
		return ChangeId{}, err
	}
	return changeId, nil
}
