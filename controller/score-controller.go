package controller

import (
	"bpl/client"
	"bpl/repository"
	"bpl/scoring"
	"bpl/service"
	"bpl/utils"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type ScoreController struct {
	eventService      *service.EventService
	scoreService      *service.ScoreService
	userService       *service.UserService
	mu                sync.Mutex
	connections       map[int]map[*websocket.Conn]int
	simpleConnections map[int]map[*websocket.Conn]int
}

func NewScoreController(PoEClient *client.PoEClient) *ScoreController {
	eventService := service.NewEventService()
	controller := &ScoreController{
		eventService:      eventService,
		scoreService:      service.NewScoreService(PoEClient),
		userService:       service.NewUserService(),
		connections:       make(map[int]map[*websocket.Conn]int),
		simpleConnections: make(map[int]map[*websocket.Conn]int),
	}
	controller.StartScoreUpdater()
	return controller
}

func setupScoreController(PoEClient *client.PoEClient) []RouteInfo {
	e := NewScoreController(PoEClient)
	baseUrl := "events/:event_id/scores"
	routes := []RouteInfo{
		{Method: "GET", Path: "/latest", HandlerFunc: e.getLatestScoresForEventHandler()},
		{Method: "GET", Path: "/ws", HandlerFunc: e.WebSocketHandler},
		{Method: "GET", Path: "/simple/ws", HandlerFunc: e.SimpleWebSocketHandler},
	}
	for i, route := range routes {
		routes[i].Path = baseUrl + route.Path
	}
	return routes
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// allow any host origin to connect to the websocket
		return true
	},
}

// @id ScoreWebSocket
// @Description Websocket for score updates. Once connected, the client will receive score updates in real-time.
// @Tags scores
// @Router /events/{event_id}/scores/ws [get]
// @Param event_id path int true "Event Id"
// @Security BearerAuth
// @in header
// @name Authorization
// @Success 200 {object} ScoreDiff
func (e *ScoreController) WebSocketHandler(c *gin.Context) {
	c.Request.Header.Set("Authorization", "Bearer "+c.Request.URL.Query().Get("token"))
	event := getEvent(c)
	if event == nil {
		return
	}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		http.NotFound(c.Writer, c.Request)
		return
	}
	defer conn.Close()

	teamId := 0
	teamUser, _, err := e.userService.GetTeamForUser(c, event)
	if err == nil {
		teamId = teamUser.TeamId
	}

	if _, ok := e.scoreService.LatestScores[event.Id]; !ok {
		e.scoreService.LatestScores[event.Id] = make(service.ScoreMap)
	}

	e.mu.Lock()
	if _, ok := e.connections[event.Id]; !ok {
		e.connections[event.Id] = make(map[*websocket.Conn]int)
	}
	e.connections[event.Id][conn] = teamId
	e.mu.Unlock()

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			e.mu.Lock()
			delete(e.connections[event.Id], conn)
			if len(e.connections[event.Id]) == 0 {
				delete(e.connections, event.Id)
			}
			e.mu.Unlock()
			return
		}
	}
}

// @id SimpleScoreWebSocket
// @Description Websocket for simple score updates.
// @Tags scores
// @Router /events/{event_id}/scores/simple/ws [get]
// @Param event_id path int true "Event Id"
// @Security BearerAuth
// @Success 200 {object} map[int]int
func (e *ScoreController) SimpleWebSocketHandler(c *gin.Context) {

	event := getEvent(c)
	if event == nil {
		return
	}
	teamId := 0
	teamUser, _, err := e.userService.GetTeamForUser(c, event)
	if err == nil {
		teamId = teamUser.TeamId
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		http.NotFound(c.Writer, c.Request)
		return
	}
	defer conn.Close()

	serialized, err := json.Marshal(e.scoreService.LatestScores[event.Id].GetSimpleScore())
	if err != nil {
		return
	}

	if err := conn.WriteMessage(websocket.TextMessage, serialized); err != nil {
		return
	}

	e.mu.Lock()
	if _, ok := e.simpleConnections[event.Id]; !ok {
		e.simpleConnections[event.Id] = make(map[*websocket.Conn]int)
	}
	e.simpleConnections[event.Id][conn] = teamId
	e.mu.Unlock()

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			e.mu.Lock()
			delete(e.simpleConnections[event.Id], conn)
			if len(e.simpleConnections[event.Id]) == 0 {
				delete(e.simpleConnections, event.Id)
			}
			e.mu.Unlock()
			return
		}
	}
}

