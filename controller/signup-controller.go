package controller

import (
	"bpl/repository"
	"bpl/service"
	"bpl/utils"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SignupController struct {
	signupService *service.SignupService
	userService   *service.UserService
}

func NewSignupController(db *gorm.DB) *SignupController {
	return &SignupController{
		signupService: service.NewSignupService(db),
		userService:   service.NewUserService(db),
	}
}

func setupSignupController(db *gorm.DB) []RouteInfo {
	e := NewSignupController(db)
	basePath := "/events/:event_id/signups"
	routes := []RouteInfo{
		{Method: "GET", Path: "", HandlerFunc: e.getEventSignupsHandler(), Authenticated: true},
		{Method: "GET", Path: "/self", HandlerFunc: e.getPersonalSignupHandler(), Authenticated: true},
		{Method: "PUT", Path: "/self", HandlerFunc: e.createSignupHandler(), Authenticated: true},
		{Method: "DELETE", Path: "/self", HandlerFunc: e.deleteSignupHandler(), Authenticated: true},
	}
	for i, route := range routes {
		routes[i].Path = basePath + route.Path
	}
	return routes
}

// @Description Fetches an authenticated user's signup for the event
// @Tags signup
// @Produce json
// @Success 200 {object} SignupResponse
// @Router /events/{event_id}/signups/self [get]
func (e *SignupController) getPersonalSignupHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID, err := strconv.Atoi(c.Param("event_id"))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		user, err := e.userService.GetUserFromAuthCookie(c)
		if err != nil {
			c.JSON(401, gin.H{"error": "Not authenticated"})
			return
		}
		signup, err := e.signupService.GetSignupForUser(user.ID, eventID)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(404, gin.H{"error": "Not signed up"})
			} else {
				c.JSON(400, gin.H{"error": err.Error()})
			}
			return
		}
		c.JSON(200, toSignupResponse(signup))
	}
}

// @Description Creates a signup for the authenticated user
// @Tags signup
// @Accept json
// @Produce json
// @Success 201 {object} SignupResponse
// @Router /events/{event_id}/signups/self [put]
func (e *SignupController) createSignupHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID, err := strconv.Atoi(c.Param("event_id"))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		user, err := e.userService.GetUserFromAuthCookie(c)
		if err != nil {
			c.JSON(401, gin.H{"error": "Not authenticated"})
			return
		}
		signup := &repository.Signup{
			UserID:    user.ID,
			EventID:   eventID,
			Timestamp: time.Now(),
		}
		signup, err = e.signupService.CreateSignup(signup)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		c.JSON(201, toSignupResponse(signup))
	}
}

// @Description Deletes the authenticated user's signup for the event
// @Tags signup
// @Produce json
// @Success 204
// @Router /events/{event_id}/signups/self [delete]
func (e *SignupController) deleteSignupHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID, err := strconv.Atoi(c.Param("event_id"))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		user, err := e.userService.GetUserFromAuthCookie(c)
		if err != nil {
			c.JSON(401, gin.H{"error": "Not authenticated"})
			return
		}
		err = e.signupService.RemoveSignup(user.ID, eventID)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{})
	}
}

// @Description Fetches all signups for the event
// @Tags signup
// @Produce json
// @Success 200 {object} map[int][]SignupResponse
// @Router /events/{event_id}/signups [get]
func (e *SignupController) getEventSignupsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID, err := strconv.Atoi(c.Param("event_id"))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		signups, err := e.signupService.GetSignupsForEvent(eventID)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		signupsResponse := make(map[int][]*SignupResponse, 0)
		for teamID, teamSignups := range signups {
			signupsResponse[teamID] = utils.Map(teamSignups, toSignupResponse)
		}
		c.JSON(200, signupsResponse)
	}
}

type SignupResponse struct {
	ID        int                       `json:"id"`
	User      *NonSensitiveUserResponse `json:"user"`
	Timestamp time.Time                 `json:"timestamp"`
}

func toSignupResponse(signup *repository.Signup) *SignupResponse {
	if signup == nil {
		return nil
	}

	return &SignupResponse{
		ID:        signup.ID,
		User:      toNonSensitiveUserResponse(signup.User),
		Timestamp: signup.Timestamp,
	}
}
