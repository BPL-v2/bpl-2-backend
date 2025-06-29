package controller

import (
	"bpl/repository"
	"bpl/service"
	"bpl/utils"
	"strconv"

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
	basePath := "users/:user_id/characters"
	routes := []RouteInfo{
		{Method: "GET", Path: "", HandlerFunc: e.getUserCharactersHandler()},
		{Method: "GET", Path: "/:character_id", HandlerFunc: e.getCharacterHistoryHandler()},
		// {Method: "POST", Path: "/:user_id/pob/:character_name", HandlerFunc: e.getPoBExportHandler()},
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

// func (e *CharacterController) getPoBExportHandler() gin.HandlerFunc {
// 	return func(c *gin.Context) {
// 		fmt.Println("getPoBExportHandler")
// 		userId, err := strconv.Atoi(c.Param("user_id"))
// 		if err != nil {
// 			c.JSON(400, gin.H{"error": err.Error()})
// 			return
// 		}
// 		user, err := service.NewUserService().GetUserById(userId, "OauthAccounts")
// 		if err != nil {
// 			c.JSON(500, gin.H{"error": err.Error()})
// 			return
// 		}
// 		characterName := c.Param("character_name")
// 		poeClient := client.NewPoEClient(100, true, 100)
// 		token := user.GetPoEToken()
// 		if token == "" {
// 			c.JSON(400, gin.H{"error": "User does not have a PoE token"})
// 			return
// 		}
// 		character, httpError := poeClient.GetCharacter(token, characterName, nil)
// 		if httpError != nil {
// 			c.JSON(httpError.StatusCode, gin.H{"error": httpError.Error})
// 			return
// 		}
// 		pob, export, err := client.GetPoBExport(character.Character)
// 		armourGauge.WithLabelValues(character.Character.Name).Set(pob.Build.PlayerStats.Armour)
// 		evasionGauge.WithLabelValues(character.Character.Name).Set(pob.Build.PlayerStats.Evasion)
// 		energyShieldGauge.WithLabelValues(character.Character.Name).Set(pob.Build.PlayerStats.EnergyShield)
// 		ehpGauge.WithLabelValues(character.Character.Name).Set(pob.Build.PlayerStats.TotalEHP)
// 		hpGauge.WithLabelValues(character.Character.Name).Set(pob.Build.PlayerStats.Life)
// 		manaGauge.WithLabelValues(character.Character.Name).Set(pob.Build.PlayerStats.Mana)
// 		physMaxHitGauge.WithLabelValues(character.Character.Name).Set(pob.Build.PlayerStats.PhysicalMaximumHitTaken)
// 		eleMaxHitGauge.WithLabelValues(character.Character.Name).Set(utils.Max(pob.Build.PlayerStats.LightningMaximumHitTaken, pob.Build.PlayerStats.FireMaximumHitTaken, pob.Build.PlayerStats.ColdMaximumHitTaken))
// 		if err != nil {
// 			c.JSON(500, gin.H{"error": err.Error()})
// 			return
// 		}
// 		c.JSON(200, gin.H{"pob": pob, "export": export})
// 	}
// }

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
// @Param character_id path int true "Character ID"
// @Success 200 {array} CharacterStat
// @Router /users/{user_id}/characters/{character_id} [get]
func (c *CharacterController) getCharacterHistoryHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		characterId, err := strconv.Atoi(ctx.Param("character_id"))
		if err != nil {
			ctx.JSON(400, gin.H{"error": "Invalid character ID"})
			return
		}
		stats, err := c.characterService.GetCharacterHistory(characterId)
		if err != nil {
			ctx.JSON(500, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(200, utils.Map(stats, toCharacterStatResponse))
	}
}

type Character struct {
	Id               int    `json:"id" binding:"required"`
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

type CharacterStat struct {
	TimeStamp  int `json:"timestamp" binding:"required"`
	DPS        int `json:"dps" binding:"required"`
	EHP        int `json:"ehp" binding:"required"`
	PhysMaxHit int `json:"phys_max_hit" binding:"required"`
	EleMaxHit  int `json:"ele_max_hit" binding:"required"`
	HP         int `json:"hp" binding:"required"`
	Mana       int `json:"mana" binding:"required"`
	ES         int `json:"es" binding:"required"`
	Armour     int `json:"armour" binding:"required"`
	Evasion    int `json:"evasion" binding:"required"`
	XP         int `json:"xp" binding:"required"`
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
		TimeStamp:  int(characterStat.Time.Unix()),
		DPS:        characterStat.DPS,
		EHP:        characterStat.EHP,
		PhysMaxHit: characterStat.PhysMaxHit,
		EleMaxHit:  characterStat.EleMaxHit,
		HP:         characterStat.HP,
		Mana:       characterStat.Mana,
		ES:         characterStat.ES,
		Armour:     characterStat.Armour,
		Evasion:    characterStat.Evasion,
		XP:         characterStat.XP,
	}
}
