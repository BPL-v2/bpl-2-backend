package controller

import (
	"bpl/model/restmodel"
	"bpl/service"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GetEventRoutes(db *gorm.DB) []gin.RouteInfo {
	return []gin.RouteInfo{
		{Method: "GET", Path: "/events", HandlerFunc: getEventsHandler(db)},
		{Method: "POST", Path: "/events", HandlerFunc: createEventHandler(db)},
		{Method: "GET", Path: "/events/:id", HandlerFunc: getEventHandler(db)},
	}
}

func getEventsHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		events, err := service.GetAllEvents(db)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, events)
	}
}

func createEventHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var event restmodel.Event
		if err := c.BindJSON(&event); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		dbevent, err := service.CreateEvent(db, event.Name)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(201, dbevent)
	}
}

func getEventHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventId, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		event, err := service.GetEventById(db, eventId)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(404, gin.H{"error": "Event not found"})
			} else {
				c.JSON(500, gin.H{"error": err.Error()})
			}
			return
		}
		c.JSON(200, event)
	}
}
