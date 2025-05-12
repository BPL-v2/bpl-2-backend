package controller

import (
	"bpl/client"
	"bpl/cron"
	"bpl/repository"
	"bpl/utils"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

type RecurringJobsController struct {
	service *cron.RecurringJobService
}

type JobCreate struct {
	JobType                  repository.JobType `json:"job_type"`
	SleepAfterEachRunSeconds int                `json:"sleep_after_each_run_seconds"`
	EventId                  int                `json:"event_id"`
	DurationInSeconds        *int               `json:"duration_in_seconds"`
	EndDate                  *time.Time         `json:"end_date"`
}

func (j *JobCreate) toJob() (*cron.RecurringJob, error) {
	if !utils.Contains(jobList, j.JobType) {
		return nil, fmt.Errorf("job type does not exist")
	}
	if j.DurationInSeconds != nil && j.EndDate != nil {
		return nil, fmt.Errorf("cannot specify both duration and end date")
	}
	if j.DurationInSeconds == nil && j.EndDate == nil {
		return nil, fmt.Errorf("must specify either duration or end date")
	}
	if j.DurationInSeconds != nil {
		endDate := time.Now().Add(time.Duration(*j.DurationInSeconds) * time.Second)
		j.EndDate = &endDate
	}
	return &cron.RecurringJob{
		JobType:                  j.JobType,
		SleepAfterEachRunSeconds: j.SleepAfterEachRunSeconds,
		EndDate:                  *j.EndDate,
		EventId:                  j.EventId,
	}, nil
}

var jobList = []repository.JobType{
	repository.FetchStashChanges,
	repository.EvaluateStashChanges,
	// service.CalculateScores,
	repository.FetchCharacterData,
}

func NewRecurringJobsController() *RecurringJobsController {
	poeClient := client.NewPoEClient(10, false, 600)
	controller := &RecurringJobsController{
		service: cron.NewRecurringJobService(poeClient),
	}
	// controller.StartScoreUpdater()
	return controller
}

func setupRecurringJobsController() []RouteInfo {
	c := NewRecurringJobsController()
	baseUrl := "jobs"
	routes := []RouteInfo{
		{Method: "GET", Path: "", HandlerFunc: c.getJobsHandler(), Authenticated: true, RequiredRoles: []repository.Permission{repository.PermissionAdmin}},
		{Method: "POST", Path: "", HandlerFunc: c.startJobHandler(), Authenticated: true, RequiredRoles: []repository.Permission{repository.PermissionAdmin}},
	}
	for i, route := range routes {
		routes[i].Path = baseUrl + route.Path
	}
	return routes
}

// @id GetJobs
// @Description Get all recurring jobs
// @Security BearerAuth
// @Tags jobs
// @Accept json
// @Produce json
// @Success 200 {array} cron.RecurringJob
// @Router /jobs [get]
func (c *RecurringJobsController) getJobsHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		jobs := make([]*cron.RecurringJob, 0)
		for _, jobType := range jobList {
			job, ok := c.service.Jobs[jobType]
			if ok {
				jobs = append(jobs, job)
			}
		}
		ctx.JSON(200, jobs)
	}
}

// @id StartJob
// @Description Start a recurring job
// @Security BearerAuth
// @Tags jobs
// @Accept json
// @Produce json
// @Param job body JobCreate true "Job to create"
// @Success 201 {object} cron.RecurringJob
// @Router /jobs [post]
func (c *RecurringJobsController) startJobHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var jobCreate JobCreate
		if err := ctx.BindJSON(&jobCreate); err != nil {
			ctx.JSON(400, gin.H{"error": err.Error()})
			return
		}

		job, err := jobCreate.toJob()
		if err != nil {
			ctx.JSON(400, gin.H{"error": err.Error()})
			return
		}
		err = c.service.StartJob(job)
		if err != nil {
			ctx.JSON(500, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(201, job)
	}
}
