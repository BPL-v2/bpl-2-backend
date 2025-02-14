package controller

import (
	"bpl/client"
	"bpl/repository"
	"bpl/service"
	"bpl/utils"
	"crypto/sha256"
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ObjectiveController struct {
	service      *service.ObjectiveService
	eventService *service.EventService
}

func NewObjectiveController() *ObjectiveController {
	return &ObjectiveController{service: service.NewObjectiveService(), eventService: service.NewEventService()}
}

func setupObjectiveController() []RouteInfo {
	e := NewObjectiveController()
	baseUrl := "/scoring/objectives"
	routes := []RouteInfo{
		{Method: "PUT", Path: "", HandlerFunc: e.createObjectiveHandler(), Authenticated: true, RequiredRoles: []repository.Permission{repository.PermissionAdmin}},
		{Method: "GET", Path: "/:id", HandlerFunc: e.getObjectiveByIdHandler(), Authenticated: true, RequiredRoles: []repository.Permission{repository.PermissionAdmin}},
		{Method: "DELETE", Path: "/:id", HandlerFunc: e.deleteObjectiveHandler(), Authenticated: true, RequiredRoles: []repository.Permission{repository.PermissionAdmin}},
		{Method: "GET", Path: "/parser", HandlerFunc: e.getObjectiveParserHandler()},
	}
	for i, route := range routes {
		routes[i].Path = baseUrl + route.Path
	}
	return routes
}

// @id CreateObjective
// @Description Creates a new objective
// @Tags objective
// @Accept json
// @Produce json
// @Param body body ObjectiveCreate true "Objective to create"
// @Success 201 {object} ObjectiveResponse
// @Router /scoring/objectives [put]
func (e *ObjectiveController) createObjectiveHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var objectiveCreate ObjectiveCreate
		if err := c.BindJSON(&objectiveCreate); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		objective, err := e.service.CreateObjective(objectiveCreate.toModel())
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(404, gin.H{"error": "Category not found"})
			} else {
				c.JSON(400, gin.H{"error": err.Error()})
			}
			return
		}
		c.JSON(201, toObjectiveResponse(objective))
	}
}

// @id DeleteObjective
// @Description Deletes an objective
// @Tags objective
// @Produce json
// @Param id path int true "Objective ID"
// @Success 204
// @Router /scoring/objectives/{id} [delete]
func (e *ObjectiveController) deleteObjectiveHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		err = e.service.DeleteObjective(id)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(404, gin.H{"error": "Objective not found"})
			} else {
				c.JSON(500, gin.H{"error": err.Error()})
			}
			return
		}
		c.JSON(204, nil)
	}
}

// @id GetObjective
// @Description Gets an objective by id
// @Tags objective
// @Produce json
// @Param id path int true "Objective ID"
// @Success 200 {object} ObjectiveResponse
// @Router /scoring/objectives/{id} [get]
func (e *ObjectiveController) getObjectiveByIdHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		objective, err := e.service.GetObjectiveById(id)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(404, gin.H{"error": "Objective not found"})
			} else {
				c.JSON(500, gin.H{"error": err.Error()})
			}
			return
		}
		c.JSON(200, toObjectiveResponse(objective))
	}
}

func (e *ObjectiveController) getObjectiveParserHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		currentEvent, err := e.eventService.GetCurrentEvent()
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		parser, err := e.service.GetParser(currentEvent.ScoringCategoryID)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(404, gin.H{"error": "Category not found"})
			} else {
				c.JSON(500, gin.H{"error": err.Error()})
			}
			return
		}

		var item client.Item
		if err := c.BindJSON(&item); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, parser.CheckForCompletions(&item))
	}
}

type ObjectiveCreate struct {
	ID             int                        `json:"id"`
	Name           string                     `json:"name" binding:"required"`
	Extra          string                     `json:"extra"`
	RequiredNumber int                        `json:"required_number" binding:"required"`
	ObjectiveType  repository.ObjectiveType   `json:"objective_type" binding:"required"`
	NumberField    repository.NumberField     `json:"number_field" binding:"required"`
	Aggregation    repository.AggregationType `json:"aggregation" binding:"required"`
	CategoryId     int                        `json:"category_id" binding:"required"`
	Conditions     []ConditionCreate          `json:"conditions" binding:"required"`
	ValidFrom      *time.Time                 `json:"valid_from" binding:"omitempty"`
	ValidTo        *time.Time                 `json:"valid_to" binding:"omitempty"`
	ScoringId      *int                       `json:"scoring_preset_id"`
}

type ObjectiveResponse struct {
	ID              int                        `json:"id" binding:"required"`
	Name            string                     `json:"name" binding:"required"`
	Extra           string                     `json:"extra" binding:"required"`
	RequiredNumber  int                        `json:"required_number" binding:"required"`
	CategoryID      int                        `json:"category_id" binding:"required"`
	ObjectiveType   repository.ObjectiveType   `json:"objective_type" binding:"required"`
	Conditions      []*ConditionResponse       `json:"conditions" binding:"required"`
	ValidFrom       *time.Time                 `json:"valid_from" binding:"omitempty"`
	ValidTo         *time.Time                 `json:"valid_to" binding:"omitempty"`
	ScoringPresetID *int                       `json:"scoring_preset_id"`
	NumberField     repository.NumberField     `json:"number_field" binding:"required"`
	Aggregation     repository.AggregationType `json:"aggregation" binding:"required"`
}

func (e *ObjectiveCreate) toModel() *repository.Objective {
	return &repository.Objective{
		ID:             e.ID,
		Name:           e.Name,
		Extra:          e.Extra,
		RequiredAmount: e.RequiredNumber,
		ObjectiveType:  e.ObjectiveType,
		NumberField:    e.NumberField,
		Aggregation:    e.Aggregation,
		Conditions:     utils.Map(e.Conditions, func(c ConditionCreate) *repository.Condition { return c.toModel() }),
		ValidFrom:      e.ValidFrom,
		ValidTo:        e.ValidTo,
		CategoryID:     e.CategoryId,
		ScoringId:      e.ScoringId,
	}
}

func toObjectiveResponse(objective *repository.Objective) *ObjectiveResponse {
	if objective == nil {
		return nil
	}
	return &ObjectiveResponse{
		ID:              objective.ID,
		Name:            objective.Name,
		Extra:           objective.Extra,
		RequiredNumber:  objective.RequiredAmount,
		CategoryID:      objective.CategoryID,
		ObjectiveType:   objective.ObjectiveType,
		ValidFrom:       objective.ValidFrom,
		ValidTo:         objective.ValidTo,
		Conditions:      utils.Map(objective.Conditions, toConditionResponse),
		NumberField:     objective.NumberField,
		Aggregation:     objective.Aggregation,
		ScoringPresetID: objective.ScoringId,
	}
}

func toPublicObjectiveResponse(objective *repository.Objective) *ObjectiveResponse {
	if objective == nil {
		return nil
	}

	if objective.ValidFrom != nil && time.Now().Before(*objective.ValidFrom) {
		return &ObjectiveResponse{
			Name:       fmt.Sprintf("%x", sha256.Sum256([]byte(objective.Name))),
			CategoryID: objective.CategoryID,
			ValidFrom:  objective.ValidFrom,
			ValidTo:    objective.ValidTo,
		}
	}
	return toObjectiveResponse(objective)
}
