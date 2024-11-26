package service

import (
	"bpl/repository"

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

func (e *EventService) GetAllEvents() ([]repository.Event, error) {
	return e.event_repository.FindAll()
}

func (e *EventService) CreateEvent(event *repository.Event) (*repository.Event, error) {
	if event.IsCurrent {
		err := e.event_repository.InvalidateCurrentEvent()
		if err != nil {
			return nil, err
		}
	}
	scoringCategory := &repository.ScoringCategory{Name: "default", Event: *event, Inheritance: repository.OVERWRITE}
	category, err := e.scoring_category_repository.SaveCategory(scoringCategory)
	if err != nil {
		return nil, err
	}
	event.ScoringCategoryID = category.ID
	return &category.Event, nil
}

func (e *EventService) GetEventById(eventId int, preloads ...string) (*repository.Event, error) {
	return e.event_repository.GetEventById(eventId, preloads...)
}

func (e *EventService) UpdateEvent(eventId int, updateEvent *repository.Event) (*repository.Event, error) {
	return e.event_repository.Update(eventId, updateEvent)
}

func (e *EventService) DeleteEvent(eventId int) error {
	return e.event_repository.Delete(eventId)
}
