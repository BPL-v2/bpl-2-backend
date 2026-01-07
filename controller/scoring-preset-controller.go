package controller

import (
	"bpl/repository"
	"bpl/service"
	"bpl/utils"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ScoringPresetController struct {
	presetService *service.ScoringPresetService
	eventService  *service.EventService
}

func NewScoringPresetController() *ScoringPresetController {
	return &ScoringPresetController{
		presetService: service.NewScoringPresetsService(),
		eventService:  service.NewEventService(),
	}
}

func setupScoringPresetController() []RouteInfo {
	e := NewScoringPresetController()
	routes := []RouteInfo{
		{Method: "GET", Path: "/events/:event_id/scoring-presets", HandlerFunc: e.getScoringPresetsForEventHandler()},
		{Method: "PUT", Path: "/events/:event_id/scoring-presets", HandlerFunc: e.createScoringPresetHandler(), Authenticated: true, RequiredRoles: []repository.Permission{repository.PermissionAdmin, repository.PermissionObjectiveDesigner}},
		{Method: "DELETE", Path: "/events/:event_id/scoring-presets/:id", HandlerFunc: e.deleteScoringPresetHandler(), Authenticated: true, RequiredRoles: []repository.Permission{repository.PermissionAdmin, repository.PermissionObjectiveDesigner}},
	}
	return routes
}

// @id GetScoringPresetsForEvent
// @Description Fetches the scoring presets for the current event
// @Tags scoring
// @Produce json
// @Param event_id path int true "Event Id"
// @Success 200 {array} ScoringPreset
// @Router /events/{event_id}/scoring-presets [get]
func (e *ScoringPresetController) getScoringPresetsForEventHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		event := getEvent(c)
		if event == nil {
			return
		}
		presets, err := e.presetService.GetPresetsForEvent(event.Id)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, utils.Map(presets, toScoringPresetResponse))
	}
}

// @id CreateScoringPreset
// @Description Creates a new scoring preset
// @Tags scoring
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param event_id path int true "Event Id"
// @Param body body ScoringPresetCreate true "Preset to create"
// @Success 200 {object} ScoringPreset
// @Router /events/{event_id}/scoring-presets [put]
func (e *ScoringPresetController) createScoringPresetHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		event := getEvent(c)
		if event == nil {
			return
		}
		if event.Locked {
			c.JSON(400, gin.H{"error": "event is locked"})
			return
		}
		var presetCreate ScoringPresetCreate
		if err := c.ShouldBindJSON(&presetCreate); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		create := presetCreate.toModel()
		create.EventId = event.Id
		preset, err := e.presetService.SavePreset(create)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, toScoringPresetResponse(preset))
	}
}

// @id DeleteScoringPreset
// @Description Deletes a scoring preset by id
// @Security BearerAuth
// @Tags scoring
// @Produce json
// @Param event_id path int true "Event Id"
// @Param id path int true "Preset Id"
// @Success 200
// @Router /events/{event_id}/scoring-presets/{id} [delete]
func (e *ScoringPresetController) deleteScoringPresetHandler() gin.HandlerFunc {
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

		err = e.presetService.DeletePreset(id)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(404, gin.H{"error": "preset not found"})
				return
			}
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{})
	}
}

type ScoringPresetCreate struct {
	Id            *int                     `json:"id"`
	Name          string                   `json:"name" binding:"required"`
	Description   string                   `json:"description"`
	Points        []float64                `json:"points" binding:"required"`
	ScoringMethod repository.ScoringMethod `json:"scoring_method" binding:"required"`
	PointCap      int                      `json:"point_cap"`
	Extra         map[string]string        `json:"extra"`
}

func (e *ScoringPresetCreate) toModel() *repository.ScoringPreset {
	preset := &repository.ScoringPreset{
		Name:          e.Name,
		Description:   e.Description,
		Points:        e.Points,
		ScoringMethod: e.ScoringMethod,
		PointCap:      e.PointCap,
		Extra:         e.Extra,
	}
	if e.Id != nil {
		preset.Id = *e.Id
	}
	return preset
}

type ScoringPreset struct {
	Id            int                      `json:"id" binding:"required"`
	Name          string                   `json:"name" binding:"required"`
	Description   string                   `json:"description" binding:"required"`
	Points        []float64                `json:"points" binding:"required"`
	ScoringMethod repository.ScoringMethod `json:"scoring_method" binding:"required"`
	PointCap      int                      `json:"point_cap"`
	Extra         map[string]string        `json:"extra"`
}

func toScoringPresetResponse(preset *repository.ScoringPreset) *ScoringPreset {
	if preset == nil {
		return nil
	}
	return &ScoringPreset{
		Id:            preset.Id,
		Name:          preset.Name,
		Description:   preset.Description,
		Points:        preset.Points,
		ScoringMethod: preset.ScoringMethod,
		PointCap:      preset.PointCap,
		Extra:         preset.Extra,
	}
}
