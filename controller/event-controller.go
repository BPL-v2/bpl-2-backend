package controller

import (
	"bpl/repository"
	"bpl/service"
	"bpl/utils"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type EventController struct {
	eventService           *service.EventService
	teamService            *service.TeamService
	userService            *service.UserService
	signupService          *service.SignupService
	scoringCategoryService *service.ScoringCategoryService
	scoringPresetService   *service.ScoringPresetService
}

func NewEventController() *EventController {
	return &EventController{
		eventService:           service.NewEventService(),
		teamService:            service.NewTeamService(),
		userService:            service.NewUserService(),
		signupService:          service.NewSignupService(),
		scoringCategoryService: service.NewScoringCategoryService(),
		scoringPresetService:   service.NewScoringPresetsService(),
	}
}

func setupEventController() []RouteInfo {
	e := NewEventController()
	basePath := "/events"
	routes := []RouteInfo{
		{Method: "GET", Path: "", HandlerFunc: e.getEventsHandler()},
		{Method: "PUT", Path: "", HandlerFunc: e.createEventHandler(), Authenticated: true, RequiredRoles: []repository.Permission{repository.PermissionAdmin}},

		{Method: "POST", Path: "/:event_id/duplicate", HandlerFunc: e.duplicateEventHandler()},
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
		roles, _ := getUserRoles(c)
		if !utils.Contains(roles, repository.PermissionAdmin) {
			events = utils.Filter(events, func(event *repository.Event) bool {
				return event.Public
			})
		}
		c.JSON(200, utils.Map(events, toEventResponse))
	}
}

// @id CreateEvent
// @Description Creates or updates an event
// @Tags event
// @Accept json
// @Produce json
// @Param event body EventCreate true "Event to create"
// @Success 201 {object} Event
// @Router /events [put]
func (e *EventController) createEventHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var eventCreate EventCreate
		if err := c.BindJSON(&eventCreate); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		dbevent, err := e.eventService.CreateEvent(eventCreate.toModel())
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(404, gin.H{"error": "Event not found"})
			} else {
				c.JSON(500, gin.H{"error": err.Error()})
			}
			return
		}
		c.JSON(201, toEventResponse(dbevent))
	}
}

// @id DeleteEvent
// @Description Deletes an event
// @Tags event
// @Param event_id path int true "Event Id"
// @Success 204
// @Router /events/{event_id} [delete]
func (e *EventController) deleteEventHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		event := getEvent(c)
		if event == nil {
			return
		}
		if event.Locked {
			c.JSON(400, gin.H{"error": "Event is locked"})
			return
		}
		err := e.eventService.DeleteEvent(event)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
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
// @Param event_id path int true "Event Id"
// @Success 200 {object} EventStatus
// @Router /events/{event_id}/status [get]
func (e *EventController) getEventStatusForUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		event := getEvent(c)
		if event == nil {
			return
		}
		user, err := e.userService.GetUserFromAuthCookie(c)
		if err != nil {
			c.JSON(401, gin.H{"error": "Not authenticated"})
			return
		}
		response := EventStatus{}

		team, err := e.teamService.GetTeamForUser(event.Id, user.Id)
		if err != nil && err != gorm.ErrRecordNotFound {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		if team != nil {
			response.TeamId = &team.Id
			response.ApplicationStatus = ApplicationStatusAccepted
		} else {

			signup, _ := e.signupService.GetSignupForUser(user.Id, event.Id)
			if signup != nil {
				response.ApplicationStatus = ApplicationStatusApplied
			} else {
				response.ApplicationStatus = ApplicationStatusNone
			}

		}
		c.JSON(200, response)
	}
}

// @id DuplicateEvent
// @Description Duplicates an event's configuration
// @Tags event
// @Accept json
// @Produce json
// @Param event_id path int true "Event Id"
// @Param event body EventCreate true "Event to create"
// @Success 201 {object} Event
// @Router /events/{event_id}/duplicate [post]
func (e *EventController) duplicateEventHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		oldEvent := getEvent(c)
		if oldEvent == nil {
			return
		}
		var eventCreate EventCreate
		if err := c.BindJSON(&eventCreate); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		eventCreate.Id = nil
		event := eventCreate.toModel()
		event, err := e.eventService.CreateEvent(event)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		presetIdMap, err := e.scoringPresetService.DuplicatePresets(oldEvent.Id, event.Id)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		_, err = e.scoringCategoryService.DuplicateScoringCategories(oldEvent.Id, presetIdMap)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(201, toEventResponse(event))
	}
}

type EventCreate struct {
	Id                   *int                   `json:"id"`
	Name                 string                 `json:"name" binding:"required"`
	IsCurrent            bool                   `json:"is_current"`
	GameVersion          repository.GameVersion `json:"game_version" binding:"required"`
	MaxSize              int                    `json:"max_size" binding:"required"`
	EventStartTime       time.Time              `json:"event_start_time" binding:"required"`
	EventEndTime         time.Time              `json:"event_end_time" binding:"required"`
	ApplicationStartTime time.Time              `json:"application_start_time" binding:"required"`
	Public               bool                   `json:"is_public"`
	Locked               bool                   `json:"is_locked"`
}

type Event struct {
	Id                   int                    `json:"id" binding:"required"`
	Name                 string                 `json:"name" binding:"required"`
	IsCurrent            bool                   `json:"is_current" binding:"required"`
	GameVersion          repository.GameVersion `json:"game_version" binding:"required"`
	MaxSize              int                    `json:"max_size" binding:"required"`
	Teams                []*Team                `json:"teams" binding:"required"`
	ApplicationStartTime time.Time              `json:"application_start_time" binding:"required"`
	EventStartTime       time.Time              `json:"event_start_time" binding:"required"`
	EventEndTime         time.Time              `json:"event_end_time" binding:"required"`
	Public               bool                   `json:"is_public" binding:"required"`
	Locked               bool                   `json:"is_locked" binding:"required"`
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
		Public:               e.Public,
		Locked:               e.Locked,
	}
	if e.Id != nil {
		event.Id = *e.Id
	}
	return event
}

func toEventResponse(event *repository.Event) *Event {
	if event == nil {
		return nil
	}
	return &Event{
		Id:                   event.Id,
		Name:                 event.Name,
		GameVersion:          event.GameVersion,
		IsCurrent:            event.IsCurrent,
		MaxSize:              event.MaxSize,
		Teams:                utils.Map(event.Teams, toTeamResponse),
		ApplicationStartTime: event.ApplicationStartTime,
		EventStartTime:       event.EventStartTime,
		EventEndTime:         event.EventEndTime,
		Public:               event.Public,
		Locked:               event.Locked,
	}
}

type EventStatus struct {
	TeamId            *int              `json:"team_id"`
	ApplicationStatus ApplicationStatus `json:"application_status" binding:"required"`
}

type ApplicationStatus string

const (
	ApplicationStatusApplied    ApplicationStatus = "applied"
	ApplicationStatusAccepted   ApplicationStatus = "accepted"
	ApplicationStatusWaitlisted ApplicationStatus = "waitlisted"
	ApplicationStatusNone       ApplicationStatus = "none"
)
