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
	categoryService *service.ScoringCategoryService
	eventService    *service.EventService
}

func NewScoringCategoryController() *ScoringCategoryController {
	return &ScoringCategoryController{categoryService: service.NewScoringCategoryService(), eventService: service.NewEventService()}
}

func setupScoringCategoryController() []RouteInfo {
	e := NewScoringCategoryController()
	routes := []RouteInfo{
		{Method: "GET", Path: "/events/:event_id/categories", HandlerFunc: e.getRulesForEventHandler()},
		{Method: "PUT", Path: "/events/:event_id/categories", HandlerFunc: e.createCategoryHandler(), Authenticated: true, RequiredRoles: []repository.Permission{repository.PermissionAdmin, repository.PermissionObjectiveDesigner}},
		{Method: "GET", Path: "/events/:event_id/categories/:id", HandlerFunc: e.getScoringCategoryHandler(), Authenticated: true, RequiredRoles: []repository.Permission{repository.PermissionAdmin, repository.PermissionObjectiveDesigner}},
		{Method: "DELETE", Path: "/events/:event_id/categories/:id", HandlerFunc: e.deleteCategoryHandler(), Authenticated: true, RequiredRoles: []repository.Permission{repository.PermissionAdmin, repository.PermissionObjectiveDesigner}},
	}
	return routes
}

// @id GetRulesForEvent
// @Description Fetches the rules for the current event
// @Tags scoring
// @Produce json
// @Param event_id path int true "Event Id"
// @Success 200 {object} Category
// @Router /events/{event_id}/categories [get]
func (e *ScoringCategoryController) getRulesForEventHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		event := getEvent(c)
		if event == nil {
			return
		}
		rules, err := e.categoryService.GetRulesForEvent(event.Id, "Objectives", "Objectives.Conditions")
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, toPublicCategoryResponse(rules))
	}
}

// @id GetScoringCategory
// @Description Fetches a scoring category by id
// @Security BearerAuth
// @Tags scoring
// @Produce json
// @Param event_id path int true "Event Id"
// @Param id path int true "Category Id"
// @Success 200 {object} Category
// @Router /events/{event_id}/categories/{id} [get]
func (e *ScoringCategoryController) getScoringCategoryHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		event := getEvent(c)
		if event == nil {
			return
		}
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		category, err := e.categoryService.GetCategoryById(id, "Objectives", "Objectives.Conditions")
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

// @id CreateCategory
// @Description Creates a new scoring category
// @Security BearerAuth
// @Tags scoring
// @Accept json
// @Produce json
// @Param event_id path int true "Event Id"
// @Param body body CategoryCreate true "Category to create"
// @Success 201 {object} Category
// @Router /events/{event_id}/categories [put]
func (e *ScoringCategoryController) createCategoryHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		event := getEvent(c)
		if event == nil {
			return
		}
		if event.Locked {
			c.JSON(400, gin.H{"error": "event is locked"})
			return
		}

		var categoryCreate CategoryCreate
		if err := c.BindJSON(&categoryCreate); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		category := categoryCreate.toModel()
		category.EventId = event.Id
		category, err := e.categoryService.CreateCategory(category)
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

// @id DeleteCategory
// @Description Deletes a scoring category
// @Security BearerAuth
// @Tags scoring
// @Produce json
// @Param event_id path int true "Event Id"
// @Param id path int true "Category Id"
// @Success 204
// @Router /events/{event_id}/categories/{id} [delete]
func (e *ScoringCategoryController) deleteCategoryHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		event := getEvent(c)
		if event == nil {
			return
		}
		if event.Locked {
			c.JSON(400, gin.H{"error": "event is locked"})
			return
		}
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		category, err := e.categoryService.GetCategoryById(id, "Objectives", "Objectives.Conditions")
		if err != nil {
			c.JSON(404, gin.H{"error": err.Error()})
			return
		}
		if category.EventId != event.Id {
			c.JSON(404, gin.H{"error": "Category not found"})
			return
		}

		err = e.categoryService.DeleteCategory(category)
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
	Id        *int   `json:"id"`
	ParentId  int    `json:"parent_id" binding:"required"`
	Name      string `json:"name" binding:"required"`
	ScoringId *int   `json:"scoring_preset_id"`
}

type Category struct {
	Id              int            `json:"id" binding:"required"`
	Name            string         `json:"name" binding:"required"`
	SubCategories   []*Category    `json:"sub_categories" binding:"required"`
	Objectives      []*Objective   `json:"objectives" binding:"required"`
	ScoringPresetId *int           `json:"scoring_preset_id"`
	ScoringPreset   *ScoringPreset `json:"scoring_preset"`
}

func (e *CategoryCreate) toModel() *repository.ScoringCategory {
	category := &repository.ScoringCategory{
		ParentId:  &e.ParentId,
		Name:      e.Name,
		ScoringId: e.ScoringId,
	}
	if e.Id != nil {
		category.Id = *e.Id
	}
	return category
}

type ScoringMethod struct {
	Type   repository.ScoringMethod `json:"type" binding:"required"`
	Points []int                    `json:"points" binding:"required"`
}

func toCategoryResponse(category *repository.ScoringCategory) *Category {
	if category == nil {
		return nil
	}
	return &Category{
		Id:              category.Id,
		Name:            category.Name,
		SubCategories:   utils.Map(category.SubCategories, toCategoryResponse),
		Objectives:      utils.Map(category.Objectives, toObjectiveResponse),
		ScoringPresetId: category.ScoringId,
	}
}
func toPublicCategoryResponse(category *repository.ScoringCategory) *Category {
	if category == nil {
		return nil
	}
	return &Category{
		Id:              category.Id,
		Name:            category.Name,
		SubCategories:   utils.Map(category.SubCategories, toPublicCategoryResponse),
		Objectives:      utils.Map(category.Objectives, toPublicObjectiveResponse),
		ScoringPresetId: category.ScoringId,
		ScoringPreset:   toScoringPresetResponse(category.ScoringPreset),
	}
}
