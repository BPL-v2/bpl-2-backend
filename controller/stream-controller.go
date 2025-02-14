package controller

import (
	"bpl/service"

	"github.com/gin-gonic/gin"
)

type StreamController struct {
	teamService   *service.TeamService
	oauthService  *service.OauthService
	streamService *service.StreamService
}

func NewStreamController() *StreamController {
	return &StreamController{
		teamService:   service.NewTeamService(),
		oauthService:  service.NewOauthService(),
		streamService: service.NewStreamService(),
	}
}

func setupStreamController() []RouteInfo {
	e := NewStreamController()
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
