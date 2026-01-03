package controller

import (
	"bpl/repository"
	"bpl/service"
	"bpl/utils"

	"github.com/gin-gonic/gin"
)

type AchievementController struct {
	AchievementService *service.AchievementService
}

func NewAchievementController() *AchievementController {
	return &AchievementController{
		AchievementService: service.NewAchievementService(),
	}
}

func setupAchievementController() []RouteInfo {
	e := NewAchievementController()
	baseUrl := "/achievements"
	routes := []RouteInfo{
		{Method: "GET", Path: "", HandlerFunc: e.getAchievements()},
		{Method: "POST", Path: "", HandlerFunc: e.addAchievement()},
		{Method: "PATCH", Path: "", HandlerFunc: e.updateAchievements()},
	}
	for i, route := range routes {
		routes[i].Path = baseUrl + route.Path
	}
	return routes
}

// @ID getAchievements
// @Summary Get achievements
// @Description Retrieve all user achievements in the system.
// @Tags Achievement
// @Produce json
// @Success 200 {array} Achievement
// @Router /achievements [get]
func (c *AchievementController) getAchievements() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		achievements, err := c.AchievementService.FindAllAchievements()
		if err != nil {
			ctx.JSON(404, gin.H{"error": "Event not found"})
			return
		}
		ctx.JSON(200, utils.Map(achievements, toAchievement))
	}
}

// @ID addAchievement
// @Summary Add achievement
// @Description Add new achievement to the system.
// @Security BearerAuth
// @Tags Achievement
// @Accept json
// @Produce json
// @Param achievements body AchievementCreate true "Achievement to add"
// @Success 201
// @Router /achievements [post]
func (c *AchievementController) addAchievement() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var achievementCreate AchievementCreate
		if err := ctx.ShouldBindJSON(&achievementCreate); err != nil {
			ctx.JSON(400, gin.H{"error": "Invalid request body"})
			return
		}
		achievement := achievementCreate.toAchievement()
		err := c.AchievementService.SaveAchievement(achievementCreate.toAchievement())
		if err != nil {
			ctx.JSON(500, gin.H{"error": "Failed to save achievement"})
			return
		}
		ctx.JSON(201, toAchievement(achievement))
	}
}

// @ID updateAchievements
// @Summary Update achievements
// @Description Update achievements for all users based on their current progress.
// @Security BearerAuth
// @Tags Achievement
// @Produce json
// @Success 204
// @Router /achievements [patch]
func (c *AchievementController) updateAchievements() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		err := c.AchievementService.UpdateAchievements()
		if err != nil {
			ctx.JSON(500, gin.H{"error": "Failed to update achievements"})
			return
		}
		ctx.Status(204)
	}
}

type Achievement struct {
	Name   repository.AchievementName `json:"name" binding:"required"`
	UserId int                        `json:"user_id" binding:"required"`
}

type AchievementCreate struct {
	Name   repository.AchievementName `json:"name" binding:"required"`
	UserId int                        `json:"user_id" binding:"required"`
}

func (t AchievementCreate) toAchievement() *repository.Achievement {
	return &repository.Achievement{
		Name:   t.Name,
		UserId: t.UserId,
	}
}

func toAchievement(a *repository.Achievement) Achievement {
	return Achievement{
		Name:   a.Name,
		UserId: a.UserId,
	}
}
