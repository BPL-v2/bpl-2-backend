package controller

import (
	"bpl/repository"
	"bpl/service"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type ActivityController struct {
	activityService *service.ActivityService
	userService     *service.UserService
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
		{Method: "GET", Path: "/:user_id", HandlerFunc: e.getEventActivitiesForUserHandler(), Authenticated: true},
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
			c.JSON(500, gin.H{"error": err.Error()})
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
		user, err := e.userService.GetUserFromAuthHeader(c)
		if err != nil {
			c.JSON(401, gin.H{"error": "Not authenticated"})
			return
		}
		if user.Id != userId {
			c.JSON(403, gin.H{"error": "You are not allowed to view activities of other users"})
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
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, activity.Milliseconds())
	}
}
