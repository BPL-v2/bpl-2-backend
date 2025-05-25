package service

import (
	"bpl/repository"
	"fmt"

	"gorm.io/gorm"
)

type ApplicationStatus string

const (
	ApplicationStatusApplied    ApplicationStatus = "applied"
	ApplicationStatusAccepted   ApplicationStatus = "accepted"
	ApplicationStatusWaitlisted ApplicationStatus = "waitlisted"
	ApplicationStatusNone       ApplicationStatus = "none"
)

type EventStatus struct {
	TeamId            *int              `json:"team_id"`
	IsTeamLead        bool              `json:"is_team_lead" binding:"required"`
	ApplicationStatus ApplicationStatus `json:"application_status" binding:"required"`
	NumberOfSignups   int               `json:"number_of_signups" binding:"required"`
}

type EventService struct {
	eventRepository         *repository.EventRepository
	scoringPresetRepository *repository.ScoringPresetRepository
	objectiveRepository     *repository.ObjectiveRepository
	teamService             *TeamService
	signupService           *SignupService
}

func NewEventService() *EventService {
	return &EventService{
		eventRepository:         repository.NewEventRepository(),
		scoringPresetRepository: repository.NewScoringPresetRepository(),
		objectiveRepository:     repository.NewObjectiveRepository(),
		teamService:             NewTeamService(),
		signupService:           NewSignupService(),
	}
}

func (e *EventService) GetAllEvents(preloads ...string) ([]*repository.Event, error) {
	return e.eventRepository.FindAll(preloads...)
}

func (e *EventService) CreateEvent(event *repository.Event) (*repository.Event, error) {
	if event.Id == 0 {
		event.Objectives = []*repository.Objective{{
			Name: "default",
		}}
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
func (e *EventService) CreateEventWithoutCategory(event *repository.Event) (*repository.Event, error) {
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

func (e *EventService) GetEventById(eventId int, preloads ...string) (*repository.Event, error) {
	return e.eventRepository.GetEventById(eventId, preloads...)
}

func (e *EventService) GetCurrentEvent(preloads ...string) (*repository.Event, error) {
	return e.eventRepository.GetCurrentEvent(preloads...)
}

func (e *EventService) DeleteEvent(event *repository.Event) error {
	err := e.objectiveRepository.DeleteObjectivesByEventId(event.Id)
	if err != nil {
		return err
	}
	err = e.eventRepository.Delete(event)
	if err != nil {
		return err
	}
	return e.scoringPresetRepository.DeletePresetsForEvent(event.Id)
}

func (e *EventService) GetEventByObjectiveId(objectiveId int) (*repository.Event, error) {
	return e.eventRepository.GetEventByObjectiveId(objectiveId)
}

func (e *EventService) GetEventByConditionId(conditionId int) (*repository.Event, error) {
	return e.eventRepository.GetEventByConditionId(conditionId)
}

func (e *EventService) GetEventStatus(event *repository.Event, user *repository.User) (*EventStatus, error) {
	eventStatus := &EventStatus{
		ApplicationStatus: ApplicationStatusNone,
	}
	signups, err := e.signupService.GetSignupsForEvent(event)
	if err != nil {
		return nil, err
	}
	eventStatus.NumberOfSignups = len(signups)
	if user == nil {
		return eventStatus, nil
	}
	team, err := e.teamService.GetTeamForUser(event.Id, user.Id)
	if err != nil && err != gorm.ErrRecordNotFound {
		return eventStatus, err
	}
	if team != nil {
		eventStatus.TeamId = &team.TeamId
		eventStatus.IsTeamLead = team.IsTeamLead
		eventStatus.ApplicationStatus = ApplicationStatusAccepted
	} else {
		count := 0
		for _, signup := range signups {
			count++
			if signup.UserId == user.Id {
				if count > event.MaxSize {
					eventStatus.ApplicationStatus = ApplicationStatusWaitlisted
				} else {
					eventStatus.ApplicationStatus = ApplicationStatusApplied
				}
			}
		}
	}
	return eventStatus, nil
}
