package repository

import (
	"bpl/config"
	"log"
	"time"

	"gorm.io/gorm"
)

type ObjectiveMatch struct {
	ObjectiveId   int       `gorm:"index:obj_match_obj;index:obj_match_obj_user;not null;references:objectives(id)"`
	Timestamp     time.Time `gorm:"not null"`
	Number        int       `gorm:"not null"`
	UserId        int       `gorm:"index:obj_match_user;index:obj_match_obj_user;not null;references:users(id)"`
	EventId       int       `gorm:"index:obj_match_event;not null;references:events(id)"`
	StashChangeId *int      `gorm:"index:obj_match_stash_change;references:stash_change(id)"`
}

type KafkaConsumer struct {
	EventId int `gorm:"primaryKey;not null;references events(id)"`
	GroupId int `gorm:"not null"`
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

func (r *ObjectiveMatchRepository) OverwriteMatches(objectiveMatches []*ObjectiveMatch, objectiveIds []int) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		t := time.Now()
		err := r.DeleteMatches(objectiveIds)
		if err != nil {
			return err
		}
		err = r.SaveMatches(objectiveMatches)
		if err != nil {
			return err
		}
		log.Printf("Overwrite took %s", time.Since(t))
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
		consumer.EventId = eventId
		consumer.GroupId = 1
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
func (r *ObjectiveMatchRepository) DeleteMatches(objectiveIds []int) error {
	return r.DB.Where("objective_id IN ?", objectiveIds).Delete(&ObjectiveMatch{}).Error
}
