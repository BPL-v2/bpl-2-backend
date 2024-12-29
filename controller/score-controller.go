package controller

import (
	"bpl/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ScoreController struct {
	service *service.ObjectiveMatchService
}

func NewScoreController(db *gorm.DB) *ScoreController {
	return &ScoreController{service: service.NewObjectiveMatchService(db)}
}

func setupScoreController(db *gorm.DB) []RouteInfo {
	e := NewScoreController(db)
	baseUrl := "events/:eventId/scores"
	routes := []RouteInfo{
		{Method: "GET", Path: "/scores", HandlerFunc: e.getLatestScoresForEventHandler()},
	}
	for i, route := range routes {
		routes[i].Path = baseUrl + route.Path
	}
	return routes
}

func (e *ScoreController) getLatestScoresForEventHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// eventId, err := strconv.Atoi(c.Param("eventId"))
		// if err != nil {
		// 	c.JSON(400, gin.H{"error": err.Error()})
		// 	return
		// }
		c.JSON(404, gin.H{"error": "Not implemented"})
		// scores, err := e.service.GetLatestScoresForEvent(eventId)
		// if err != nil {
		// 	c.JSON(400, gin.H{"error": err.Error()})
		// 	return
		// }
		// c.JSON(200, scores)
	}
}
