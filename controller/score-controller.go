package controller

import (
	"bpl/scoring"
	"bpl/service"
	"bpl/utils"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ScoreController struct {
	db                     *gorm.DB
	scoringCategoryService *service.ScoringCategoryService
}

func NewScoreController(db *gorm.DB) *ScoreController {
	return &ScoreController{db: db, scoringCategoryService: service.NewScoringCategoryService(db)}
}

func setupScoreController(db *gorm.DB) []RouteInfo {
	e := NewScoreController(db)
	baseUrl := "scores"
	routes := []RouteInfo{
		{Method: "GET", Path: "/latest", HandlerFunc: e.getLatestScoresForEventHandler()},
	}
	for i, route := range routes {
		routes[i].Path = baseUrl + route.Path
	}
	return routes
}

func (e *ScoreController) getLatestScoresForEventHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// eventId, err := strconv.Atoi(c.Param("event_id"))
		// if err != nil {
		// 	c.JSON(400, gin.H{"error": err.Error()})
		// 	return
		// }
		event, err := service.NewEventService(e.db).GetCurrentEvent("Teams", "Teams.Users")
		// event, err := service.NewEventService(e.db).GetEventById(eventId, "Teams", "Teams.Users")
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		rules, err := e.scoringCategoryService.GetRulesForEvent(event.ID)
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
	Type      scoring.ScoreType `json:"type"`
	ID        int               `json:"id"`
	Points    int               `json:"points"`
	TeamID    int               `json:"team_id"`
	UserID    int               `json:"user_id"`
	Rank      int               `json:"rank"`
	Timestamp time.Time         `json:"timestamp"`
	Number    int               `json:"number"`
	Finished  bool              `json:"finished"`
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
