package repository

import (
	"bpl/config"
	"time"

	"gorm.io/gorm"
)

type JobType string

const (
	FetchStashChanges    JobType = "FetchStashChanges"
	EvaluateStashChanges JobType = "EvaluateStashChanges"
	FetchCharacterData   JobType = "FetchCharacterData"
	// CalculateScores      JobType = "CalculateScores"
)

type RecurringJob struct {
	JobType                  JobType   `gorm:"primaryKey;not null;unique"`
	EventId                  int       `gorm:"not null"`
	SleepAfterEachRunSeconds int       `gorm:"not null"`
	EndDate                  time.Time `gorm:"not null"`
}

type RecurringJobsRepository struct {
	DB *gorm.DB
}

func NewRecurringJobsRepository() *RecurringJobsRepository {
	return &RecurringJobsRepository{DB: config.DatabaseConnection()}
}

func (r *RecurringJobsRepository) CreateRecurringJob(job *RecurringJob) error {
	r.DB.Delete(&RecurringJob{}, "job_type = ?", job.JobType)
	return r.DB.Create(job).Error
}

func (r *RecurringJobsRepository) GetRecurringJob(jobType JobType, eventId int) (job *RecurringJob, err error) {
	err = r.DB.Where("job_type = ? AND event_id = ?", jobType, eventId).First(&job).Error
	return job, err
}

func (r *RecurringJobsRepository) GetJobsForEvent(eventId int) (jobs []*RecurringJob, err error) {
	err = r.DB.Where("event_id = ?", eventId).Find(&jobs).Error
	return jobs, err
}

func (r *RecurringJobsRepository) GetAllJobs() (jobs []*RecurringJob, err error) {
	err = r.DB.Find(&jobs).Error
	return jobs, err
}
