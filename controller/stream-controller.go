package controller

import (
	"bpl/service"
	"time"

	"github.com/gin-contrib/cache"
	"github.com/gin-contrib/cache/persistence"
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

func setupStreamController(cacheStore *persistence.InMemoryStore) []RouteInfo {
	e := NewStreamController()
	basePath := "events/:event_id/streams"
	routes := []RouteInfo{
		{Method: "GET", Path: "", HandlerFunc: cache.CachePage(cacheStore, 5*time.Minute, e.getStreamsHandler())},
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
// @Param event_id path int true "Event ID"
// @Success 200 {array} client.TwitchStream
// @Router /events/{event_id}/streams [get]
func (e *StreamController) getStreamsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		event := getEvent(c)
		if event == nil {
			return
		}
		streams, err := e.streamService.GetStreamsForEvent(event.Id)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, streams)
	}
}
