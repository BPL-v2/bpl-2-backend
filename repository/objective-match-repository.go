package repository

import (
	"bpl/config"
	"time"

	"gorm.io/gorm"
)

type ObjectiveMatch struct {
	ID          int       `gorm:"primaryKey"`
	ObjectiveID int       `gorm:"index:obj_match_obj;index:obj_match_obj_user;not null;references:objectives(id)"`
	Timestamp   time.Time `gorm:"not null"`
	Number      int       `gorm:"not null"`
	UserID      int       `gorm:"index:obj_match_user;index:obj_match_obj_user;not null;references:users(id)"`
	EventId     int       `gorm:"index:obj_match_event;not null;references:events(id)"`
	StashId     *string   `gorm:"index:obj_match_stash_change;null;references:stash_change(int_change_id)"` // Only relevant for item objectives
	ChangeId    *int64    `gorm:"index:obj_match_stash_change;null;references:stash_change(change_id)"`     // Only relevant for item objectives
}

type KafkaConsumer struct {
	EventID int `gorm:"primaryKey;not null;references events(id)"`
	GroupID int `gorm:"not null"`
}

type ObjectiveMatchRepository struct {
	DB *gorm.DB
}

func NewObjectiveMatchRepository() *ObjectiveMatchRepository {
	return &ObjectiveMatchRepository{DB: config.DatabaseConnection()}
}

func (r *ObjectiveMatchRepository) SaveMatches(objectiveMatches []*ObjectiveMatch) error {
	result := r.DB.CreateInBatches(objectiveMatches, 1000)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (r *ObjectiveMatchRepository) OverwriteMatches(objectiveMatches []*ObjectiveMatch, changeIds []int64, objectiveIds []int) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		err := r.DeleteMatches(changeIds, objectiveIds)
		if err != nil {
			return err
		}
		err = r.SaveMatches(objectiveMatches)
		if err != nil {
			return err
		}
		return nil
	})
}

func (r *ObjectiveMatchRepository) DeleteMatch(id int) error {
	result := r.DB.Delete(&ObjectiveMatch{}, id)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (r *ObjectiveMatchRepository) GetKafkaConsumer(eventId int) (*KafkaConsumer, error) {
	var consumer *KafkaConsumer
	result := r.DB.Where("event_id = ?", eventId).First(&consumer)
	if result.Error != nil {
		consumer.EventID = eventId
		consumer.GroupID = 1
		result = r.DB.Create(consumer)
		if result.Error != nil {
			return nil, result.Error
		}
		return consumer, nil

	}
	return consumer, nil
}

func (r *ObjectiveMatchRepository) SaveKafkaConsumer(consumer *KafkaConsumer) error {
	result := r.DB.Save(consumer)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (r *ObjectiveMatchRepository) DeleteOldMatches(changeId int64, objectiveIds []int) error {
	result := r.DB.
		Where("change_id < ? AND objective_id IN (?)", changeId, objectiveIds).
		Delete(&ObjectiveMatch{})
	if result.Error != nil {
		return result.Error
	}
	return nil
}
func (r *ObjectiveMatchRepository) DeleteMatches(changeIds []int64, objectiveIds []int) error {
	result := r.DB.
		Where("change_id in (?) AND objective_id IN (?)", changeIds, objectiveIds).
		Delete(&ObjectiveMatch{})
	if result.Error != nil {
		return result.Error
	}
	return nil
}
