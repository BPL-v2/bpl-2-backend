package service

import (
	"bpl/repository"
	"fmt"
)

type EventService struct {
	eventRepository           *repository.EventRepository
	scoringCategoryRepository *repository.ScoringCategoryRepository
	scoringPresetRepository   *repository.ScoringPresetRepository
}

func NewEventService() *EventService {
	return &EventService{
		eventRepository:           repository.NewEventRepository(),
		scoringCategoryRepository: repository.NewScoringCategoryRepository(),
		scoringPresetRepository:   repository.NewScoringPresetRepository(),
	}
}

func (e *EventService) GetAllEvents(preloads ...string) ([]*repository.Event, error) {
	return e.eventRepository.FindAll(preloads...)
}

func (e *EventService) CreateEvent(event *repository.Event) (*repository.Event, error) {
	if event.Id == 0 {
		category, err := e.scoringCategoryRepository.SaveCategory(&repository.ScoringCategory{Name: "default"})
		if err != nil {
			return nil, fmt.Errorf("failed to create default scoring category: %v", err)
		}
		event.ScoringCategory = category
	} else {
		currentEvent, err := e.eventRepository.GetEventById(event.Id)
		if err != nil {
			return nil, fmt.Errorf("event with this id does not exist: %v", err)
		}
		event.ScoringCategoryId = currentEvent.ScoringCategoryId
	}
	if event.IsCurrent {
		err := e.eventRepository.InvalidateCurrentEvent()
		if err != nil {
			return nil, err
		}
	}
	result := e.eventRepository.DB.Save(event)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to save event: %v", result.Error)
	}
	return event, nil
}

func (e *EventService) SaveEvent(event *repository.Event) (*repository.Event, error) {
	return e.eventRepository.Save(event)
}

func (e *EventService) GetEventById(eventId int, preloads ...string) (*repository.Event, error) {
	return e.eventRepository.GetEventById(eventId, preloads...)
}

func (e *EventService) GetCurrentEvent(preloads ...string) (*repository.Event, error) {
	return e.eventRepository.GetCurrentEvent(preloads...)
}

func (e *EventService) UpdateEvent(eventId int, updateEvent *repository.Event) (*repository.Event, error) {
	return e.eventRepository.Update(eventId, updateEvent)
}

func (e *EventService) DeleteEvent(eventId int) error {
	event, err := e.eventRepository.GetEventById(eventId)
	if err != nil {
		return err
	}
	err = e.eventRepository.Delete(event)
	if err != nil {
		return err
	}
	err = e.scoringCategoryRepository.DeleteCategoryById(event.ScoringCategoryId)
	if err != nil {
		return err
	}
	return e.scoringPresetRepository.DeletePresetsForEvent(eventId)
}
