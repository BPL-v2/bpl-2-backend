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

func setupEventController(db *gorm.DB) []gin.RouteInfo {
	e := NewEventController(db)
	basePath := "/events"
	routes := []gin.RouteInfo{
		{Method: "GET", Path: "", HandlerFunc: e.getEventsHandler()},
		{Method: "POST", Path: "", HandlerFunc: e.createEventHandler()},
		{Method: "GET", Path: "/:event_id", HandlerFunc: e.getEventHandler()},
		{Method: "PATCH", Path: "/:event_id", HandlerFunc: e.updateEventHandler()},
		{Method: "DELETE", Path: "/:event_id", HandlerFunc: e.deleteEventHandler()},
	}
	for i, route := range routes {
		routes[i].Path = basePath + route.Path
	}
	return routes
}

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

func (e *EventController) getEventHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		eventId, err := strconv.Atoi(c.Param("event_id"))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		event, err := e.eventService.GetEventById(eventId)
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

func (e *EventController) updateEventHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		eventId, err := strconv.Atoi(c.Param("event_id"))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		var event EventUpdate
		if err := c.BindJSON(&event); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		dbevent, err := e.eventService.UpdateEvent(eventId, event.toModel())
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(404, gin.H{"error": "Event not found"})
			} else {
				c.JSON(500, gin.H{"error": err.Error()})
			}
			return
		}
		c.JSON(200, toEventResponse(*dbevent))
	}
}

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
	Name string `json:"name" binding:"required"`
}

type EventUpdate struct {
	Name string `json:"name"`
}

type EventResponse struct {
	ID                int    `json:"id"`
	Name              string `json:"name"`
	ScoringCategoryID int    `json:"scoring_category_id"`
}

func (e *EventCreate) toModel() *repository.Event {
	return &repository.Event{
		Name: e.Name,
	}
}

func (e *EventUpdate) toModel() *repository.Event {
	return &repository.Event{
		Name: e.Name,
	}
}

func toEventResponse(event repository.Event) EventResponse {
	return EventResponse{
		ID:                event.ID,
		Name:              event.Name,
		ScoringCategoryID: event.ScoringCategoryID,
	}
}
