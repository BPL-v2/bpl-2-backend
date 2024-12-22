package controller

import (
	"bpl/client"
	"bpl/repository"
	"bpl/service"
	"bpl/utils"
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

func setupObjectiveController(db *gorm.DB) []RouteInfo {
	e := NewObjectiveController(db)
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
		c.JSON(200, utils.Map(objectives, toObjectiveResponse))
	}
}

func (e *ObjectiveController) getObjectiveParserHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		category_id, err := strconv.Atoi(c.Param("category_id"))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		parser, err := e.service.GetParser(category_id)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(404, gin.H{"error": "Category not found"})
			} else {
				c.JSON(500, gin.H{"error": err.Error()})
			}
			return
		}
		item := client.Item{
			BaseType: c.Query("baseType"),
			Name:     c.Query("name"),
		}

		c.JSON(200, parser.CheckForCompletions(&item))
	}
}

type ObjectiveCreate struct {
	ID             int                        `json:"id"`
	Name           string                     `json:"name" binding:"required"`
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
	ID             int                        `json:"id"`
	Name           string                     `json:"name"`
	RequiredNumber int                        `json:"required_number"`
	CategoryID     int                        `json:"category_id"`
	ObjectiveType  repository.ObjectiveType   `json:"objective_type"`
	Conditions     []ConditionResponse        `json:"conditions"`
	ValidFrom      *time.Time                 `json:"valid_from" binding:"omitempty"`
	ValidTo        *time.Time                 `json:"valid_to" binding:"omitempty"`
	ScoringPreset  *ScoringPresetResponse     `json:"scoring_preset"`
	NumberField    repository.NumberField     `json:"number_field"`
	Aggregation    repository.AggregationType `json:"aggregation"`
}

func (e *ObjectiveCreate) toModel() *repository.Objective {
	return &repository.Objective{
		ID:             e.ID,
		Name:           e.Name,
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

func toObjectiveResponse(objective *repository.Objective) ObjectiveResponse {
	resp := ObjectiveResponse{
		ID:             objective.ID,
		Name:           objective.Name,
		RequiredNumber: objective.RequiredAmount,
		CategoryID:     objective.CategoryID,
		ObjectiveType:  objective.ObjectiveType,
		ValidFrom:      objective.ValidFrom,
		ValidTo:        objective.ValidTo,
		Conditions:     utils.Map(objective.Conditions, toConditionResponse),
		NumberField:    objective.NumberField,
		Aggregation:    objective.Aggregation,
	}
	if objective.ScoringPreset != nil {
		resp.ScoringPreset = toScoringPresetResponse(objective.ScoringPreset)
	}
	return resp
}
