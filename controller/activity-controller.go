package controller

import (
	"bpl/repository"
	"bpl/service"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type ActivityController struct {
	activityService service.ActivityService
	userService     service.UserService
}

func NewActivityController() *ActivityController {
	return &ActivityController{
		activityService: service.NewActivityService(),
		userService:     service.NewUserService(),
	}
}

func setupActivityController() []RouteInfo {
	e := NewActivityController()
	baseUrl := "/events/:event_id/activity"
	routes := []RouteInfo{
		{Method: "GET", Path: "", HandlerFunc: e.getEventActivitiesHandler(), Authenticated: true, RequiredRoles: []repository.Permission{repository.PermissionAdmin}},
		{Method: "GET", Path: "/:user_id", HandlerFunc: e.getEventActivitiesForUserHandler(), Authenticated: true, RequiresUserSelf: true},
		{Method: "GET", Path: "/:user_id/timestamps", HandlerFunc: e.getEventActivityTimestampsForUserHandler(), Authenticated: true, RequiredRoles: []repository.Permission{repository.PermissionAdmin}},
	}
	for i, route := range routes {
		routes[i].Path = baseUrl + route.Path
	}
	return routes
}

// @id GetEventActivities
// @Security BearerAuth
// @Description Get calculated active times for all users in an event
// @Tags activity
// @Produce json
// @Param event_id path int true "Event ID"
// @Param threshold_seconds query int false "Threshold in seconds to consider a user active before and after an activity (default: 300)"
// @Success 200 {object} map[int]int
// @Router /events/{event_id}/activity [get]
func (e *ActivityController) getEventActivitiesHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		event := getEvent(c)
		if event == nil {
			return
		}
		thresholdStr := c.DefaultQuery("threshold_seconds", "1800")
		thresholdSeconds, err := strconv.Atoi(thresholdStr)
		if err != nil || thresholdSeconds <= 0 {
			c.JSON(400, gin.H{"error": "Invalid threshold_seconds"})
			return
		}
		activities, err := e.activityService.CalculateActiveTimesForEvent(event, time.Duration(thresholdSeconds)*time.Second)
		if err != nil {
			respondError(c, err)
			return
		}
		c.JSON(200, activities)
	}
}

// @id GetEventActivitiesForUser
// @Security BearerAuth
// @Description Get calculated active times for a user in an event
// @Tags activity
// @Produce json
// @Param event_id path int true "Event ID"
// @Param user_id path int true "User ID"
// @Param threshold_seconds query int false "Threshold in seconds to consider a user active before and after an activity (default: 300)"
// @Success 200 {object} int
// @Router /events/{event_id}/activity/{user_id} [get]
func (e *ActivityController) getEventActivitiesForUserHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		event := getEvent(c)
		if event == nil {
			return
		}
		userId, err := strconv.Atoi(c.Param("user_id"))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid user id"})
			return
		}
		thresholdStr := c.DefaultQuery("threshold_seconds", "1800")
		thresholdSeconds, err := strconv.Atoi(thresholdStr)
		if err != nil || thresholdSeconds <= 0 {
			c.JSON(400, gin.H{"error": "Invalid threshold_seconds"})
			return
		}
		activity, err := e.activityService.CalculateActiveTime(userId, event, time.Duration(thresholdSeconds)*time.Second)
		if err != nil {
			respondError(c, err)
			return
		}
		c.JSON(200, activity.Milliseconds())
	}
}

// @id GetEventActivityTimestampsForUser
// @Security BearerAuth
// @Description Get the raw activity timestamps of a user in an event
// @Tags activity
// @Produce json
// @Param event_id path int true "Event ID"
// @Param user_id path int true "User ID"
// @Success 200 {array} time.Time
// @Router /events/{event_id}/activity/{user_id}/timestamps [get]
func (e *ActivityController) getEventActivityTimestampsForUserHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		event := getEvent(c)
		if event == nil {
			return
		}
		userId, err := strconv.Atoi(c.Param("user_id"))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid user id"})
			return
		}
		timestamps, err := e.activityService.GetActivityTimestamps(userId, event.Id)
		if err != nil {
			respondError(c, err)
			return
		}
		c.JSON(200, timestamps)
	}
}
