package controller

import (
	"bpl/repository"
	"bpl/service"
	"bpl/utils"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ScoringRuleController struct {
	ruleService  service.ScoringRuleService
	eventService service.EventService
}

func NewScoringRuleController() *ScoringRuleController {
	return &ScoringRuleController{
		ruleService:  service.NewScoringRulesService(),
		eventService: service.NewEventService(),
	}
}

func setupScoringRuleController() []RouteInfo {
	e := NewScoringRuleController()
	editorRoles := []repository.Permission{repository.PermissionAdmin, repository.PermissionManager, repository.PermissionObjectiveDesigner}
	routes := []RouteInfo{
		{Method: "GET", Path: "/events/:event_id/scoring-rules", HandlerFunc: e.getScoringRulesForEventHandler()},
		{Method: "PUT", Path: "/events/:event_id/scoring-rules", HandlerFunc: e.createScoringRuleHandler(), Authenticated: true, RequiredRoles: editorRoles},
		{Method: "DELETE", Path: "/events/:event_id/scoring-rules/:id", HandlerFunc: e.deleteScoringRuleHandler(), Authenticated: true, RequiredRoles: editorRoles},
	}
	return routes
}

// @id GetScoringRulesForEvent
// @Description Fetches the scoring rules for the current event
// @Tags scoring
// @Produce json
// @Param event_id path int true "Event Id"
// @Success 200 {array} ScoringRule
// @Router /events/{event_id}/scoring-rules [get]
func (e *ScoringRuleController) getScoringRulesForEventHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		event := getEvent(c)
		if event == nil {
			return
		}
		rules, err := e.ruleService.GetRulesForEvent(event.Id)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, utils.Map(rules, toScoringRuleResponse))
	}
}

// @id CreateScoringRule
// @Description Creates a new scoring rule
// @Tags scoring
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param event_id path int true "Event Id"
// @Param scoringRuleCreate body ScoringRuleCreate true "Rule to create"
// @Success 200 {object} ScoringRule
// @Router /events/{event_id}/scoring-rules [put]
func (e *ScoringRuleController) createScoringRuleHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		event := getEvent(c)
		if event == nil {
			return
		}
		if event.Locked {
			c.JSON(400, gin.H{"error": "event is locked"})
			return
		}
		var ruleCreate ScoringRuleCreate
		if err := c.ShouldBindJSON(&ruleCreate); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		rule := ruleCreate.toModel()
		rule.EventId = event.Id
		rule, err := e.ruleService.SaveRule(rule)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, toScoringRuleResponse(rule))
	}
}

// @id DeleteScoringRule
// @Description Deletes a scoring rule by id
// @Security BearerAuth
// @Tags scoring
// @Produce json
// @Param event_id path int true "Event Id"
// @Param id path int true "Rule Id"
// @Success 200
// @Router /events/{event_id}/scoring-rules/{id} [delete]
func (e *ScoringRuleController) deleteScoringRuleHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		event := getEvent(c)
		if event == nil {
			return
		}
		if event.Locked {
			c.JSON(400, gin.H{"error": "event is locked"})
			return
		}

		err = e.ruleService.DeleteRule(id)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(404, gin.H{"error": "rule not found"})
				return
			}
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{})
	}
}

type ScoringRuleCreate struct {
	Id          *int                       `json:"id"`
	Name        string                     `json:"name" binding:"required"`
	Description string                     `json:"description"`
	Points      []float64                  `json:"points" binding:"required"`
	RuleType    repository.ScoringRuleType `json:"scoring_rule" binding:"required"`
	PointCap    int                        `json:"point_cap"`
	Extra       map[string]string          `json:"extra"`
}

func (e *ScoringRuleCreate) toModel() *repository.ScoringRule {
	rule := &repository.ScoringRule{
		Name:        e.Name,
		Description: e.Description,
		Points:      e.Points,
		RuleType:    e.RuleType,
		PointCap:    e.PointCap,
		Extra:       e.Extra,
	}
	if e.Id != nil {
		rule.Id = *e.Id
	}
	return rule
}

type ScoringRule struct {
	Id          int                        `json:"id" binding:"required"`
	Name        string                     `json:"name" binding:"required"`
	Description string                     `json:"description" binding:"required"`
	Points      []float64                  `json:"points" binding:"required"`
	RuleType    repository.ScoringRuleType `json:"scoring_rule" binding:"required"`
	PointCap    int                        `json:"point_cap"`
	Extra       map[string]string          `json:"extra"`
}

func toScoringRuleResponse(rule *repository.ScoringRule) *ScoringRule {
	if rule == nil {
		return nil
	}
	return &ScoringRule{
		Id:          rule.Id,
		Name:        rule.Name,
		Description: rule.Description,
		Points:      rule.Points,
		RuleType:    rule.RuleType,
		PointCap:    rule.PointCap,
		Extra:       rule.Extra,
	}
}
