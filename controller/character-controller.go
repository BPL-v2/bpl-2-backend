package controller

import (
	"bpl/client"
	"bpl/cron"
	"bpl/repository"
	"bpl/service"
	"bpl/utils"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type CharacterController struct {
	characterService      *service.CharacterService
	userService           *service.UserService
	playerFetchingService *cron.PlayerFetchingService
}

func NewCharacterController(poeClient *client.PoEClient) *CharacterController {
	return &CharacterController{
		characterService:      service.NewCharacterService(poeClient),
		userService:           service.NewUserService(),
		playerFetchingService: cron.NewPlayerFetchingService(poeClient),
	}
}

func setupCharacterController(poeClient *client.PoEClient) []RouteInfo {
	e := NewCharacterController(poeClient)
	basePath := "users/:user_id/characters"
	routes := []RouteInfo{
		{Method: "GET", Path: "", HandlerFunc: e.getUserCharactersHandler()},
		{Method: "GET", Path: "/:character_id", HandlerFunc: e.getCharacterHistoryHandler()},
		{Method: "PATCH", Path: "/:character_id", HandlerFunc: e.updateCharacterHandler()},
		{Method: "GET", Path: "/:character_id/pobs", HandlerFunc: e.getPoBExportHandler()},
		// {Method: "GET", Path: "/:user_id/:event_id/:character_name", HandlerFunc: e.getTimeSeries()},
	}
	for i, route := range routes {
		routes[i].Path = basePath + route.Path
	}
	go e.characterService.UpdatePoBStats()
	return routes
}

// @id UpdateCharacter
// @Description Update character details
// @Tags characters
// @Produce json
// @Param user_id path int true "User ID"
// @Param character_id path string true "Character ID"
// @Success 200 {object} client.Character
// @Router /users/{user_id}/characters/{character_id} [patch]
func (e *CharacterController) updateCharacterHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		characterId := c.Param("character_id")
		characterInfo, err := e.characterService.GetInfoForCharacter(characterId)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		playerUpdate, err := characterInfo.ToPlayerUpdate()
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		character, err := e.playerFetchingService.UpdateCharacter(playerUpdate, characterInfo.Event)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, character)
	}
}

// @id GetPoBs
// @Description Get all PoB exports for a character
// @Tags characters
// @Produce application/json
// @Param user_id path int true "User ID"
// @Param character_id path string true "Character ID"
// @Success 200 {array} PoB
// @Router /users/{user_id}/characters/{character_id}/pobs [get]
func (e *CharacterController) getPoBExportHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		pobs, err := e.characterService.GetPobs(c.Param("character_id"))
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				c.String(404, "PoB export not found for character")
				return
			}
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, utils.Map(pobs, toPoBResponse))
	}
}

// @id GetUserCharacters
// @Description Fetches all event characters for a user
// @Tags characters
// @Produce json
// @Param user_id path int true "User Id"
// @Success 200 {array} Character
// @Router /users/{user_id}/characters [get]
func (e *CharacterController) getUserCharactersHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		userId, err := strconv.Atoi(c.Param("user_id"))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		user, err := e.userService.GetUserById(userId, "OauthAccounts")
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				c.String(404, "User not found")
				return
			}
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		characters, err := e.characterService.GetCharactersForUser(user)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, utils.Map(characters, toCharacterResponse))
	}
}

// @id GetCharacterHistory
// @Description Get all character data for an event for a user
// @Tags characters
// @Accept json
// @Produce json
// @Param user_id path int true "User ID"
// @Param character_id path string true "Character ID"
// @Success 200 {array} CharacterStat
// @Router /users/{user_id}/characters/{character_id} [get]
func (c *CharacterController) getCharacterHistoryHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		characterId := ctx.Param("character_id")
		stats, err := c.characterService.GetCharacterHistory(characterId)
		if err != nil {
			ctx.JSON(500, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(200, utils.Map(stats, toCharacterStatResponse))
	}
}

type Character struct {
	Id               string `json:"id" binding:"required"`
	UserId           *int   `json:"user_id"`
	EventId          int    `json:"event_id" binding:"required"`
	Name             string `json:"name" binding:"required"`
	Level            int    `json:"level" binding:"required"`
	MainSkill        string `json:"main_skill" binding:"required"`
	Ascendancy       string `json:"ascendancy" binding:"required"`
	AscendancyPoints int    `json:"ascendancy_points" binding:"required"`
	AtlasNodeCount   int    `json:"atlas_node_count" binding:"required"`
	Pantheon         bool   `json:"pantheon" binding:"required"`
}

type PoB struct {
	ExportString string    `json:"export_string" binding:"required"`
	Level        int       `json:"level" binding:"required"`
	Ascendancy   string    `json:"ascendancy" binding:"required"`
	Mainskill    string    `json:"main_skill" binding:"required"`
	Timestamp    time.Time `json:"timestamp" binding:"required"`
}

func toPoBResponse(pob *repository.CharacterPob) *PoB {
	if pob == nil {
		return nil
	}
	return &PoB{
		ExportString: pob.Export.ToString(),
		Level:        pob.Level,
		Ascendancy:   pob.Ascendancy,
		Mainskill:    pob.MainSkill,
		Timestamp:    pob.CreatedAt,
	}
}

type CharacterStat struct {
	TimeStamp     int   `json:"timestamp" binding:"required"`
	DPS           int64 `json:"dps" binding:"required"`
	EHP           int32 `json:"ehp" binding:"required"`
	PhysMaxHit    int32 `json:"phys_max_hit" binding:"required"`
	EleMaxHit     int32 `json:"ele_max_hit" binding:"required"`
	HP            int32 `json:"hp" binding:"required"`
	Mana          int32 `json:"mana" binding:"required"`
	ES            int32 `json:"es" binding:"required"`
	Armour        int32 `json:"armour" binding:"required"`
	Evasion       int32 `json:"evasion" binding:"required"`
	XP            int64 `json:"xp" binding:"required"`
	MovementSpeed int32 `json:"movement_speed" binding:"required"`
}

func toCharacterResponse(character *repository.Character) *Character {
	if character == nil {
		return nil
	}
	return &Character{
		Id:               character.Id,
		UserId:           character.UserId,
		EventId:          character.EventId,
		Name:             character.Name,
		Level:            character.Level,
		MainSkill:        character.MainSkill,
		Ascendancy:       character.Ascendancy,
		AscendancyPoints: character.AscendancyPoints,
		AtlasNodeCount:   character.AtlasPoints,
		Pantheon:         character.Pantheon,
	}
}

func toCharacterStatResponse(characterStat *repository.CharacterStat) *CharacterStat {
	if characterStat == nil {
		return nil
	}
	return &CharacterStat{
		TimeStamp:     int(characterStat.Time.Unix()),
		DPS:           characterStat.DPS,
		EHP:           characterStat.EHP,
		PhysMaxHit:    characterStat.PhysMaxHit,
		EleMaxHit:     characterStat.EleMaxHit,
		HP:            characterStat.HP,
		Mana:          characterStat.Mana,
		ES:            characterStat.ES,
		Armour:        characterStat.Armour,
		Evasion:       characterStat.Evasion,
		XP:            characterStat.XP,
		MovementSpeed: characterStat.MovementSpeed,
	}
}
