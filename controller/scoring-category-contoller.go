package controller

import (
	"bpl/repository"
	"bpl/service"
	"bpl/utils"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ScoringCategoryController struct {
	service *service.ScoringCategoryService
}

func NewScoringCategoryController(db *gorm.DB) *ScoringCategoryController {
	return &ScoringCategoryController{service: service.NewScoringCategoryService(db)}
}

func setupScoringCategoryController(db *gorm.DB) []RouteInfo {
	e := NewScoringCategoryController(db)
	routes := []RouteInfo{
		{Method: "GET", Path: "/events/:event_id/rules", HandlerFunc: e.getRulesForEventHandler()},
		{Method: "PUT", Path: "/scoring/categories", HandlerFunc: e.createCategoryHandler(), Authenticated: true, RequiredRoles: []repository.Permission{repository.PermissionAdmin}},
		{Method: "GET", Path: "/scoring/categories/:id", HandlerFunc: e.getScoringCategoryHandler()},
		{Method: "DELETE", Path: "/scoring/categories/:id", HandlerFunc: e.deleteCategoryHandler(), Authenticated: true, RequiredRoles: []repository.Permission{repository.PermissionAdmin}}}
	return routes
}

func (e *ScoringCategoryController) getRulesForEventHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		event_id, err := strconv.Atoi(c.Param("event_id"))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		rules, err := e.service.GetRulesForEvent(event_id, "Objectives", "Objectives.Conditions")
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, toCategoryResponse(rules))
	}
}

func (e *ScoringCategoryController) getScoringCategoryHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		category, err := e.service.GetCategoryById(id)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(404, gin.H{"error": "Category not found"})
			} else {
				c.JSON(500, gin.H{"error": err.Error()})
			}
			return
		}
		c.JSON(200, toCategoryResponse(category))
	}
}

func (e *ScoringCategoryController) createCategoryHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var categoryCreate CategoryCreate
		if err := c.BindJSON(&categoryCreate); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		category, err := e.service.CreateCategory(categoryCreate.toModel())
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(404, gin.H{"error": "Parent category not found"})
			} else {
				c.JSON(500, gin.H{"error": err.Error()})
			}
			return
		}
		c.JSON(201, toCategoryResponse(category))
	}
}

func (e *ScoringCategoryController) deleteCategoryHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		err = e.service.DeleteCategory(id)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(404, gin.H{"error": "Category not found"})
			} else {
				c.JSON(500, gin.H{"error": err.Error()})
			}
			return
		}
		c.JSON(204, nil)
	}
}

type CategoryCreate struct {
	ID        *int   `json:"id"`
	ParentID  int    `json:"parent_id" binding:"required"`
	Name      string `json:"name" binding:"required"`
	ScoringId *int   `json:"scoring_preset_id"`
}

type CategoryResponse struct {
	ID              int                  `json:"id"`
	Name            string               `json:"name"`
	SubCategories   []*CategoryResponse  `json:"sub_categories"`
	Objectives      []*ObjectiveResponse `json:"objectives"`
	ScoringPresetID *int                 `json:"scoring_preset_id"`
}

func (e *CategoryCreate) toModel() *repository.ScoringCategory {
	category := &repository.ScoringCategory{
		ParentID:  &e.ParentID,
		Name:      e.Name,
		ScoringId: e.ScoringId,
	}
	if e.ID != nil {
		category.ID = *e.ID
	}
	return category
}

type ScoringMethodResponse struct {
	Type   repository.ScoringMethod `json:"type"`
	Points []int                    `json:"points"`
}

func toCategoryResponse(category *repository.ScoringCategory) *CategoryResponse {
	if category == nil {
		return nil
	}
	return &CategoryResponse{
		ID:              category.ID,
		Name:            category.Name,
		SubCategories:   utils.Map(category.SubCategories, toCategoryResponse),
		Objectives:      utils.Map(category.Objectives, toObjectiveResponse),
		ScoringPresetID: category.ScoringId,
	}
}
