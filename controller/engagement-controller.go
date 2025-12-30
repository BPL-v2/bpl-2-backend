package controller

import (
	"bpl/service"

	"github.com/gin-gonic/gin"
)

type EngagementController struct {
	engagementService *service.EngagementService
}

func NewEngagementController() *EngagementController {
	return &EngagementController{
		engagementService: service.NewEngagementService(),
	}
}

func setupEngagementController() []RouteInfo {
	e := NewEngagementController()
	baseUrl := "/engagement"
	routes := []RouteInfo{
		{Method: "POST", Path: "", HandlerFunc: e.addEngagementHandler()},
	}
	for i, route := range routes {
		routes[i].Path = baseUrl + route.Path
	}
	return routes
}

type EngagementAdd struct {
	Name string `json:"name" binding:"required"`
}

// @id AddEngagement
// @Security BearerAuth
// @Description Add a new engagement or increment existing engagement number
// @Tags engagement
// @Accept json
// @Param engagement body EngagementAdd true "Engagement to add"
// @Success 204 "No Content"
// @Router /engagement [post]
func (e *EngagementController) addEngagementHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var engagementAdd EngagementAdd
		if err := c.ShouldBindJSON(&engagementAdd); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request body"})
			return
		}
		err := e.engagementService.AddEngagement(engagementAdd.Name)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(204, nil)
	}
}
