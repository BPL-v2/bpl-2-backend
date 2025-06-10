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
	FetchGuildStashes    JobType = "FetchGuildStashes"
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
	r.DB.Delete(&RecurringJob{}, &RecurringJob{JobType: job.JobType})
	return r.DB.Create(job).Error
}

func (r *RecurringJobsRepository) GetAllJobs() (jobs []*RecurringJob, err error) {
	err = r.DB.Find(&jobs).Error
	return jobs, err
}
