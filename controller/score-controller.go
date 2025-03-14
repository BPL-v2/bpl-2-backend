package controller

import (
	"bpl/scoring"
	"bpl/service"
	"bpl/utils"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type ScoreController struct {
	scoringCategoryService *service.ScoringCategoryService
	eventService           *service.EventService
	scoreService           *service.ScoreService
	mu                     sync.Mutex
	connections            map[int]map[*websocket.Conn]bool
}

func NewScoreController() *ScoreController {
	scoringCategoryService := service.NewScoringCategoryService()
	eventService := service.NewEventService()
	controller := &ScoreController{
		scoringCategoryService: scoringCategoryService,
		eventService:           eventService,
		scoreService:           service.NewScoreService(),
		connections:            make(map[int]map[*websocket.Conn]bool),
	}
	controller.StartScoreUpdater()
	return controller
}

func setupScoreController() []RouteInfo {
	e := NewScoreController()
	baseUrl := "events/:event_id/scores"
	routes := []RouteInfo{
		{Method: "GET", Path: "/latest", HandlerFunc: e.getLatestScoresForEventHandler()},
		{Method: "GET", Path: "/ws", HandlerFunc: e.WebSocketHandler},
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
// @Security ApiKeyAuth
// @Success 200 {object} ScoreDiff
func (e *ScoreController) WebSocketHandler(c *gin.Context) {
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

	if _, ok := e.scoreService.LatestScores[event.Id]; !ok {
		e.scoreService.LatestScores[event.Id] = make(service.ScoreMap)
	}
	// Send the latest score to the new subscriber
	serialized, err := json.Marshal(toScoreMapResponse(e.scoreService.LatestScores[event.Id]))
	if err != nil {
		return
	}

	if err := conn.WriteMessage(websocket.TextMessage, serialized); err != nil {
		return
	}

	e.mu.Lock()
	if _, ok := e.connections[event.Id]; !ok {
		e.connections[event.Id] = make(map[*websocket.Conn]bool)
	}
	e.connections[event.Id][conn] = true
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

func (e *ScoreController) StartScoreUpdater() {
	go func() {
		for {
			e.mu.Lock()
			// calculate scores for events with active websocket connections
			for eventId, conns := range e.connections {
				if len(utils.Values(conns)) == 0 {
					continue
				}
				diff, err := e.scoreService.GetNewDiff(eventId)
				if err != nil {
					continue
				}
				serializedDiff, err := json.Marshal(toScoreMapResponse(diff))
				if err != nil {
					log.Fatal(err)
					continue
				}
				for conn := range conns {
					if err := conn.WriteMessage(websocket.TextMessage, serializedDiff); err != nil {
						conn.Close()
						delete(conns, conn)
					}
				}
			}
			e.mu.Unlock()
			time.Sleep(5 * time.Second)
		}
	}()
}

// @id GetLatestScoresForEvent
// @Description Fetches the latest scores for the current event
// @Tags scores
// @Produce json
// @Success 200 {object} ScoreMap
// @Param event_id path int true "Event Id"
// @Router /events/{event_id}/scores/latest [get]
func (e *ScoreController) getLatestScoresForEventHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		event := getEvent(c)
		if event == nil {
			return
		}
		scores := e.scoreService.LatestScores[event.Id]
		if scores == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "No scores found for event"})
			return
		}
		c.JSON(http.StatusOK, scores)
	}
}

type Score struct {
	Points    int       `json:"points" binding:"required"`
	UserId    int       `json:"user_id" binding:"required"`
	Rank      int       `json:"rank" binding:"required"`
	Timestamp time.Time `json:"timestamp" binding:"required"`
	Number    int       `json:"number" binding:"required"`
	Finished  bool      `json:"finished" binding:"required"`
}

type ScoreDiff struct {
	Score     *Score           `json:"score" binding:"required"`
	FieldDiff []string         `json:"field_diff" binding:"required"`
	DiffType  service.Difftype `json:"diff_type" binding:"required"`
}

type ScoreMap map[string]*ScoreDiff

func toScoreDiffResponse(scoreDiff *service.ScoreDifference) *ScoreDiff {
	return &ScoreDiff{
		Score:     toScoreResponse(scoreDiff.Score),
		FieldDiff: scoreDiff.FieldDiff,
		DiffType:  scoreDiff.DiffType,
	}
}

func toScoreMapResponse(scoreMap service.ScoreMap) ScoreMap {
	response := make(ScoreMap)
	for id, score := range scoreMap {
		response[id] = toScoreDiffResponse(score)
	}
	return response
}

func toScoreResponse(score *scoring.Score) *Score {
	return &Score{
		Points:    score.Points,
		UserId:    score.UserId,
		Rank:      score.Rank,
		Timestamp: score.Timestamp,
		Number:    score.Number,
		Finished:  score.Finished,
	}
}
