package controller

import (
	"bpl/client"
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
		// {Method: "POST", Path: "/:user_id/pob/:character_name", HandlerFunc: e.getPoBExportHandler()},
		{Method: "GET", Path: "/:user_id/:event_id/:character_name", HandlerFunc: e.getTimeSeries()},
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
func (e *CharacterController) getTimeSeries() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := c.Request.URL.Query().Get("start")
		end := c.Request.URL.Query().Get("end")
		if start == "" || end == "" {
			c.JSON(400, gin.H{"error": "start and end are required"})
			return
		}
		startTime, err := time.Parse(time.RFC3339, start)
		if err != nil {
			c.JSON(400, gin.H{"error": "start is invalid"})
			return
		}
		endTime, err := time.Parse(time.RFC3339, end)
		if err != nil {
			c.JSON(400, gin.H{"error": "end is invalid"})
			return
		}
		characterName := c.Param("character_name")
		metrics := []string{
			"XP",
			"EHP",
			"DPS",
			"PhysMaxHit",
			"EleMaxHit",
			"HP",
			"Mana",
		}
		statValues := client.GetCharacterMetrics(characterName, metrics, startTime, endTime)
		c.JSON(200, statValues)
	}
}

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
