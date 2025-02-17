package service

import (
	"bpl/client"
	"bpl/repository"
	"context"
	"fmt"
	"time"
)

type JobType string

const (
	FetchStashChanges    JobType = "FetchStashChanges"
	EvaluateStashChanges JobType = "EvaluateStashChanges"
	// CalculateScores      JobType = "CalculateScores"
	FetchCharacterData JobType = "FetchCharacterData"
)

type RecurringJob struct {
	JobType           JobType            `json:"job_type"`
	SleepAfterEachRun time.Duration      `json:"sleep_after_each_run"`
	Cancel            context.CancelFunc `json:"-"`
	EndDate           *time.Time         `json:"end_date"`
	EventId           *int               `json:"event_id"`
}

type RecurringJobService struct {
	objective_repository        *repository.ObjectiveRepository
	condition_repository        *repository.ConditionRepository
	scoring_category_repository *repository.ScoringCategoryRepository
	eventService                *EventService
	poeClient                   *client.PoEClient
}

func NewRecurringJobService(poeClient *client.PoEClient) *RecurringJobService {
	eventService := NewEventService()
	return &RecurringJobService{
		objective_repository:        repository.NewObjectiveRepository(),
		condition_repository:        repository.NewConditionRepository(),
		scoring_category_repository: repository.NewScoringCategoryRepository(),
		eventService:                eventService,
		poeClient:                   poeClient,
	}
}

func (s *RecurringJobService) StartJob(job *RecurringJob) error {
	switch job.JobType {
	case FetchStashChanges:
		return s.FetchStashChanges(job)
	case EvaluateStashChanges:
		return s.EvaluateStashChanges(job)
	// case CalculateScores:
	// 	return s.CalculateScores(job)
	case FetchCharacterData:
		return s.FetchCharacterData(job)
	default:
		return fmt.Errorf("invalid job type")
	}
}

func (s *RecurringJobService) EvaluateStashChanges(job *RecurringJob) error {
	if job.EventId == nil {
		return fmt.Errorf("EventId is required")
	}
	event, err := s.eventService.GetEventById(*job.EventId, "Teams", "Teams.Users")
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), job.EndDate.Sub(time.Now()))
	job.Cancel = cancel
	return StashLoop(ctx, s.poeClient, event)
}

func (s *RecurringJobService) FetchStashChanges(job *RecurringJob) error {
	if job.EventId == nil {
		return fmt.Errorf("EventId is required")
	}
	event, err := s.eventService.GetEventById(*job.EventId)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), job.EndDate.Sub(time.Now()))
	job.Cancel = cancel
	FetchLoop(ctx, event, s.poeClient)
	return nil
}

func (s *RecurringJobService) FetchCharacterData(job *RecurringJob) error {
	return fmt.Errorf("Not implemented")
}
