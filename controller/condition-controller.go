package controller

import (
	"bpl/client"
	"bpl/parser"
	"bpl/repository"
	"bpl/service"
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ConditionController struct {
	conditionService *service.ConditionService
	objectiveService *service.ObjectiveService
	eventService     *service.EventService
}

func NewConditionController() *ConditionController {
	return &ConditionController{
		conditionService: service.NewConditionService(),
		objectiveService: service.NewObjectiveService(),
		eventService:     service.NewEventService(),
	}
}

func setupConditionController() []RouteInfo {
	e := NewConditionController()
	baseUrl := "/events/:event_id/conditions"
	routes := []RouteInfo{
		{Method: "PUT", Path: "", HandlerFunc: e.createConditionHandler(), Authenticated: true, RequiredRoles: []repository.Permission{repository.PermissionAdmin}},
		{Method: "DELETE", Path: "/:id", HandlerFunc: e.deleteConditionHandler(), Authenticated: true, RequiredRoles: []repository.Permission{repository.PermissionAdmin}},
		{Method: "GET", Path: "/valid-mappings", HandlerFunc: e.getValidMappingsHandler(), Authenticated: true, RequiredRoles: []repository.Permission{repository.PermissionAdmin}},
		{Method: "POST", Path: "/test", HandlerFunc: e.testConditionHandler(), Authenticated: true, RequiredRoles: []repository.Permission{repository.PermissionAdmin}},
	}
	for i, route := range routes {
		routes[i].Path = baseUrl + route.Path
	}
	return routes
}

// @id CreateCondition
// @Description Creates a condition
// @Tags condition
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param event_id path int true "Event Id"
// @Param condition body ConditionCreate true "Condition to create"
// @Success 201 {object} Condition
// @Router /events/{event_id}/conditions [put]
func (e *ConditionController) createConditionHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var conditionCreate ConditionCreate
		if err := c.BindJSON(&conditionCreate); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		event, err := e.eventService.GetEventByObjectiveId(conditionCreate.ObjectiveId)
		if err != nil || event.Locked {
			c.JSON(400, gin.H{"error": "Event is locked"})
			return
		}
		condition, err := e.conditionService.CreateCondition(conditionCreate.toModel())
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(404, gin.H{"error": "Objective not found"})
			} else {
				c.JSON(400, gin.H{"error": err.Error()})
			}
			return
		}
		c.JSON(201, toConditionResponse(condition))
	}
}

// @id DeleteCondition
// @Description Deletes a condition
// @Security BearerAuth
// @Tags condition
// @Param event_id path int true "Event Id"
// @Param id path int true "Condition Id"
// @Router /events/{event_id}/conditions/{id} [delete]
func (e *ConditionController) deleteConditionHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		conditionId, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		event, err := e.eventService.GetEventByConditionId(conditionId)
		if err != nil || event.Locked {
			fmt.Println(err)
			c.JSON(400, gin.H{"error": "Event is locked"})
			return
		}

		err = e.conditionService.DeleteCondition(conditionId)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(404, gin.H{"error": "Condition not found"})
			} else {
				c.JSON(500, gin.H{"error": err.Error()})
			}
			return
		}
		c.JSON(204, nil)
	}
}

// @id GetValidMappings
// @Description Get valid mappings for conditions
// @Security BearerAuth
// @Tags condition
// @Produce json
// @Param event_id path int true "Event Id"
// @Success 200 {object} ConditionMappings
// @Router /events/{event_id}/conditions/valid-mappings [get]
func (e *ConditionController) getValidMappingsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, ConditionMappings{
			FieldToType:                 repository.FieldToType,
			ValidOperators:              repository.OperatorsForTypes,
			ObjectiveTypeToNumberFields: repository.ObjectiveTypeToNumberFields,
		})
	}
}

func (e *ConditionController) testConditionHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var conditionTest ConditionTest
		if err := c.BindJSON(&conditionTest); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		conditions := make([]*repository.Condition, 0, len(conditionTest.Conditions))
		for _, condition := range conditionTest.Conditions {
			conditions = append(conditions, &repository.Condition{
				Operator: repository.Operator(condition.Operator),
				Field:    repository.ItemField(condition.ItemField),
				Value:    condition.FieldValue,
			})
		}
		itemChecker, err := parser.ComperatorFromConditions(conditions)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, itemChecker(&conditionTest.Item))
	}
}

type ConditionTest struct {
	Conditions []struct {
		Operator   repository.Operator  `json:"operator" binding:"required"`
		ItemField  repository.ItemField `json:"field" binding:"required"`
		FieldValue string               `json:"value" binding:"required"`
	} `json:"conditions" binding:"required"`
	Item client.Item `json:"item" binding:"required"`
}

type ConditionCreate struct {
	Operator    repository.Operator  `json:"operator" binding:"required,oneof=EQ NEQ GT GTE LT LTE IN NOT_IN MATCHES CONTAINS CONTAINS_ALL CONTAINS_MATCH CONTAINS_ALL_MATCHES"`
	ItemField   repository.ItemField `json:"field" binding:"required,oneof=BASE_TYPE NAME TYPE_LINE RARITY ILVL FRAME_TYPE TALISMAN_TIER ENCHANT_MODS EXPLICIT_MODS IMPLICIT_MODS CRAFTED_MODS FRACTURED_MODS SIX_LINK"`
	FieldValue  string               `json:"value" binding:"required"`
	Id          int                  `json:"id"`
	ObjectiveId int                  `json:"objective_id" binding:"required"`
}

type Condition struct {
	Operator   repository.Operator  `json:"operator" binding:"required"`
	ItemField  repository.ItemField `json:"field" binding:"required"`
	FieldValue string               `json:"value" binding:"required"`
	Id         int                  `json:"id" binding:"required"`
}

func (e *ConditionCreate) toModel() *repository.Condition {
	return &repository.Condition{
		Id:          e.Id,
		Operator:    repository.Operator(e.Operator),
		Field:       repository.ItemField(e.ItemField),
		Value:       e.FieldValue,
		ObjectiveId: e.ObjectiveId,
	}
}

func toConditionResponse(condition *repository.Condition) *Condition {
	if condition == nil {
		return nil
	}
	return &Condition{
		Operator:   condition.Operator,
		ItemField:  condition.Field,
		FieldValue: condition.Value,
		Id:         condition.Id,
	}
}

type ConditionMappings struct {
	FieldToType                 map[repository.ItemField]repository.FieldType         `json:"field_to_type" binding:"required"`
	ValidOperators              map[repository.FieldType][]repository.Operator        `json:"valid_operators" binding:"required"`
	ObjectiveTypeToNumberFields map[repository.ObjectiveType][]repository.NumberField `json:"objective_type_to_number_fields" binding:"required"`
}
