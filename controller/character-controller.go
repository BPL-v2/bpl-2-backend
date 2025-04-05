package controller

import (
	"bpl/repository"
	"bpl/service"
	"bpl/utils"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type CharacterController struct {
	characterService *service.CharacterService
}

func NewCharacterController() *CharacterController {
	return &CharacterController{
		characterService: service.NewCharacterService(),
	}
}

func setupCharacterController() []RouteInfo {
	e := NewCharacterController()
	basePath := "characters"
	routes := []RouteInfo{
		{Method: "GET", Path: "/:user_id", HandlerFunc: e.getUserCharactersHandler()},
		{Method: "GET", Path: "/:user_id/:event_id", HandlerFunc: e.getCharacterEventHistoryForUser()},
	}
	for i, route := range routes {
		routes[i].Path = basePath + route.Path
	}
	return routes
}

// @id GetUserCharacters
// @Description Fetches all event characters for a user
// @Tags characters
// @Produce json
// @Param userId path int true "User Id"
// @Success 200 {array} Character
// @Router /characters/{userId} [get]
func (e *CharacterController) getUserCharactersHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		userId, err := strconv.Atoi(c.Param("user_id"))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		characters, err := e.characterService.GetLatestEventCharactersForUser(userId)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, utils.Map(characters, toCharacterResponse))
	}
}

// @id GetCharacterEventHistoryForUser
// @Description Get all character data for an event for a user
// @Tags characters
// @Accept json
// @Produce json
// @Param event_id path int true "Event ID"
// @Param user_id path int true "User ID"
// @Success 200 {array} Character
// @Router /characters/{user_id}/{event_id} [get]
func (c *CharacterController) getCharacterEventHistoryForUser() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		event := getEvent(ctx)
		if event == nil {
			return
		}
		userId, err := strconv.Atoi(ctx.Param("user_id"))
		if err != nil {
			ctx.JSON(400, gin.H{"error": "Invalid user ID"})
			return
		}
		characters, err := c.characterService.GetEventCharacterHistoryForUser(userId, event.Id)
		if err != nil {
			ctx.JSON(500, gin.H{"error": err.Error()})
			return
		}

		ctx.JSON(200, utils.Map(characters, toCharacterResponse))
	}
}

type Character struct {
	UserId           int       `json:"user_id" binding:"required"`
	EventId          int       `json:"event_id" binding:"required"`
	Name             string    `json:"name" binding:"required"`
	Level            int       `json:"level" binding:"required"`
	MainSkill        string    `json:"main_skill" binding:"required"`
	Ascendancy       string    `json:"ascendancy" binding:"required"`
	AscendancyPoints int       `json:"ascendancy_points" binding:"required"`
	AtlasNodeCount   int       `json:"atlas_node_count" binding:"required"`
	Pantheon         bool      `json:"pantheon" binding:"required"`
	Timestamp        time.Time `json:"timestamp" binding:"required"`
}

func toCharacterResponse(character *repository.Character) *Character {
	if character == nil {
		return nil
	}
	return &Character{
		UserId:           character.UserID,
		EventId:          character.EventID,
		Name:             character.Name,
		Level:            character.Level,
		MainSkill:        character.MainSkill,
		Ascendancy:       character.Ascendancy,
		AscendancyPoints: character.AscendancyPoints,
		AtlasNodeCount:   character.AtlasNodeCount,
		Pantheon:         character.Pantheon,
		Timestamp:        character.Timestamp,
	}
}
