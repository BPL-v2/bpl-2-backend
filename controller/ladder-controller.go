package controller

import (
	"bpl/repository"
	"bpl/service"
	"bpl/utils"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type LadderController struct {
	ladderService    *service.LadderService
	characterService *service.CharacterService
	userService      *service.UserService
}

func NewLadderController() *LadderController {
	return &LadderController{
		ladderService:    service.NewLadderService(),
		characterService: service.NewCharacterService(),
		userService:      service.NewUserService(),
	}
}

func setupLadderController() []RouteInfo {
	c := NewLadderController()
	baseUrl := "events/:event_id"
	routes := []RouteInfo{
		{Method: "GET", Path: "/ladder", HandlerFunc: c.getLadderHandler()},
		{Method: "GET", Path: "/characters", HandlerFunc: c.getLatestCharactersForEvent()},
		{Method: "GET", Path: "/characters/:user_id", HandlerFunc: c.getCharacterEventHistoryForUser()},
		{Method: "GET", Path: "/atlas", HandlerFunc: c.getAtlasesForEvent(), Authenticated: true},
	}
	for i, route := range routes {
		routes[i].Path = baseUrl + route.Path
	}
	return routes
}

// @id GetLadder
// @Description Get the ladder for an event
// @Tags ladder
// @Accept json
// @Produce json
// @Param event_id path int true "Event ID"
// @Success 200 {array} LadderEntry
// @Router /events/{event_id}/ladder [get]
func (c *LadderController) getLadderHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		event := getEvent(ctx)
		if event == nil {
			return
		}
		ladder, err := c.ladderService.GetLadderForEvent(event.Id)
		if err != nil {
			ctx.JSON(500, gin.H{"error": err.Error()})
			return
		}
		characters, err := c.characterService.GetLatestCharactersForEvent(event.Id)
		if err != nil {
			ctx.JSON(500, gin.H{"error": err.Error()})
			return
		}

		ctx.JSON(200, toLadderResponse(ladder, characters))
	}
}

// @id GetCharactersForEvent
// @Description Get all characters for an event
// @Tags characters
// @Accept json
// @Produce json
// @Param event_id path int true "Event ID"
// @Success 200 {array} Character
// @Router /events/{event_id}/characters [get]
func (c *LadderController) getLatestCharactersForEvent() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		event := getEvent(ctx)
		if event == nil {
			return
		}
		characters, err := c.characterService.GetLatestCharactersForEvent(event.Id)
		if err != nil {
			ctx.JSON(500, gin.H{"error": err.Error()})
			return
		}

		ctx.JSON(200, utils.Map(characters, toCharacterResponse))
	}
}

// @id GetCharacterEventHistoryForUser
// @Description Get all characters for an event for a user
// @Tags characters
// @Accept json
// @Produce json
// @Param event_id path int true "Event ID"
// @Param user_id path int true "User ID"
// @Success 200 {array} Character
// @Router /events/{event_id}/characters/{user_id} [get]
func (c *LadderController) getCharacterEventHistoryForUser() gin.HandlerFunc {
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

// @id GetTeamAtlasesForEvent
// @Description Get atlas trees for your team for an event
// @Tags atlas
// @Accept json
// @Produce json
// @Param event_id path int true "Event ID"
// @Success 200 {array} Atlas
// @Router /events/{event_id}/atlas [get]
func (c *LadderController) getAtlasesForEvent() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		event := getEvent(ctx)
		if event == nil {
			return
		}
		user, err := c.userService.GetUserFromAuthCookie(ctx)
		if err != nil {
			ctx.JSON(401, gin.H{"error": "Not authenticated"})
			return
		}
		atlases, err := c.characterService.GetTeamAtlasesForEvent(event.Id, user.Id)
		if err != nil {
			ctx.JSON(500, gin.H{"error": err.Error()})
			return
		}

		ctx.JSON(200, utils.Map(atlases, toAtlasResponse))
	}
}

type LadderEntry struct {
	UserId        int        `json:"user_id" binding:"required"`
	CharacterName string     `json:"character_name" binding:"required"`
	AccountName   string     `json:"account_name" binding:"required"`
	Level         int        `json:"level" binding:"required"`
	Class         string     `json:"character_class" binding:"required"`
	Experience    int        `json:"experience" binding:"required"`
	Delve         int        `json:"delve" binding:"required"`
	Rank          int        `json:"rank" binding:"required"`
	Extra         *Character `json:"extra"`
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

type Atlas struct {
	UserId  int     `json:"user_id" binding:"required"`
	EventId int     `json:"event_id" binding:"required"`
	Index   int     `json:"index" binding:"required"`
	Trees   [][]int `json:"trees" binding:"required"`
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

func toAtlasResponse(atlas *repository.Atlas) *Atlas {
	if atlas == nil {
		return nil
	}
	response := &Atlas{
		UserId:  atlas.UserID,
		EventId: atlas.EventID,
		Index:   atlas.Index,
		Trees:   [][]int{},
	}
	response.Trees = append(response.Trees, utils.Map(atlas.Tree1, func(hash int32) int { return int(hash) }))
	response.Trees = append(response.Trees, utils.Map(atlas.Tree2, func(hash int32) int { return int(hash) }))
	response.Trees = append(response.Trees, utils.Map(atlas.Tree3, func(hash int32) int { return int(hash) }))
	return response
}

func toLadderResponse(entries []*repository.LadderEntry, characters []*repository.Character) []*LadderEntry {
	response := make([]*LadderEntry, 0, len(entries))
	characterMap := make(map[int]*repository.Character)
	for _, character := range characters {
		characterMap[character.UserID] = character
	}
	for _, entry := range entries {
		character := characterMap[entry.UserId]
		responseEntry := &LadderEntry{
			UserId:        entry.UserId,
			CharacterName: entry.Character,
			AccountName:   entry.Account,
			Level:         entry.Level,
			Class:         entry.Class,
			Experience:    entry.Experience,
			Delve:         entry.Delve,
			Rank:          entry.Rank,
		}
		if character != nil {
			responseEntry.Extra = toCharacterResponse(character)
		}
		response = append(response, responseEntry)
	}

	return response
}
