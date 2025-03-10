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
		{Method: "PUT", Path: "", HandlerFunc: e.createObjectiveHandler()},
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
// @Success 201 {object} Objective
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
// @Param id path int true "Objective Id"
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
// @Param id path int true "Objective Id"
// @Success 200 {object} Objective
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
		parser, err := e.service.GetParser(currentEvent.ScoringCategoryId)
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

type ObjectiveConditionCreate struct {
	Id         int                  `json:"id"`
	Operator   repository.Operator  `json:"operator" binding:"required,oneof=EQ NEQ GT GTE LT LTE IN NOT_IN MATCHES CONTAINS CONTAINS_ALL CONTAINS_MATCH CONTAINS_ALL_MATCHES"`
	ItemField  repository.ItemField `json:"field" binding:"required,oneof=BASE_TYPE NAME TYPE_LINE RARITY ILVL FRAME_TYPE TALISMAN_TIER ENCHANT_MODS EXPLICIT_MODS IMPLICIT_MODS CRAFTED_MODS FRACTURED_MODS SIX_LINK"`
	FieldValue string               `json:"value" binding:"required"`
}

func (e *ObjectiveConditionCreate) toModel() *repository.Condition {
	return &repository.Condition{
		Id:       e.Id,
		Operator: e.Operator,
		Field:    e.ItemField,
		Value:    e.FieldValue,
	}
}

type ObjectiveCreate struct {
	Id             int                        `json:"id"`
	Name           string                     `json:"name" binding:"required"`
	Extra          string                     `json:"extra"`
	RequiredNumber int                        `json:"required_number" binding:"required"`
	ObjectiveType  repository.ObjectiveType   `json:"objective_type" binding:"required"`
	NumberField    repository.NumberField     `json:"number_field" binding:"required"`
	Aggregation    repository.AggregationType `json:"aggregation" binding:"required"`
	CategoryId     int                        `json:"category_id" binding:"required"`
	Conditions     []ObjectiveConditionCreate `json:"conditions" binding:"required"`
	ValidFrom      *time.Time                 `json:"valid_from" binding:"omitempty"`
	ValidTo        *time.Time                 `json:"valid_to" binding:"omitempty"`
	ScoringId      *int                       `json:"scoring_preset_id"`
}

type Objective struct {
	Id              int                        `json:"id" binding:"required"`
	Name            string                     `json:"name" binding:"required"`
	Extra           string                     `json:"extra" binding:"required"`
	RequiredNumber  int                        `json:"required_number" binding:"required"`
	CategoryId      int                        `json:"category_id" binding:"required"`
	ObjectiveType   repository.ObjectiveType   `json:"objective_type" binding:"required"`
	Conditions      []*Condition               `json:"conditions" binding:"required"`
	ValidFrom       *time.Time                 `json:"valid_from" binding:"omitempty"`
	ValidTo         *time.Time                 `json:"valid_to" binding:"omitempty"`
	ScoringPresetId *int                       `json:"scoring_preset_id"`
	ScoringPreset   *ScoringPreset             `json:"scoring_preset"`
	NumberField     repository.NumberField     `json:"number_field" binding:"required"`
	Aggregation     repository.AggregationType `json:"aggregation" binding:"required"`
}

func (e *ObjectiveCreate) toModel() *repository.Objective {
	return &repository.Objective{
		Id:             e.Id,
		Name:           e.Name,
		Extra:          e.Extra,
		RequiredAmount: e.RequiredNumber,
		ObjectiveType:  e.ObjectiveType,
		NumberField:    e.NumberField,
		Aggregation:    e.Aggregation,
		Conditions:     utils.Map(e.Conditions, func(c ObjectiveConditionCreate) *repository.Condition { return c.toModel() }),
		ValidFrom:      e.ValidFrom,
		ValidTo:        e.ValidTo,
		CategoryId:     e.CategoryId,
		ScoringId:      e.ScoringId,
	}
}

func toObjectiveResponse(objective *repository.Objective) *Objective {
	if objective == nil {
		return nil
	}
	return &Objective{
		Id:              objective.Id,
		Name:            objective.Name,
		Extra:           objective.Extra,
		RequiredNumber:  objective.RequiredAmount,
		CategoryId:      objective.CategoryId,
		ObjectiveType:   objective.ObjectiveType,
		ValidFrom:       objective.ValidFrom,
		ValidTo:         objective.ValidTo,
		Conditions:      utils.Map(objective.Conditions, toConditionResponse),
		NumberField:     objective.NumberField,
		Aggregation:     objective.Aggregation,
		ScoringPresetId: objective.ScoringId,
		ScoringPreset:   toScoringPresetResponse(objective.ScoringPreset),
	}
}

func toPublicObjectiveResponse(objective *repository.Objective) *Objective {
	if objective == nil {
		return nil
	}

	if objective.ValidFrom != nil && time.Now().Before(*objective.ValidFrom) {
		return &Objective{
			Name:       fmt.Sprintf("%x", sha256.Sum256([]byte(objective.Name))),
			CategoryId: objective.CategoryId,
			ValidFrom:  objective.ValidFrom,
			ValidTo:    objective.ValidTo,
		}
	}
	return toObjectiveResponse(objective)
}
