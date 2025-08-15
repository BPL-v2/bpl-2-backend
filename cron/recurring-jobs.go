package cron

import (
	"bpl/client"
	"bpl/repository"
	"bpl/service"
	"context"
	"fmt"
	"log"
	"time"
)

type RecurringJob struct {
	JobType                  repository.JobType `json:"job_type" binding:"required"`
	SleepAfterEachRunSeconds int                `json:"sleep_after_each_run_seconds" binding:"required"`
	Cancel                   context.CancelFunc `json:"-"`
	EndDate                  time.Time          `json:"end_date" binding:"required"`
	EventId                  int                `json:"event_id" binding:"required"`
}

type RecurringJobService struct {
	objectiveRepository *repository.ObjectiveRepository
	conditionRepository *repository.ConditionRepository
	eventService        *service.EventService
	poeClient           *client.PoEClient
	jobRepository       *repository.RecurringJobsRepository
	Jobs                map[repository.JobType]*RecurringJob
}

func NewRecurringJobService(poeClient *client.PoEClient) *RecurringJobService {
	eventService := service.NewEventService()

	s := &RecurringJobService{
		objectiveRepository: repository.NewObjectiveRepository(),
		conditionRepository: repository.NewConditionRepository(),
		jobRepository:       repository.NewRecurringJobsRepository(),
		eventService:        eventService,
		poeClient:           poeClient,
		Jobs:                make(map[repository.JobType]*RecurringJob),
	}

	jobs, err := s.InitializeJobs()
	if err != nil {
		log.Fatal(err)
	}
	s.Jobs = jobs
	return s
}

func (s *RecurringJobService) InitializeJobs() (map[repository.JobType]*RecurringJob, error) {
	jobs := make(map[repository.JobType]*RecurringJob)
	repoJobs, err := s.jobRepository.GetAllJobs()
	if err != nil {
		return jobs, err
	}
	for _, job := range repoJobs {
		jobs[job.JobType] = &RecurringJob{
			JobType:                  job.JobType,
			SleepAfterEachRunSeconds: job.SleepAfterEachRunSeconds,
			EndDate:                  job.EndDate,
			EventId:                  job.EventId,
		}
		serviceJob := jobs[job.JobType]
		if job.EndDate.Before(time.Now()) {
			continue
		}
		err := s.StartJob(serviceJob)
		if err != nil {
			fmt.Println(err)
			if serviceJob.Cancel != nil {
				serviceJob.Cancel()
			}
			jobs[job.JobType] = nil
		}
	}
	return jobs, nil
}

func (s *RecurringJobService) StartJob(job *RecurringJob) error {
	existingJob, ok := s.Jobs[job.JobType]
	if ok {
		if existingJob.Cancel != nil {
			existingJob.Cancel()
		}
	}
	err := s.jobRepository.CreateRecurringJob(&repository.RecurringJob{
		JobType:                  job.JobType,
		SleepAfterEachRunSeconds: job.SleepAfterEachRunSeconds,
		EndDate:                  job.EndDate,
		EventId:                  job.EventId,
	})
	if err != nil {
		return err
	}
	s.Jobs[job.JobType] = job

	switch job.JobType {
	case repository.FetchStashChanges:
		return s.FetchStashChanges(job)
	case repository.EvaluateStashChanges:
		return s.EvaluateStashChanges(job)
	// case CalculateScores:
	// 	return s.CalculateScores(job)
	case repository.FetchCharacterData:
		return s.FetchCharacterData(job)
	case repository.FetchGuildStashes:
		return s.FetchGuildStashes(job)
	default:
		return fmt.Errorf("invalid job type")
	}
}

func (s *RecurringJobService) EvaluateStashChanges(job *RecurringJob) error {

	event, err := s.eventService.GetEventById(job.EventId, "Teams")
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Until(job.EndDate))
	job.Cancel = cancel
	return StashEvaluationLoop(ctx, s.poeClient, event)
}

func (s *RecurringJobService) FetchStashChanges(job *RecurringJob) error {
	event, err := s.eventService.GetEventById(job.EventId)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Until(job.EndDate))
	job.Cancel = cancel
	ItemFetchLoop(ctx, event, s.poeClient)
	return nil
}

func (s *RecurringJobService) FetchCharacterData(job *RecurringJob) error {
	event, err := s.eventService.GetEventById(job.EventId, "Teams")
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Until(job.EndDate))
	job.Cancel = cancel
	go PlayerFetchLoop(ctx, event, s.poeClient)
	go PlayerStatsLoop(ctx, event)
	return nil
}

func (s *RecurringJobService) FetchGuildStashes(job *RecurringJob) error {
	event, err := s.eventService.GetEventById(job.EventId, "Teams")
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Until(job.EndDate))
	job.Cancel = cancel
	go GuildStashFetchLoop(ctx, event, s.poeClient)
	return nil
}
