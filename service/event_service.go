package service

import (
	model "bpl/model/dbmodel"
	event_repository "bpl/repository/events"
	scoring_category_repository "bpl/repository/scoring-category"

	"gorm.io/gorm"
)

func GetAllEvents(db *gorm.DB) ([]model.Event, error) {
	return event_repository.FindAll(db)
}

func CreateEvent(db *gorm.DB, name string) (*model.Event, error) {
	event := &model.Event{Name: name}
	scoringCategory := &model.ScoringCategory{Name: "default"}
	scoringCategory.Event = *event
	category, err := scoring_category_repository.Save(db, scoringCategory)
	if err != nil {
		return nil, err
	}
	return &category.Event, nil
}

func GetEventById(db *gorm.DB, eventId int) (*model.Event, error) {
	return event_repository.GetEventById(db, eventId)
}
