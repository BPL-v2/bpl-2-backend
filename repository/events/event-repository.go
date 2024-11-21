package event_repository

import (
	model "bpl/model/dbmodel"

	"gorm.io/gorm"
)

func GetEventById(db *gorm.DB, eventId int) (*model.Event, error) {
	var event model.Event
	result := db.First(&event, eventId)
	if result.Error != nil {
		return nil, result.Error
	}
	return &event, nil
}

func Save(db *gorm.DB, event *model.Event) (*model.Event, error) {
	result := db.Create(event)
	if result.Error != nil {
		return nil, result.Error
	}
	return event, nil
}

func Update(db *gorm.DB, eventId int, eventName string) (*model.Event, error) {
	event, err := GetEventById(db, eventId)
	if err != nil {
		return nil, err
	}
	event.Name = eventName
	result := db.Save(&event)
	if result.Error != nil {
		return nil, result.Error
	}
	return event, nil
}

func Delete(db *gorm.DB, eventId int) error {
	result := db.Delete(model.Event{}, eventId)
	return result.Error
}

func FindAll(db *gorm.DB) ([]model.Event, error) {
	var events []model.Event
	result := db.Find(&events)
	if result.Error != nil {
		return nil, result.Error
	}
	return events, nil
}
