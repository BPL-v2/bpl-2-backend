package controller

import (
	"bpl/scoring"
	"bpl/service"
	"bpl/utils"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ScoreController struct {
	db                     *gorm.DB
	scoringCategoryService *service.ScoringCategoryService
	eventService           *service.EventService
}

func NewScoreController(db *gorm.DB) *ScoreController {
	return &ScoreController{
		db:                     db,
		scoringCategoryService: service.NewScoringCategoryService(db),
		eventService:           service.NewEventService(db),
	}
}

func setupScoreController(db *gorm.DB) []RouteInfo {
	e := NewScoreController(db)
	baseUrl := "events/:event_id/scores"
	routes := []RouteInfo{
		{Method: "GET", Path: "/latest", HandlerFunc: e.getLatestScoresForEventHandler()},
	}
	for i, route := range routes {
		routes[i].Path = baseUrl + route.Path
	}
	return routes
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
		event_id, err := strconv.Atoi(c.Param("event_id"))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		event, err := e.eventService.GetEventById(event_id, "Teams", "Teams.Users")
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		rules, err := e.scoringCategoryService.GetRulesForEvent(event.ID, "Objectives", "Objectives.Conditions", "ScoringPreset", "Objectives.ScoringPreset")
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		matches, err := scoring.AggregateMatches(e.db, event)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		scores, err := scoring.EvaluateAggregations(rules, matches)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, utils.Map(scores, toScoreResponse))
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
