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
	baseUrl := "/events/:event_id/objectives"
	routes := []RouteInfo{
		{Method: "GET", Path: "", HandlerFunc: e.GetObjectiveTreeForEventHandler()},
		{Method: "PUT", Path: "", HandlerFunc: e.createObjectiveHandler(), Authenticated: true, RequiredRoles: []repository.Permission{repository.PermissionAdmin, repository.PermissionObjectiveDesigner}},
		{Method: "GET", Path: "/:id", HandlerFunc: e.getObjectiveByIdHandler(), Authenticated: true, RequiredRoles: []repository.Permission{repository.PermissionAdmin, repository.PermissionObjectiveDesigner}},
		{Method: "DELETE", Path: "/:id", HandlerFunc: e.deleteObjectiveHandler(), Authenticated: true, RequiredRoles: []repository.Permission{repository.PermissionAdmin, repository.PermissionObjectiveDesigner}},
		// todo: move this somewhere else
		{Method: "GET", Path: "/parser", HandlerFunc: e.getObjectiveParserHandler(), Authenticated: true, RequiredRoles: []repository.Permission{repository.PermissionAdmin, repository.PermissionObjectiveDesigner}},
	}
	for i, route := range routes {
		routes[i].Path = baseUrl + route.Path
	}
	return routes
}

// @id GetObjectiveTreeForEvent
// @Description Gets all objectives for an event
// @Tags objective
// @Produce json
// @Param event_id path int true "Event Id"
// @Success 200 {object} Objective
// @Router /events/{event_id}/objectives [get]
func (e *ObjectiveController) GetObjectiveTreeForEventHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		event := getEvent(c)
		if event == nil {
			return
		}
		rootObjective, err := e.service.GetObjectiveTreeForEvent(event.Id, "Conditions")
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(404, gin.H{"error": "Objectives not found"})
			} else {
				c.JSON(500, gin.H{"error": err.Error()})
			}
			return
		}
		if rootObjective == nil {
			c.JSON(404, gin.H{"error": "Objectives not found"})
			return
		}
		if !getEvent(c).Public {
			c.JSON(200, toObjectiveResponse(rootObjective))
			return
		}
		// if the event is public, we return a public version of the objective
		c.JSON(200, toPublicObjectiveResponse(rootObjective))

	}
}

// @id CreateObjective
// @Description Creates a new objective
// @Security BearerAuth
// @Tags objective
// @Accept json
// @Produce json
// @Param event_id path int true "Event Id"
// @Param body body ObjectiveCreate true "Objective to create"
// @Success 201 {object} Objective
// @Router /events/{event_id}/objectives [put]
func (e *ObjectiveController) createObjectiveHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var objectiveCreate ObjectiveCreate
		if err := c.BindJSON(&objectiveCreate); err != nil {
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
// @Security BearerAuth
// @Tags objective
// @Produce json
// @Param event_id path int true "Event Id"
// @Param id path int true "Objective Id"
// @Success 204
// @Router /events/{event_id}/objectives/{id} [delete]
func (e *ObjectiveController) deleteObjectiveHandler() gin.HandlerFunc {
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
// @Security BearerAuth
// @Tags objective
// @Produce json
// @Param event_id path int true "Event Id"
// @Param id path int true "Objective Id"
// @Success 200 {object} Objective
// @Router /events/{event_id}/objectives/{id} [get]
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
		parser, err := e.service.GetParser(currentEvent.Id)
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
	ParentId       int                        `json:"parent_id" binding:"required"`
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
	ParentId        *int                       `json:"parent_id" binding:"required"`
	ObjectiveType   repository.ObjectiveType   `json:"objective_type" binding:"required"`
	Conditions      []*Condition               `json:"conditions" binding:"required"`
	ValidFrom       *time.Time                 `json:"valid_from" binding:"omitempty"`
	ValidTo         *time.Time                 `json:"valid_to" binding:"omitempty"`
	ScoringPresetId *int                       `json:"scoring_preset_id"`
	ScoringPreset   *ScoringPreset             `json:"scoring_preset"`
	NumberField     repository.NumberField     `json:"number_field" binding:"required"`
	Aggregation     repository.AggregationType `json:"aggregation" binding:"required"`
	Children        []*Objective               `json:"children" binding:"required"`
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
		ParentId:       &e.ParentId,
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
		ParentId:        objective.ParentId,
		ObjectiveType:   objective.ObjectiveType,
		ValidFrom:       objective.ValidFrom,
		ValidTo:         objective.ValidTo,
		Conditions:      utils.Map(objective.Conditions, toConditionResponse),
		NumberField:     objective.NumberField,
		Aggregation:     objective.Aggregation,
		ScoringPresetId: objective.ScoringId,
		ScoringPreset:   toScoringPresetResponse(objective.ScoringPreset),
		Children:        utils.Map(objective.Children, toObjectiveResponse),
	}
}

func toPublicObjectiveResponse(objective *repository.Objective) *Objective {
	if objective == nil {
		return nil
	}

	if objective.ValidFrom != nil && time.Now().Before(*objective.ValidFrom) {
		return &Objective{
			Name:            fmt.Sprintf("%x", sha256.Sum256([]byte(objective.Name))),
			ParentId:        objective.ParentId,
			ValidFrom:       objective.ValidFrom,
			ValidTo:         objective.ValidTo,
			ScoringPresetId: objective.ScoringId,
			ScoringPreset:   toScoringPresetResponse(objective.ScoringPreset),
		}
	}
	return toObjectiveResponse(objective)
}
