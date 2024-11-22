package controller

import (
	"bpl/repository"
	"bpl/service"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ObjectiveController struct {
	service *service.ObjectiveService
}

func NewObjectiveController(db *gorm.DB) *ObjectiveController {
	return &ObjectiveController{service: service.NewObjectiveService(db)}
}

func setupObjectiveController(db *gorm.DB) []gin.RouteInfo {
	e := NewObjectiveController(db)
	baseUrl := "/scoring-categories/:category_id/objectives"
	routes := []gin.RouteInfo{
		{Method: "GET", Path: "", HandlerFunc: e.getCategoryObjectivesHandler()},
		{Method: "POST", Path: "", HandlerFunc: e.createObjectiveHandler()},
		{Method: "DELETE", Path: "/:objective_id", HandlerFunc: e.deleteObjectiveHandler()},
		{Method: "PATCH", Path: "/:objective_id", HandlerFunc: e.updateObjectiveHandler()},
	}
	for i, route := range routes {
		routes[i].Path = baseUrl + route.Path
	}
	return routes
}

func (e *ObjectiveController) createObjectiveHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		category_id, err := strconv.Atoi(c.Param("category_id"))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		var objectiveCreate ObjectiveCreate
		if err := c.BindJSON(&objectiveCreate); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		objective, err := e.service.CreateObjective(category_id, objectiveCreate.toModel())
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(404, gin.H{"error": "Category not found"})
			} else {
				c.JSON(500, gin.H{"error": err.Error()})
			}
			return
		}
		c.JSON(201, toObjectiveResponse(objective))
	}
}

func (e *ObjectiveController) deleteObjectiveHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		objective_id, err := strconv.Atoi(c.Param("objective_id"))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		err = e.service.DeleteObjective(objective_id)
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

func (e *ObjectiveController) updateObjectiveHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		objective_id, err := strconv.Atoi(c.Param("objective_id"))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		var objectiveCreate ObjectiveUpdate
		if err := c.BindJSON(&objectiveCreate); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		objective, err := e.service.UpdateObjective(objective_id, objectiveCreate.toModel())
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

func (e *ObjectiveController) getCategoryObjectivesHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		category_id, err := strconv.Atoi(c.Param("category_id"))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		objectives, err := e.service.GetObjectivesByCategoryId(category_id)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(404, gin.H{"error": "Category not found"})
			} else {
				c.JSON(500, gin.H{"error": err.Error()})
			}
			return
		}
		c.JSON(200, Map(objectives, toObjectiveResponse))
	}
}

type ObjectiveCreate struct {
	Name           string                   `json:"name" binding:"required"`
	RequiredNumber int                      `json:"required_number" binding:"required"`
	ObjectiveType  repository.ObjectiveType `json:"objective_type" binding:"required"`
	ValidFrom      time.Time                `json:"valid_from" binding:"omitempty"`
	ValidTo        time.Time                `json:"valid_to" binding:"omitempty"`
	Conditions     []ConditionCreate        `json:"conditions"`
}
type ObjectiveUpdate struct {
	Name           string                   `json:"name"`
	RequiredNumber int                      `json:"required_number"`
	ObjectiveType  repository.ObjectiveType `json:"objective_type"`
	ValidFrom      time.Time                `json:"valid_from"`
	ValidTo        time.Time                `json:"valid_to"`
}

type ObjectiveResponse struct {
	ID             int                      `json:"id"`
	Name           string                   `json:"name"`
	RequiredNumber int                      `json:"required_number"`
	CategoryID     int                      `json:"category_id"`
	ObjectiveType  repository.ObjectiveType `json:"objective_type"`
	ValidFrom      time.Time                `json:"valid_from" binding:"omitempty"`
	ValidTo        time.Time                `json:"valid_to" binding:"omitempty"`
	Contitions     []ConditionOut           `json:"conditions"`
}

func (e *ObjectiveCreate) toModel() *repository.Objective {
	return &repository.Objective{
		Name:           e.Name,
		RequiredNumber: e.RequiredNumber,
		ObjectiveType:  e.ObjectiveType,
		Conditions:     Map(e.Conditions, func(c ConditionCreate) *repository.Condition { return c.toModel() }),
		ValidFrom:      &e.ValidFrom,
		ValidTo:        &e.ValidTo,
	}
}

func (e *ObjectiveUpdate) toModel() *repository.Objective {
	return &repository.Objective{
		Name:           e.Name,
		RequiredNumber: e.RequiredNumber,
		ObjectiveType:  e.ObjectiveType,
		ValidFrom:      &e.ValidFrom,
		ValidTo:        &e.ValidTo,
	}
}

func toObjectiveResponse(objective *repository.Objective) ObjectiveResponse {
	return ObjectiveResponse{
		ID:             objective.ID,
		Name:           objective.Name,
		RequiredNumber: objective.RequiredNumber,
		CategoryID:     objective.CategoryID,
		ObjectiveType:  objective.ObjectiveType,
		ValidFrom:      *objective.ValidFrom,
		ValidTo:        *objective.ValidTo,
		Contitions:     Map(objective.Conditions, toConditionResponse),
	}
}
