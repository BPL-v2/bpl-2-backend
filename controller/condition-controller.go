package controller

import (
	"bpl/repository"
	"bpl/service"
	"bpl/utils"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ConditionController struct {
	service *service.ConditionService
}

func NewConditionController(db *gorm.DB) *ConditionController {
	return &ConditionController{service: service.NewConditionService(db)}
}

func setupConditionController(db *gorm.DB) []gin.RouteInfo {
	e := NewConditionController(db)
	baseUrl := "/scoring-categories/:category_id/objectives/:objective_id/conditions"
	routes := []gin.RouteInfo{
		{Method: "GET", Path: "", HandlerFunc: e.getObjectiveConditionsHandler()},
		{Method: "POST", Path: "", HandlerFunc: e.createConditionHandler()},
		{Method: "DELETE", Path: "/:condition_id", HandlerFunc: e.deleteConditionHandler()},
		{Method: "PATCH", Path: "/:condition_id", HandlerFunc: e.updateConditionHandler()},
	}
	for i, route := range routes {
		routes[i].Path = baseUrl + route.Path
	}
	return routes
}

func (e *ConditionController) getObjectiveConditionsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		objective_id, err := strconv.Atoi(c.Param("objective_id"))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		conditions, err := e.service.GetConditionsByObjectiveId(objective_id)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(404, gin.H{"error": "Objective not found"})
			} else {
				c.JSON(500, gin.H{"error": err.Error()})
			}
			return
		}
		c.JSON(200, utils.Map(conditions, toConditionResponse))
	}
}

func (e *ConditionController) createConditionHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		objective_id, err := strconv.Atoi(c.Param("objective_id"))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		var conditionCreate ConditionCreate
		if err := c.BindJSON(&conditionCreate); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		condition := conditionCreate.toModel()
		condition.ObjectiveID = objective_id

		condition, err = e.service.CreateCondition(condition)
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

func (e *ConditionController) deleteConditionHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		condition_id, err := strconv.Atoi(c.Param("condition_id"))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		err = e.service.DeleteCondition(condition_id)
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

func (e *ConditionController) updateConditionHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		condition_id, err := strconv.Atoi(c.Param("condition_id"))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		var conditionCreate ConditionUpdate
		if err := c.BindJSON(&conditionCreate); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		condition, err := e.service.UpdateCondition(condition_id, conditionCreate.toModel())
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(404, gin.H{"error": "Condition not found"})
			} else {
				c.JSON(500, gin.H{"error": err.Error()})
			}
			return
		}
		c.JSON(200, toConditionResponse(condition))
	}
}

type ConditionCreate struct {
	Operator   repository.Operator  `json:"operator" binding:"required,oneof=EQ NEQ GT GTE LT LTE IN NOT_IN MATCHES CONTAINS CONTAINS_ALL CONTAINS_MATCH CONTAINS_ALL_MATCHES"`
	ItemField  repository.ItemField `json:"field" binding:"required,oneof=BASE_TYPE NAME TYPE_LINE RARITY ILVL FRAME_TYPE TALISMAN_TIER ENCHANT_MODS EXPLICIT_MODS IMPLICIT_MODS CRAFTED_MODS FRACTURED_MODS SIX_LINK"`
	FieldValue string               `json:"value" binding:"required"`
}

type ConditionUpdate struct {
	Operator   repository.Operator  `json:"operator"`
	ItemField  repository.ItemField `json:"field"`
	FieldValue string               `json:"value"`
}

type ConditionOut struct {
	Operator   repository.Operator  `json:"operator"`
	ItemField  repository.ItemField `json:"field"`
	FieldValue string               `json:"value"`
}

func (e *ConditionCreate) toModel() *repository.Condition {
	return &repository.Condition{
		Operator: repository.Operator(e.Operator),
		Field:    repository.ItemField(e.ItemField),
		Value:    e.FieldValue,
	}
}

func (e *ConditionUpdate) toModel() *repository.Condition {
	return &repository.Condition{
		Operator: repository.Operator(e.Operator),
		Field:    repository.ItemField(e.ItemField),
		Value:    e.FieldValue,
	}
}

func toConditionResponse(condition *repository.Condition) ConditionOut {
	return ConditionOut{
		Operator:   condition.Operator,
		ItemField:  condition.Field,
		FieldValue: condition.Value,
	}
}
