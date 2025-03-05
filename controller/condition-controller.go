package controller

import (
	"bpl/repository"
	"bpl/service"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ConditionController struct {
	service *service.ConditionService
}

func NewConditionController() *ConditionController {
	return &ConditionController{service: service.NewConditionService()}
}

func setupConditionController() []RouteInfo {
	e := NewConditionController()
	baseUrl := "/scoring/conditions"
	routes := []RouteInfo{
		{Method: "PUT", Path: "", HandlerFunc: e.createConditionHandler()},
		{Method: "DELETE", Path: "/:id", HandlerFunc: e.deleteConditionHandler()},
		{Method: "GET", Path: "/valid-mappings", HandlerFunc: e.getValidMappingsHandler()},
	}
	for i, route := range routes {
		routes[i].Path = baseUrl + route.Path
	}
	return routes
}

// @id CreateCondition
// @Description Creates a condition
// @Tags condition
// @Accept json
// @Produce json
// @Param condition body ConditionCreate true "Condition to create"
// @Success 201 {object} Condition
// @Router /scoring/conditions [put]
func (e *ConditionController) createConditionHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var conditionCreate ConditionCreate
		if err := c.BindJSON(&conditionCreate); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		condition := conditionCreate.toModel()

		condition, err := e.service.CreateCondition(condition)
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
// @Tags condition
// @Param id path int true "Condition Id"
// @Router /scoring/conditions/{id} [delete]
func (e *ConditionController) deleteConditionHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		conditionId, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		err = e.service.DeleteCondition(conditionId)
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
// @Tags condition
// @Produce json
// @Success 200 {object} ConditionMappings
// @Router /scoring/conditions/valid-mappings [get]
func (e *ConditionController) getValidMappingsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, ConditionMappings{
			FieldToType:    repository.FieldToType,
			ValidOperators: repository.OperatorsForTypes,
		})
	}
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
	FieldToType    map[repository.ItemField]repository.FieldType  `json:"field_to_type" binding:"required"`
	ValidOperators map[repository.FieldType][]repository.Operator `json:"valid_operators" binding:"required"`
}
