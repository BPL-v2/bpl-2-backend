package repository

import (
	"fmt"

	"gorm.io/gorm"
)

type Event struct {
	ID                int     `gorm:"primaryKey"`
	Name              string  `gorm:"not null"`
	ScoringCategoryID int     `gorm:"not null"`
	Teams             []*Team `gorm:"foreignKey:EventID;constraint:OnDelete:CASCADE"`
	IsCurrent         bool
	MaxSize           int
}

type EventRepository struct {
	DB *gorm.DB
}

func NewEventRepository(db *gorm.DB) *EventRepository {
	return &EventRepository{DB: db}
}

func (r *EventRepository) GetEventById(eventId int, preloads ...string) (*Event, error) {
	var event Event
	query := r.DB

	for _, preload := range preloads {
		query = query.Preload(preload)
	}

	result := query.First(&event, eventId)
	if result.Error != nil {
		return nil, fmt.Errorf("event with id %d not found", eventId)
	}
	return &event, nil
}

func (r *EventRepository) Save(event *Event) (*Event, error) {
	if event.IsCurrent {
		err := r.InvalidateCurrentEvent()
		if err != nil {
			return nil, err
		}
	}
	result := r.DB.Save(event)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to create event: %v", result.Error)
	}
	return event, nil
}

func (r *EventRepository) Update(eventId int, updateEvent *Event) (*Event, error) {
	event, err := r.GetEventById(eventId)
	if err != nil {
		return nil, err
	}
	if updateEvent.Name != "" {
		event.Name = updateEvent.Name
	}
	event.IsCurrent = updateEvent.IsCurrent
	if updateEvent.MaxSize != 0 {
		event.MaxSize = updateEvent.MaxSize
	}
	return r.Save(event)
}

func (r *EventRepository) InvalidateCurrentEvent() error {
	result := r.DB.Model(&Event{}).Where("is_current = ?", true).Update("is_current", false)
	if result.Error != nil {
		return fmt.Errorf("failed to invalidate current event: %v", result.Error)
	}
	return nil
}

func (r *EventRepository) Delete(eventId int) error {
	event, err := r.GetEventById(eventId)
	if err != nil {
		return err
	}
	result := r.DB.Delete(&event)
	if result.Error != nil {
		return fmt.Errorf("failed to delete event: %v", result.Error)
	}

	result = r.DB.Delete(ScoringCategory{}, event.ScoringCategoryID)
	if result.Error == nil {
		result = r.DB.Delete(Event{}, eventId)
	}
	return result.Error
}

func (r *EventRepository) FindAll() ([]Event, error) {
	var events []Event
	result := r.DB.Find(&events)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to find events: %v", result.Error)
	}
	return events, nil
}
