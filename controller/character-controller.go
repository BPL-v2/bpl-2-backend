package controller

import (
	"bpl/repository"
	"bpl/service"
	"bpl/utils"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
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
	basePath := "users/:user_id/characters"
	routes := []RouteInfo{
		{Method: "GET", Path: "", HandlerFunc: e.getUserCharactersHandler()},
		{Method: "GET", Path: "/:character_id", HandlerFunc: e.getCharacterHistoryHandler()},
		{Method: "GET", Path: "/:character_id/pob", HandlerFunc: e.getPoBExportHandler()},
		// {Method: "GET", Path: "/:user_id/:event_id/:character_name", HandlerFunc: e.getTimeSeries()},
	}
	for i, route := range routes {
		routes[i].Path = basePath + route.Path
	}
	return routes
}

// @id GetCharacterTimeSeries
// @Description Get the time series for a character
// @Tags characters
// @Produce json
// @Param user_id path int true "User ID"
// @Param event_id path int true "Event ID"
// @Param character_name path string true "Character name"
// @Param start query string true "Start time"
// @Param end query string true "End time"
// @Success 200 {object} client.StatValues
// @Router /characters/{user_id}/{event_id}/{character_name} [get]

// func (e *CharacterController) getTimeSeries() gin.HandlerFunc {
// 	return func(c *gin.Context) {
// 		start := c.Request.URL.Query().Get("start")
// 		end := c.Request.URL.Query().Get("end")
// 		if start == "" || end == "" {
// 			c.JSON(400, gin.H{"error": "start and end are required"})
// 			return
// 		}
// 		startTime, err := time.Parse(time.RFC3339, start)
// 		if err != nil {
// 			c.JSON(400, gin.H{"error": "start is invalid"})
// 			return
// 		}
// 		endTime, err := time.Parse(time.RFC3339, end)
// 		if err != nil {
// 			c.JSON(400, gin.H{"error": "end is invalid"})
// 			return
// 		}
// 		characterName := c.Param("character_name")
// 		metrics := []string{
// 			"XP",
// 			"EHP",
// 			"DPS",
// 			"PhysMaxHit",
// 			"EleMaxHit",
// 			"HP",
// 			"Mana",
// 		}
// 		statValues := client.GetCharacterMetrics(characterName, metrics, startTime, endTime)
// 		c.JSON(200, statValues)
// 	}
// }

// @id GetPoBExport
// @Description Get the PoB export for a character at a specific timestamp
// @Tags characters
// @Produce application/json
// @Param user_id path int true "User ID"
// @Param character_id path string true "Character ID"
// @Param timestamp query string false "Timestamp in RFC3339 format"
// @Success 200 {object} PoB
// @Router /users/{user_id}/characters/{character_id}/pob [get]
func (e *CharacterController) getPoBExportHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		timestamp := c.Query("timestamp")
		t := time.Now()

		if timestamp != "" {
			var err error
			t, err = time.Parse(time.RFC3339, timestamp)
			if err != nil {
				c.JSON(400, gin.H{"error": "timestamp is invalid"})
				return
			}
		}
		pob, err := e.characterService.GetPobForIdBeforeTimestamp(c.Param("character_id"), t)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				c.String(404, "PoB export not found for character")
				return
			}
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, toPoBResponse(pob))
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
		characters, err := e.characterService.GetCharactersForUser(userId)
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
	UserId           int    `json:"user_id" binding:"required"`
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
		ExportString: pob.Export,
		Level:        pob.Level,
		Ascendancy:   pob.Ascendancy,
		Mainskill:    pob.MainSkill,
		Timestamp:    pob.Timestamp,
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
