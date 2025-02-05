package controller

import (
	"bpl/client"
	"bpl/scoring"
	"bpl/service"
	"bpl/utils"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
)

type ScoreController struct {
	db                     *gorm.DB
	scoringCategoryService *service.ScoringCategoryService
	eventService           *service.EventService
	poeClient              *client.PoEClient
	mu                     sync.Mutex
	connections            map[int]map[*websocket.Conn]bool
	latestScores           map[int]*LatestScore
}

type LatestScore struct {
	score []byte
	hash  []byte
}

func (l *LatestScore) Equals(other *LatestScore) bool {
	if len(l.hash) != len(other.hash) {
		return false
	}
	for i := range l.hash {
		if l.hash[i] != other.hash[i] {
			return false
		}
	}
	return true
}

func NewLatestScore(scores []*scoring.Score) (*LatestScore, error) {
	sort.Slice(scores, func(i, j int) bool {
		if scores[i].Type != scores[j].Type {
			return scores[i].Type < scores[j].Type
		}
		if scores[i].ID != scores[j].ID {
			return scores[i].ID < scores[j].ID
		}
		return scores[i].TeamID < scores[j].TeamID
	})
	scoreBytes, err := json.Marshal(utils.Map(scores, toScoreResponse))
	if err != nil {
		return nil, err
	}

	return &LatestScore{score: scoreBytes, hash: calculateHash(scores)}, nil
}

func NewScoreController(db *gorm.DB) *ScoreController {
	scoringCategoryService := service.NewScoringCategoryService(db)
	eventService := service.NewEventService(db)
	poeClient := client.NewPoEClient(os.Getenv("POE_CLIENT_AGENT"), 10, false, 10)
	controller := &ScoreController{
		db:                     db,
		scoringCategoryService: scoringCategoryService,
		eventService:           eventService,
		poeClient:              poeClient,
		connections:            make(map[int]map[*websocket.Conn]bool),
		latestScores:           make(map[int]*LatestScore),
	}
	controller.StartScoreUpdater()
	return controller
}

func setupScoreController(db *gorm.DB) []RouteInfo {
	e := NewScoreController(db)
	baseUrl := "events/:event_id/scores"
	routes := []RouteInfo{
		{Method: "GET", Path: "/latest", HandlerFunc: e.getLatestScoresForEventHandler()},
		{Method: "POST", Path: "/:minutes", HandlerFunc: e.FetchStashChangesHandler()},
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

func (e *ScoreController) WebSocketHandler(c *gin.Context) {
	eventID, err := strconv.Atoi(c.Param("event_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event ID"})
		return
	}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		http.NotFound(c.Writer, c.Request)
		return
	}
	defer conn.Close()

	e.mu.Lock()
	if _, ok := e.connections[eventID]; !ok {
		e.connections[eventID] = make(map[*websocket.Conn]bool)
	}
	e.connections[eventID][conn] = true
	e.mu.Unlock()

	// Send the latest score to the new subscriber
	if _, ok := e.latestScores[eventID]; ok {
		if err := conn.WriteMessage(websocket.TextMessage, e.latestScores[eventID].score); err != nil {
			return
		}
	}

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			e.mu.Lock()
			delete(e.connections[eventID], conn)
			if len(e.connections[eventID]) == 0 {
				delete(e.connections, eventID)
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
			for eventID, conns := range e.connections {
				if len(utils.Values(conns)) == 0 {
					continue
				}
				log.Println("Calculating scores for event", eventID)
				newScore, err := e.calcScores(eventID)
				if err != nil {
					continue
				}
				if oldScore, ok := e.latestScores[eventID]; ok && oldScore.Equals(newScore) {
					continue
				}
				fmt.Println("New score for event", eventID)
				e.latestScores[eventID] = newScore
				for conn := range conns {
					if err := conn.WriteMessage(websocket.TextMessage, newScore.score); err != nil {
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
// @Success 200 {array} ScoreResponse
// @Param event_id path int true "Event ID"
// @Router /events/{event_id}/scores/latest [get]
func (e *ScoreController) getLatestScoresForEventHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID, err := strconv.Atoi(c.Param("event_id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event ID"})
			return
		}
		scores, ok := e.latestScores[eventID]
		if !ok {
			scores, err = e.calcScores(eventID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			e.latestScores[eventID] = scores
		}
		c.Writer.WriteHeader(http.StatusOK)
		c.Writer.Write(scores.score)
	}
}

func calculateHash(scores []*scoring.Score) []byte {
	hash := sha256.New()
	for _, score := range scores {
		hash.Write([]byte(strconv.Itoa(score.TeamID + score.Number + score.Points + score.UserID)))
	}
	return hash.Sum(nil)
}

func (e *ScoreController) calcScores(eventId int) (score *LatestScore, err error) {
	event, err := e.eventService.GetEventById(eventId, "Teams", "Teams.Users")
	if err != nil {
		return nil, err
	}
	rules, err := e.scoringCategoryService.GetRulesForEvent(event.ID, "Objectives", "Objectives.Conditions", "ScoringPreset", "Objectives.ScoringPreset")
	if err != nil {
		return nil, err
	}
	matches, err := scoring.AggregateMatches(e.db, event)
	if err != nil {
		return nil, err
	}
	scores, err := scoring.EvaluateAggregations(rules, matches)
	if err != nil {
		return nil, err
	}
	return NewLatestScore(scores)
}

func (e *ScoreController) FetchStashChangesHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		minutes, err := strconv.Atoi(c.Param("minutes"))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		scoring.StashLoop(e.db, e.poeClient, time.Now().Add(time.Duration(minutes)*time.Minute))
		c.JSON(200, gin.H{"message": "Stash change fetch started"})
	}
}

type ScoreResponse struct {
	Type      scoring.ScoreType `json:"type" binding:"required"`
	ID        int               `json:"id" binding:"required"`
	Points    int               `json:"points" binding:"required"`
	TeamID    int               `json:"team_id" binding:"required"`
	UserID    int               `json:"user_id" binding:"required"`
	Rank      int               `json:"rank" binding:"required"`
	Timestamp time.Time         `json:"timestamp" binding:"required"`
	Number    int               `json:"number" binding:"required"`
	Finished  bool              `json:"finished" binding:"required"`
}

func toScoreResponse(score *scoring.Score) *ScoreResponse {
	return &ScoreResponse{
		Type:      score.Type,
		ID:        score.ID,
		Points:    score.Points,
		TeamID:    score.TeamID,
		UserID:    score.UserID,
		Rank:      score.Rank,
		Timestamp: score.Timestamp,
		Number:    score.Number,
		Finished:  score.Finished,
	}
}
