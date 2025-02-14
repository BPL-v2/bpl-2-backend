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
	service *service.ScoringPresetsService
}

func NewScoringPresetController() *ScoringPresetController {
	return &ScoringPresetController{
		service: service.NewScoringPresetsService(),
	}
}

func setupScoringPresetController() []RouteInfo {
	e := NewScoringPresetController()
	routes := []RouteInfo{
		{Method: "GET", Path: "/events/:event_id/scoring-presets", HandlerFunc: e.getScoringPresetsForEventHandler()},
		{Method: "PUT", Path: "/scoring/presets", HandlerFunc: e.createScoringPresetHandler(), Authenticated: true, RequiredRoles: []repository.Permission{repository.PermissionAdmin}},
		{Method: "GET", Path: "/scoring/presets/:id", HandlerFunc: e.getScoringPresetHandler(), Authenticated: true, RequiredRoles: []repository.Permission{repository.PermissionAdmin}},
		{Method: "DELETE", Path: "/scoring/presets/:id", HandlerFunc: e.deleteScoringPresetHandler(), Authenticated: true, RequiredRoles: []repository.Permission{repository.PermissionAdmin}},
	}
	return routes
}

// @id GetScoringPresetsForEvent
// @Description Fetches the scoring presets for the current event
// @Tags scoring
// @Produce json
// @Param event_id path int true "Event ID"
// @Success 200 {array} ScoringPresetResponse
// @Router /events/{event_id}/scoring-presets [get]
func (e *ScoringPresetController) getScoringPresetsForEventHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		event_id, err := strconv.Atoi(c.Param("event_id"))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		presets, err := e.service.GetPresetsForEvent(event_id)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, utils.Map(presets, toScoringPresetResponse))
	}
}

// @id GetScoringPreset
// @Description Fetches a scoring preset by id
// @Tags scoring
// @Produce json
// @Param id path int true "Preset ID"
// @Success 200 {object} ScoringPresetResponse
// @Router /scoring/presets/{id} [get]
func (e *ScoringPresetController) getScoringPresetHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		preset, err := e.service.GetPresetById(id)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(404, gin.H{"error": "preset not found"})
				return
			}
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, toScoringPresetResponse(preset))
	}
}

// @id CreateScoringPreset
// @Description Creates a new scoring preset
// @Tags scoring
// @Accept json
// @Produce json
// @Param body body ScoringPresetCreate true "Preset to create"
// @Success 200 {object} ScoringPresetResponse
// @Router /scoring/presets [put]
func (e *ScoringPresetController) createScoringPresetHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var presetCreate ScoringPresetCreate
		if err := c.ShouldBindJSON(&presetCreate); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		preset, err := e.service.SavePreset(presetCreate.toModel())
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, toScoringPresetResponse(preset))
	}
}

// @id DeleteScoringPreset
// @Description Deletes a scoring preset by id
// @Tags scoring
// @Produce json
// @Param id path int true "Preset ID"
// @Success 200
// @Router /scoring/presets/{id} [delete]
func (e *ScoringPresetController) deleteScoringPresetHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		err = e.service.DeletePreset(id)
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
	ID            *int                         `json:"id"`
	Name          string                       `json:"name" binding:"required"`
	Description   string                       `json:"description"`
	Points        []float64                    `json:"points" binding:"required"`
	ScoringMethod repository.ScoringMethod     `json:"scoring_method" binding:"required"`
	Type          repository.ScoringPresetType `json:"type" binding:"required"`
	EventID       int                          `json:"event_id" binding:"required"`
}

func (e *ScoringPresetCreate) toModel() *repository.ScoringPreset {
	preset := &repository.ScoringPreset{
		Name:          e.Name,
		Description:   e.Description,
		Points:        e.Points,
		ScoringMethod: e.ScoringMethod,
		Type:          e.Type,
		EventID:       e.EventID,
	}
	if e.ID != nil {
		preset.ID = *e.ID
	}
	return preset
}

type ScoringPresetResponse struct {
	ID            int                          `json:"id" binding:"required"`
	Name          string                       `json:"name" binding:"required"`
	Description   string                       `json:"description" binding:"required"`
	Points        []float64                    `json:"points" binding:"required"`
	ScoringMethod repository.ScoringMethod     `json:"scoring_method" binding:"required"`
	Type          repository.ScoringPresetType `json:"type" binding:"required"`
}

func toScoringPresetResponse(preset *repository.ScoringPreset) *ScoringPresetResponse {
	if preset == nil {
		return nil
	}
	return &ScoringPresetResponse{
		ID:            preset.ID,
		Name:          preset.Name,
		Description:   preset.Description,
		Points:        preset.Points,
		ScoringMethod: preset.ScoringMethod,
		Type:          preset.Type,
	}
}
