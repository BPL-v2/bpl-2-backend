package controller

import (
	"bpl/repository"
	"bpl/service"
	"bpl/utils"
	"time"

	"github.com/gin-gonic/gin"
)

type TimingController struct {
	TimingService *service.TimingService
}

func NewTimingController() *TimingController {
	return &TimingController{
		TimingService: service.NewTimingService(),
	}
}

func setupTimingController() []RouteInfo {
	e := NewTimingController()
	baseUrl := "/timings"
	routes := []RouteInfo{
		{Method: "GET", Path: "", HandlerFunc: e.getTimings()},
		{Method: "PUT", Path: "", HandlerFunc: e.setTimings(), Authenticated: true, RequiredRoles: []repository.Permission{repository.PermissionAdmin}}}
	for i, route := range routes {
		routes[i].Path = baseUrl + route.Path
	}
	return routes
}

// @ID getTimings
// @Summary Get timing configurations
// @Description Retrieve the current timing configurations for various operations.
// @Tags Timing
// @Produce json
// @Success 200 {array} Timing
// @Router /timings [get]
func (c *TimingController) getTimings() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		timings, err := c.TimingService.GetTimings()
		if err != nil {
			ctx.JSON(404, gin.H{"error": "Event not found"})
			return
		}
		timingsMs := make(map[repository.TimingKey]int64)
		timingList := make([]Timing, 0, len(timings))
		for key, duration := range timings {
			timingsMs[key] = int64(duration / time.Millisecond)
			timingList = append(timingList, Timing{
				Key:         key,
				DurationMs:  int64(duration / time.Millisecond),
				Description: repository.TimingKeyDescriptions[key],
			})
		}
		ctx.JSON(200, timingList)
	}
}

// @ID setTimings
// @Summary Set timing configurations
// @Description Update the timing configurations for various operations.
// @Security BearerAuth
// @Tags Timing
// @Accept json
// @Produce json
// @Param timings body []TimingCreate true "List of timing configurations"
// @Success 204
// @Router /timings [put]
func (c *TimingController) setTimings() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var timings []TimingCreate
		if err := ctx.ShouldBindJSON(&timings); err != nil {
			ctx.JSON(400, gin.H{"error": "Invalid request body"})
			return
		}
		repoTimings := utils.Map(timings, toTiming)

		err := c.TimingService.SaveTimings(repoTimings)
		if err != nil {
			ctx.JSON(500, gin.H{"error": "Failed to save timings"})
			return
		}
		ctx.JSON(204, nil)
	}
}

type Timing struct {
	Key         repository.TimingKey `json:"key" binding:"required"`
	DurationMs  int64                `json:"duration_ms" binding:"required"`
	Description string               `json:"description,omitempty" binding:"required"`
}

type TimingCreate struct {
	Key        repository.TimingKey `json:"key" binding:"required"`
	DurationMs int64                `json:"duration_ms" binding:"required"`
}

func toTiming(t TimingCreate) *repository.Timing {
	return &repository.Timing{
		Key:        t.Key,
		DurationMs: t.DurationMs,
	}
}
