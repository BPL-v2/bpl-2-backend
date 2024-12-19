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

func NewConditionController(db *gorm.DB) *ConditionController {
	return &ConditionController{service: service.NewConditionService(db)}
}

func setupConditionController(db *gorm.DB) []RouteInfo {
	e := NewConditionController(db)
	baseUrl := "/scoring/conditions"
	routes := []RouteInfo{
		{Method: "PUT", Path: "", HandlerFunc: e.createConditionHandler()},
		{Method: "DELETE", Path: "/:id", HandlerFunc: e.deleteConditionHandler()},
	}
	for i, route := range routes {
		routes[i].Path = baseUrl + route.Path
	}
	return routes
}

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

func (e *ConditionController) deleteConditionHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		condition_id, err := strconv.Atoi(c.Param("id"))
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

type ConditionCreate struct {
	Operator    repository.Operator  `json:"operator" binding:"required,oneof=EQ NEQ GT GTE LT LTE IN NOT_IN MATCHES CONTAINS CONTAINS_ALL CONTAINS_MATCH CONTAINS_ALL_MATCHES"`
	ItemField   repository.ItemField `json:"field" binding:"required,oneof=BASE_TYPE NAME TYPE_LINE RARITY ILVL FRAME_TYPE TALISMAN_TIER ENCHANT_MODS EXPLICIT_MODS IMPLICIT_MODS CRAFTED_MODS FRACTURED_MODS SIX_LINK"`
	FieldValue  string               `json:"value" binding:"required"`
	ID          int                  `json:"id"`
	ObjectiveID int                  `json:"objective_id" binding:"required"`
}

type ConditionResponse struct {
	Operator   repository.Operator  `json:"operator"`
	ItemField  repository.ItemField `json:"field"`
	FieldValue string               `json:"value"`
	ID         int                  `json:"id"`
}

func (e *ConditionCreate) toModel() *repository.Condition {
	return &repository.Condition{
		ID:          e.ID,
		Operator:    repository.Operator(e.Operator),
		Field:       repository.ItemField(e.ItemField),
		Value:       e.FieldValue,
		ObjectiveID: e.ObjectiveID,
	}
}

func toConditionResponse(condition *repository.Condition) ConditionResponse {
	return ConditionResponse{
		Operator:   condition.Operator,
		ItemField:  condition.Field,
		FieldValue: condition.Value,
		ID:         condition.ID,
	}
}