func (e *ScoreController) StartScoreUpdater() {
	go func() {
		events, err := e.eventService.GetAllEvents()
		if err != nil {
			fmt.Println("Error fetching events for score updater:", err)
			return
		}
		eventMap := make(map[int]*repository.Event)
		for _, event := range events {
			eventMap[event.Id] = event
		}

		for {
			e.mu.Lock()
			// calculate scores for events with active websocket connections
			eventIds := utils.Keys(e.connections)
			eventIds = utils.Uniques(append(eventIds, utils.Keys(e.simpleConnections)...))
			e.mu.Unlock()
			for _, eventId := range eventIds {
				event, ok := eventMap[eventId]
				// dont update event if its over and its already cached
				if !ok || (event.EventEndTime.Before(time.Now()) && e.scoreService.LatestScores[eventId] != nil) {
					continue
				}

				diff, err := e.scoreService.GetNewDiff(eventId)
				if err != nil {
					continue
				}
				simpleScore, err := json.Marshal(e.scoreService.LatestScores[eventId].GetSimpleScore())
				if err != nil {
					log.Fatal(err)
					continue
				}

				e.mu.Lock()
				for conn, teamId := range e.connections[eventId] {
					serializedDiff, err := json.Marshal(toScoreMapResponse(diff, teamId))
					if err != nil {
						log.Fatal(err)
						continue
					}
					if err := conn.WriteMessage(websocket.TextMessage, serializedDiff); err != nil {
						conn.Close()
						delete(e.connections[eventId], conn)
					}
				}
				for conn := range e.simpleConnections[eventId] {
					if err := conn.WriteMessage(websocket.TextMessage, simpleScore); err != nil {
						conn.Close()
						delete(e.simpleConnections[eventId], conn)
					}
				}
				e.mu.Unlock()
			}
			time.Sleep(5 * time.Second)
		}
	}()
}

// @id GetLatestScoresForEvent
// @Description Fetches the latest scores for the current event
// @Tags scores
// @Produce json
// @Security BearerAuth
// @Success 200 {array} ScoreDiff
// @Param event_id path int true "Event Id"
// @Router /events/{event_id}/scores/latest [get]
func (e *ScoreController) getLatestScoresForEventHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		event := getEvent(c)
		if event == nil {
			return
		}
		teamId := 0
		teamUser, _, err := e.userService.GetTeamForUser(c, event)
		if err == nil {
			teamId = teamUser.TeamId
		}
		scores, err := e.scoreService.GetCurrentScore(event.Id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "No scores found for event"})
			return
		}
		c.JSON(http.StatusOK, toScoreMapResponse(scores, teamId))
	}
}

type Score struct {
	Points    int   `json:"points" binding:"required"`
	UserId    *int  `json:"user_id,omitempty"`
	Rank      int   `json:"rank" binding:"required"`
	Timestamp int64 `json:"timestamp" binding:"required"`
	Number    int   `json:"number" binding:"required"`
	Finished  bool  `json:"finished" binding:"required"`
}

type ScoreDiff struct {
	ObjectiveId int              `json:"objective_id" binding:"required"`
	TeamId      int              `json:"team_id" binding:"required"`
	Score       *Score           `json:"score" binding:"required"`
	FieldDiff   []string         `json:"field_diff,omitempty" binding:"required"`
	DiffType    service.Difftype `json:"diff_type" binding:"required"`
}

func toScoreDiffResponse(scoreDiff *service.ScoreDifference) *ScoreDiff {
	return &ScoreDiff{
		Score:       toScoreResponse(scoreDiff.Score),
		FieldDiff:   scoreDiff.FieldDiff,
		DiffType:    scoreDiff.DiffType,
		ObjectiveId: scoreDiff.Score.Id,
		TeamId:      scoreDiff.Score.TeamId,
	}
}

func toScoreMapResponse(scoreMap service.ScoreMap, teamId int) []*ScoreDiff {
	response := make([]*ScoreDiff, 0)
	for _, teamScores := range scoreMap {
		for _, scoreDiff := range teamScores {
			if scoreDiff.Score.CanShowTo(teamId) {
				response = append(response, toScoreDiffResponse(scoreDiff))
			}
		}
	}
	return response
}

func toScoreResponse(score *scoring.Score) *Score {
	resp := &Score{
		Finished:  score.Finished,
		Timestamp: score.Timestamp.Unix(),
		Number:    score.Number,
		Points:    score.Points,
		Rank:      score.Rank,
	}
	if score.UserId != 0 {
		resp.UserId = &score.UserId
	}
	return resp
}
