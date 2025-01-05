package service

import (
	"bpl/repository"
	"fmt"

	"gorm.io/gorm"
)

type EventService struct {
	event_repository            *repository.EventRepository
	scoring_category_repository *repository.ScoringCategoryRepository
}

func NewEventService(db *gorm.DB) *EventService {
	return &EventService{
		event_repository:            repository.NewEventRepository(db),
		scoring_category_repository: repository.NewScoringCategoryRepository(db),
	}
}

func (e *EventService) GetAllEvents() ([]*repository.Event, error) {
	return e.event_repository.FindAll()
}

func (e *EventService) CreateEvent(event *repository.Event) (*repository.Event, error) {
	if event.ID == 0 {
		event.ScoringCategory = &repository.ScoringCategory{Name: "default"}
	} else {
		currentEvent, err := e.event_repository.GetEventById(event.ID)
		if err != nil {
			return nil, fmt.Errorf("event with this id does not exist: %v", err)
		}
		event.ScoringCategoryID = currentEvent.ScoringCategoryID
	}
	if event.IsCurrent {
		err := e.event_repository.InvalidateCurrentEvent()
		if err != nil {
			return nil, err
		}
	}
	result := e.event_repository.DB.Save(event)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to save event: %v", result.Error)
	}
	return event, nil
}

func (e *EventService) GetEventById(eventId int, preloads ...string) (*repository.Event, error) {
	return e.event_repository.GetEventById(eventId, preloads...)
}

func (e *EventService) GetCurrentEvent(preloads ...string) (*repository.Event, error) {
	return e.event_repository.GetCurrentEvent(preloads...)
}

func (e *EventService) UpdateEvent(eventId int, updateEvent *repository.Event) (*repository.Event, error) {
	return e.event_repository.Update(eventId, updateEvent)
}

func (e *EventService) DeleteEvent(eventId int) error {
	return e.event_repository.Delete(eventId)
}
