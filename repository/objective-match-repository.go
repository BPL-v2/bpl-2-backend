package repository

import (
	"bpl/client"
	"bpl/config"
	"log"
	"time"

	"gorm.io/gorm"
)

type ObjectiveMatch struct {
	ObjectiveId   int       `gorm:"index:obj_match_obj;index:obj_match_obj_user;not null;references:objectives(id)"`
	Timestamp     time.Time `gorm:"not null"`
	Number        int       `gorm:"not null"`
	TeamId        int       `gorm:"not null;references:teams(id)"`
	UserId        *int      `gorm:"index:obj_match_user;index:obj_match_obj_user;references:users(id)"`
	StashChangeId *int      `gorm:"index:obj_match_stash_change;references:stash_change(id)"`
}

type ObjectiveValidation struct {
	ObjectiveId int         `gorm:"primaryKey;not null;references:objectives(id)"`
	Timestamp   time.Time   `gorm:"not null"`
	Item        client.Item `gorm:"type:jsonb;not null"`
}

type KafkaConsumer struct {
	EventId int `gorm:"primaryKey;not null;references events(id)"`
	GroupId int `gorm:"not null"`
}

type ObjectiveMatchRepository interface {
	SaveValidations(objectiveValidations []*ObjectiveValidation) error
	GetValidationsByEventId(eventId int) ([]*ObjectiveValidation, error)
	SaveMatches(objectiveMatches []*ObjectiveMatch) error
	OverwriteMatches(objectiveMatches []*ObjectiveMatch, objectiveIds []int) error
	GetKafkaConsumer(eventId int) (*KafkaConsumer, error)
	SaveKafkaConsumer(consumer *KafkaConsumer) error
	DeleteMatches(objectiveIds []int) error
}

type ObjectiveMatchRepositoryImpl struct {
	DB *gorm.DB
}

func NewObjectiveMatchRepository() ObjectiveMatchRepository {
	return &ObjectiveMatchRepositoryImpl{DB: config.DatabaseConnection()}
}

func (r *ObjectiveMatchRepositoryImpl) SaveValidations(objectiveValidations []*ObjectiveValidation) error {
	result := r.DB.Save(objectiveValidations)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (r *ObjectiveMatchRepositoryImpl) GetValidationsByEventId(eventId int) ([]*ObjectiveValidation, error) {
	var validations []*ObjectiveValidation
	query := `SELECT ov.*
			  FROM objective_validations ov
			  JOIN objectives o ON ov.objective_id = o.id
			  WHERE o.event_id = ?`
	result := r.DB.Raw(query, eventId).Scan(&validations)
	if result.Error != nil {
		return nil, result.Error
	}
	return validations, nil
}

func (r *ObjectiveMatchRepositoryImpl) SaveMatches(objectiveMatches []*ObjectiveMatch) error {
	result := r.DB.CreateInBatches(objectiveMatches, 1000)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (r *ObjectiveMatchRepositoryImpl) OverwriteMatches(objectiveMatches []*ObjectiveMatch, objectiveIds []int) error {
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

func (r *ObjectiveMatchRepositoryImpl) GetKafkaConsumer(eventId int) (*KafkaConsumer, error) {
	var consumer *KafkaConsumer
	result := r.DB.Where(KafkaConsumer{EventId: eventId}).First(&consumer)
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

func (r *ObjectiveMatchRepositoryImpl) SaveKafkaConsumer(consumer *KafkaConsumer) error {
	result := r.DB.Save(consumer)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (r *ObjectiveMatchRepositoryImpl) DeleteMatches(objectiveIds []int) error {
	return r.DB.Where("objective_id IN ?", objectiveIds).Delete(&ObjectiveMatch{}).Error
}
