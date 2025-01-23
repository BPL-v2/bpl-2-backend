package controller

import (
	"bpl/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type StreamController struct {
	teamService   *service.TeamService
	oauthService  *service.OauthService
	streamService *service.StreamService
}

func NewStreamController(db *gorm.DB) *StreamController {
	return &StreamController{
		teamService:   service.NewTeamService(db),
		oauthService:  service.NewOauthService(db),
		streamService: service.NewStreamService(db),
	}
}

func setupStreamController(db *gorm.DB) []RouteInfo {
	e := NewStreamController(db)
	basePath := "/streams"
	routes := []RouteInfo{
		{Method: "GET", Path: "", HandlerFunc: e.getStreamsHandler()},
	}
	for i, route := range routes {
		routes[i].Path = basePath + route.Path
	}
	return routes
}

// @id GetStreams
// @Description Fetches all twitch streams for the current event
// @Tags streams
// @Produce json
// @Success 200 {array} client.Stream
// @Router /streams [get]
func (e *StreamController) getStreamsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		streams, err := e.streamService.GetStreamsForCurrentEvent()
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, streams)
	}
}
