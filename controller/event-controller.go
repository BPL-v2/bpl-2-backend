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

type EventController struct {
	eventService  *service.EventService
	teamService   *service.TeamService
	userService   *service.UserService
	signupService *service.SignupService
}

func NewEventController() *EventController {
	return &EventController{
		eventService:  service.NewEventService(),
		teamService:   service.NewTeamService(),
		userService:   service.NewUserService(),
		signupService: service.NewSignupService(),
	}
}

func setupEventController() []RouteInfo {
	e := NewEventController()
	basePath := "/events"
	routes := []RouteInfo{
		{Method: "GET", Path: "", HandlerFunc: e.getEventsHandler()},
		{Method: "PUT", Path: "", HandlerFunc: e.createEventHandler(), Authenticated: true, RequiredRoles: []repository.Permission{repository.PermissionAdmin}},
		{Method: "GET", Path: "/current", HandlerFunc: e.getCurrentEventHandler()},

		{Method: "GET", Path: "/:event_id", HandlerFunc: e.getEventHandler()},
		{Method: "GET", Path: "/:event_id/status", HandlerFunc: e.getEventStatusForUser(), Authenticated: true},
		{Method: "DELETE", Path: "/:event_id", HandlerFunc: e.deleteEventHandler(), Authenticated: true, RequiredRoles: []repository.Permission{repository.PermissionAdmin}},
	}
	for i, route := range routes {
		routes[i].Path = basePath + route.Path
	}
	return routes
}

// @id GetEvents
// @Description Fetches all events
// @Tags event
// @Produce json
// @Success 200 {array} Event
// @Router /events [get]
func (e *EventController) getEventsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		events, err := e.eventService.GetAllEvents("Teams")
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, utils.Map(events, toEventResponse))
	}
}

// @id GetCurrentEvent
// @Description Fetches the current event
// @Tags event
// @Produce json
// @Success 200 {object} Event
// @Router /events/current [get]
func (e *EventController) getCurrentEventHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		event, err := e.eventService.GetCurrentEvent("Teams")
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, toEventResponse(event))
	}
}

// @id CreateEvent
// @Description Creates or updates an event
// @Tags event
// @Accept json
// @Produce json
// @Param event body EventCreate true "Event to create"
// @Success 201 {object} Event
// @Router /events [post]
func (e *EventController) createEventHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var eventCreate EventCreate
		if err := c.BindJSON(&eventCreate); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		dbevent, err := e.eventService.CreateEvent(eventCreate.toModel())
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(201, toEventResponse(dbevent))
	}
}

// @id GetEvent
// @Description Gets an event by id
// @Tags event
// @Accept json
// @Produce json
// @Param event_id path int true "Event ID"
// @Success 201 {object} Event
// @Router /events/{event_id} [get]
func (e *EventController) getEventHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		eventId, err := strconv.Atoi(c.Param("event_id"))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		event, err := e.eventService.GetEventById(eventId, "Teams")
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(404, gin.H{"error": "Event not found"})
			} else {
				c.JSON(500, gin.H{"error": err.Error()})
			}
			return
		}
		c.JSON(200, toEventResponse(event))
	}
}

// @id DeleteEvent
// @Description Deletes an event
// @Tags event
// @Param event_id path int true "Event ID"
// @Success 204
// @Router /events/{event_id} [delete]
func (e *EventController) deleteEventHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		eventId, err := strconv.Atoi(c.Param("event_id"))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		err = e.eventService.DeleteEvent(eventId)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(404, gin.H{"error": "Event not found"})
			} else {
				c.JSON(500, gin.H{"error": err.Error()})
			}
			return
		}
		c.JSON(204, nil)
	}
}

// @id GetEventStatusForUser
// @Description Gets the users application status for an event
// @Tags event
// @Accept json
// @Produce json
// @Param event_id path int true "Event ID"
// @Success 200 {object} EventStatus
// @Router /events/{event_id}/status [get]
func (e *EventController) getEventStatusForUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		eventId, err := strconv.Atoi(c.Param("event_id"))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		user, err := e.userService.GetUserFromAuthCookie(c)
		if err != nil {
			c.JSON(401, gin.H{"error": "Not authenticated"})
			return
		}
		response := EventStatus{}

		team, err := e.teamService.GetTeamForUser(eventId, user.ID)
		if err != nil && err != gorm.ErrRecordNotFound {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		if team != nil {
			response.TeamID = &team.ID
			response.ApplicationStatus = ApplicationStatusAccepted
		} else {

			signup, _ := e.signupService.GetSignupForUser(user.ID, eventId)
			if signup != nil {
				response.ApplicationStatus = ApplicationStatusApplied
			} else {
				response.ApplicationStatus = ApplicationStatusNone
			}

		}
		c.JSON(200, response)
	}
}

type EventCreate struct {
	ID                   *int                   `json:"id"`
	Name                 string                 `json:"name" binding:"required"`
	IsCurrent            bool                   `json:"is_current"`
	GameVersion          repository.GameVersion `json:"game_version" binding:"required"`
	MaxSize              int                    `json:"max_size" binding:"required"`
	EventStartTime       time.Time              `json:"event_start_time" binding:"required"`
	EventEndTime         time.Time              `json:"event_end_time" binding:"required"`
	ApplicationStartTime time.Time              `json:"application_start_time" binding:"required"`
}

type Event struct {
	ID                   int                    `json:"id" binding:"required"`
	Name                 string                 `json:"name" binding:"required"`
	ScoringCategoryID    int                    `json:"scoring_category_id" binding:"required"`
	IsCurrent            bool                   `json:"is_current" binding:"required"`
	GameVersion          repository.GameVersion `json:"game_version" binding:"required"`
	MaxSize              int                    `json:"max_size" binding:"required"`
	Teams                []*Team                `json:"teams" binding:"required"`
	ApplicationStartTime time.Time              `json:"application_start_time" binding:"required"`
	EventStartTime       time.Time              `json:"event_start_time" binding:"required"`
	EventEndTime         time.Time              `json:"event_end_time" binding:"required"`
}

func (e *EventCreate) toModel() *repository.Event {
	event := &repository.Event{
		Name:                 e.Name,
		IsCurrent:            e.IsCurrent,
		GameVersion:          e.GameVersion,
		MaxSize:              e.MaxSize,
		EventStartTime:       e.EventStartTime,
		EventEndTime:         e.EventEndTime,
		ApplicationStartTime: e.ApplicationStartTime,
	}
	if e.ID != nil {
		event.ID = *e.ID
	}
	return event
}

func toEventResponse(event *repository.Event) *Event {
	if event == nil {
		return nil
	}
	return &Event{
		ID:                   event.ID,
		Name:                 event.Name,
		ScoringCategoryID:    event.ScoringCategoryID,
		GameVersion:          event.GameVersion,
		IsCurrent:            event.IsCurrent,
		MaxSize:              event.MaxSize,
		Teams:                utils.Map(event.Teams, toTeamResponse),
		ApplicationStartTime: event.ApplicationStartTime,
		EventStartTime:       event.EventStartTime,
		EventEndTime:         event.EventEndTime,
	}
}

type EventStatus struct {
	TeamID            *int              `json:"team_id"`
	ApplicationStatus ApplicationStatus `json:"application_status" binding:"required"`
}

type ApplicationStatus string

const (
	ApplicationStatusApplied    ApplicationStatus = "applied"
	ApplicationStatusAccepted   ApplicationStatus = "accepted"
	ApplicationStatusWaitlisted ApplicationStatus = "waitlisted"
	ApplicationStatusNone       ApplicationStatus = "none"
)
