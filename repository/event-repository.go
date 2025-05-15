package repository

import (
	"bpl/client"
	"bpl/config"
	"bpl/utils"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type GameVersion string

const (
	PoE1 GameVersion = "poe1"
	PoE2 GameVersion = "poe2"
)

type Event struct {
	Id                   int                `gorm:"primaryKey"`
	Name                 string             `gorm:"not null"`
	IsCurrent            bool               `gorm:"not null"`
	GameVersion          GameVersion        `gorm:"not null"`
	MaxSize              int                `gorm:"not null"`
	WaitlistSize         int                `gorm:"not null"`
	ApplicationStartTime time.Time          `gorm:"not null"`
	ApplicationEndTime   time.Time          `gorm:"not null"`
	EventStartTime       time.Time          `gorm:"not null"`
	EventEndTime         time.Time          `gorm:"not null"`
	Public               bool               `gorm:"not null"`
	Locked               bool               `gorm:"not null"`
	Teams                []*Team            `gorm:"foreignKey:EventId;constraint:OnDelete:CASCADE"`
	ScoringCategories    []*ScoringCategory `gorm:"foreignKey:EventId;constraint:OnDelete:CASCADE"`
}

func (e *Event) GetRealm() *client.Realm {
	if e.GameVersion == PoE2 {
		realm := client.PoE2
		return &realm
	}
	return nil
}

type EventRepository struct {
	DB *gorm.DB
}

func NewEventRepository() *EventRepository {
	return &EventRepository{DB: config.DatabaseConnection()}
}

func (r *EventRepository) GetCurrentEvent(preloads ...string) (*Event, error) {
	var event *Event
	query := r.DB

	for _, preload := range preloads {
		if preload == "Teams.Users" {
			continue
		}
		query = query.Preload(preload)
	}

	result := query.Where(Event{IsCurrent: true}).First(&event)
	if result.Error != nil {
		return nil, fmt.Errorf("no current event found: %v", result.Error)
	}
	if len(preloads) > 0 && utils.Contains(preloads, "Teams.Users") {
		LoadUsersIntoEvent(r.DB, event)
	}

	return event, nil
}

func (r *EventRepository) GetEventById(eventId int, preloads ...string) (*Event, error) {
	var event *Event
	query := r.DB

	for _, preload := range preloads {
		if preload == "Teams.Users" {
			continue
		}
		query = query.Preload(preload)
	}

	result := query.First(&event, eventId)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to find event: %v", result.Error)
	}
	if len(preloads) > 0 && utils.Contains(preloads, "Teams.Users") {
		LoadUsersIntoEvent(r.DB, event)
	}
	return event, nil
}

func (r *EventRepository) InvalidateCurrentEvent() error {
	result := r.DB.Model(&Event{}).Where(Event{IsCurrent: true}).Update("is_current", false)
	if result.Error != nil {
		return fmt.Errorf("failed to invalidate current event: %v", result.Error)
	}
	return nil
}

func (r *EventRepository) Delete(event *Event) error {
	return r.DB.Delete(&event).Error
}

func (r *EventRepository) FindAll(preloads ...string) ([]*Event, error) {
	var events []*Event
	query := r.DB
	for _, preload := range preloads {
		query = query.Preload(preload)
	}
	result := query.Find(&events)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to find events: %v", result.Error)
	}
	return events, nil
}

func (r *EventRepository) GetEventByObjectiveId(objectiveId int) (*Event, error) {
	var event Event
	query := `
		SELECT * FROM events
		JOIN scoring_categories ON events.id = scoring_categories.event_id
		JOIN objectives ON scoring_categories.id = objectives.category_id
		WHERE objectives.id = ?
	`
	result := r.DB.Raw(query, objectiveId).Scan(&event)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to find event by objective id: %v", result.Error)
	}
	return &event, nil
}

func (r *EventRepository) GetEventByConditionId(conditionId int) (*Event, error) {
	var event Event
	query := `
		SELECT * FROM events
		JOIN scoring_categories ON events.id = scoring_categories.event_id
		JOIN objectives ON scoring_categories.id = objectives.category_id
		JOIN conditions ON objectives.id = conditions.objective_id
		WHERE conditions.id = ?
	`
	result := r.DB.Raw(query, conditionId).Scan(&event)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to find event by condition id: %v", result.Error)
	}
	return &event, nil
}
