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

func setupScoringCategoryController(db *gorm.DB) []gin.RouteInfo {
	e := NewScoringCategoryController(db)
	routes := []gin.RouteInfo{
		{Method: "GET", Path: "/events/:event_id/rules", HandlerFunc: e.getRulesForEventHandler()},
		{Method: "GET", Path: "/scoring-categories/:category_id", HandlerFunc: e.getScoringCategoryHandler()},
		{Method: "POST", Path: "/scoring-categories/:category_id", HandlerFunc: e.createCategoryHandler()},
		{Method: "PATCH", Path: "/scoring-categories/:category_id", HandlerFunc: e.updateCategoryHandler()},
		{Method: "DELETE", Path: "/scoring-categories/:category_id", HandlerFunc: e.deleteCategoryHandler()}}
	return routes
}

func (e *ScoringCategoryController) getRulesForEventHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		event_id, err := strconv.Atoi(c.Param("event_id"))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		rules, err := e.service.GetRulesForEvent(event_id)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, toCategoryResponse(rules))
	}
}

func (e *ScoringCategoryController) getScoringCategoryHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		category_id, err := strconv.Atoi(c.Param("category_id"))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		category, err := e.service.GetCategoryById(category_id)
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
		category_id, err := strconv.Atoi(c.Param("category_id"))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		var categoryCreate CategoryCreate
		if err := c.BindJSON(&categoryCreate); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		category, err := e.service.CreateCategory(category_id, categoryCreate.toModel())
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

func (e *ScoringCategoryController) updateCategoryHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		category_id, err := strconv.Atoi(c.Param("category_id"))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		var categoryUpdate CategoryUpdate
		if err := c.BindJSON(&categoryUpdate); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		categoryModel := categoryUpdate.toModel()
		categoryModel.ID = category_id
		category, err := e.service.UpdateCategory(categoryModel)
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

func (e *ScoringCategoryController) deleteCategoryHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		category_id, err := strconv.Atoi(c.Param("category_id"))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		err = e.service.DeleteCategory(category_id)
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
	Name        string                              `json:"name" binding:"required"`
	Inheritance repository.ScoringMethodInheritance `json:"inheritance" binding:"required"`
}

type CategoryUpdate struct {
	Name        string                              `json:"name"`
	Inheritance repository.ScoringMethodInheritance `json:"inheritance"`
}

type CategoryResponse struct {
	ID             int                                 `json:"id"`
	Name           string                              `json:"name"`
	Inheritance    repository.ScoringMethodInheritance `json:"inheritance"`
	SubCategories  []CategoryResponse                  `json:"sub_categories"`
	Objectives     []ObjectiveResponse                 `json:"objectives"`
	ScoringMethods []ScoringMethodResponse             `json:"scoring_methods"`
}

func (e *CategoryCreate) toModel() *repository.ScoringCategory {
	return &repository.ScoringCategory{
		Name:        e.Name,
		Inheritance: e.Inheritance,
	}
}

func (e *CategoryUpdate) toModel() *repository.ScoringCategory {
	return &repository.ScoringCategory{
		Name:        e.Name,
		Inheritance: e.Inheritance,
	}
}

type ScoringMethodResponse struct {
	Type   repository.ScoringMethodType `json:"type"`
	Points []int                        `json:"points"`
}

func toCategoryResponse(category *repository.ScoringCategory) CategoryResponse {
	return CategoryResponse{
		ID:             category.ID,
		Name:           category.Name,
		Inheritance:    category.Inheritance,
		SubCategories:  utils.Map(category.SubCategories, toCategoryResponse),
		Objectives:     utils.Map(category.Objectives, toObjectiveResponse),
		ScoringMethods: utils.Map(category.ScoringMethods, toScoringMethodResponse),
	}
}

func toScoringMethodResponse(scoringMethod *repository.ScoringMethod) ScoringMethodResponse {
	return ScoringMethodResponse{
		Type:   scoringMethod.Type,
		Points: scoringMethod.Points,
	}
}
