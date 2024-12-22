package controller

import (
	"bpl/repository"
	"bpl/service"
	"bpl/utils"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type EventController struct {
	eventService *service.EventService
}

func NewEventController(db *gorm.DB) *EventController {
	return &EventController{
		eventService: service.NewEventService(db),
	}
}

func setupEventController(db *gorm.DB) []RouteInfo {
	e := NewEventController(db)
	basePath := "/events"
	routes := []RouteInfo{
		{Method: "GET", Path: "", HandlerFunc: e.getEventsHandler()},
		{Method: "PUT", Path: "", HandlerFunc: e.createEventHandler(), Authenticated: true, RequiredRoles: []repository.Permission{repository.PermissionAdmin}},
		{Method: "GET", Path: "/current", HandlerFunc: e.getCurrentEventHandler()},
		{Method: "GET", Path: "/:event_id", HandlerFunc: e.getEventHandler()},
		{Method: "DELETE", Path: "/:event_id", HandlerFunc: e.deleteEventHandler(), Authenticated: true, RequiredRoles: []repository.Permission{repository.PermissionAdmin}},
	}
	for i, route := range routes {
		routes[i].Path = basePath + route.Path
	}
	return routes
}

// @Description Fetches all events
// @Tags event
// @Produce json
// @Success 200 {array} EventResponse
// @Router /events [get]
func (e *EventController) getEventsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		events, err := e.eventService.GetAllEvents()
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, utils.Map(events, toEventResponse))
	}
}

// @Description Fetches the current event
// @Tags event
// @Produce json
// @Success 200 {object} EventResponse
// @Router /events/current [get]
func (e *EventController) getCurrentEventHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		event, err := e.eventService.GetCurrentEvent("Teams")
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, toEventResponse(*event))
	}
}

// @Description Creates or updates an event
// @Tags event
// @Accept json
// @Produce json
// @Param event body EventCreate true "Event to create"
// @Success 201 {object} EventResponse
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
		c.JSON(201, toEventResponse(*dbevent))
	}
}

// @Description Gets an event by id
// @Tags event
// @Accept json
// @Produce json
// @Param eventId path int true "Event ID"
// @Success 201 {object} EventResponse
// @Router /events/{eventId} [get]
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
		c.JSON(200, toEventResponse(*event))
	}
}

// @Description Deletes an event
// @Tags event
// @Param eventId path int true "Event ID"
// @Success 204
// @Router /events/{eventId} [delete]
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

type EventCreate struct {
	ID        *int   `json:"id"`
	Name      string `json:"name" binding:"required"`
	IsCurrent bool   `json:"is_current"`
	MaxSize   int    `json:"max_size"`
}

type EventResponse struct {
	ID                int    `json:"id"`
	Name              string `json:"name"`
	ScoringCategoryID int    `json:"scoring_category_id"`
	IsCurrent         bool   `json:"is_current"`
	MaxSize           int    `json:"max_size"`
}

func (e *EventCreate) toModel() *repository.Event {
	event := &repository.Event{
		Name:      e.Name,
		IsCurrent: e.IsCurrent,
		MaxSize:   e.MaxSize,
	}
	if e.ID != nil {
		event.ID = *e.ID
	}
	return event
}

func toEventResponse(event repository.Event) EventResponse {
	return EventResponse{
		ID:                event.ID,
		Name:              event.Name,
		ScoringCategoryID: event.ScoringCategoryID,
		IsCurrent:         event.IsCurrent,
		MaxSize:           event.MaxSize,
	}
}
