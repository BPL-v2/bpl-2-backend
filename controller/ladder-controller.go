package controller

import (
	"bpl/repository"
	"bpl/service"
	"bpl/utils"
	"time"

	"github.com/gin-gonic/gin"
)

type LadderController struct {
	ladderService    *service.LadderService
	characterService *service.CharacterService
	userService      *service.UserService
	signupService    *service.SignupService
	activityService  *service.ActivityService
}

func NewLadderController() *LadderController {
	return &LadderController{
		ladderService:    service.NewLadderService(),
		characterService: service.NewCharacterService(),
		userService:      service.NewUserService(),
		signupService:    service.NewSignupService(),
		activityService:  service.NewActivityService(),
	}
}

func setupLadderController() []RouteInfo {
	c := NewLadderController()
	baseUrl := "events/:event_id"
	routes := []RouteInfo{
		{Method: "GET", Path: "/ladder", HandlerFunc: c.getLadderHandler()},
		{Method: "GET", Path: "/characters", HandlerFunc: c.GetCharactersForEvent()},
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
		characters, err := c.characterService.GetCharactersForEvent(event.Id)
		if err != nil {
			ctx.JSON(500, gin.H{"error": err.Error()})
			return
		}
		characterStats, err := c.characterService.GetLatestCharacterStatsForEvent(event.Id)
		if err != nil {
			ctx.JSON(500, gin.H{"error": err.Error()})
			return
		}
		lastActivities, err := c.activityService.GetLatestActiveTimestampsForEvent(event.Id)
		if err != nil {
			ctx.JSON(500, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(200, toLadderResponse(ladder, characters, characterStats, lastActivities))
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
func (c *LadderController) GetCharactersForEvent() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		event := getEvent(ctx)
		if event == nil {
			return
		}
		characters, err := c.characterService.GetCharactersForEvent(event.Id)
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
// @Security BearerAuth
// @Param event_id path int true "Event ID"
// @Success 200 {array} Atlas
// @Router /events/{event_id}/atlas [get]
func (c *LadderController) getAtlasesForEvent() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		event := getEvent(ctx)
		if event == nil {
			return
		}
		user, err := c.userService.GetUserFromAuthHeader(ctx)
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
	UserId        *int           `json:"user_id"`
	CharacterName string         `json:"character_name" binding:"required"`
	AccountName   string         `json:"account_name" binding:"required"`
	Level         int            `json:"level" binding:"required"`
	Class         string         `json:"character_class" binding:"required"`
	Experience    int            `json:"experience" binding:"required"`
	Delve         int            `json:"delve" binding:"required"`
	Rank          int            `json:"rank" binding:"required"`
	Character     *Character     `json:"character"`
	Stats         *CharacterStat `json:"stats"`
	TwitchAccount *string        `json:"twitch_account"`
	LastActive    *int64         `json:"last_active"`
}

type Atlas struct {
	UserId  int     `json:"user_id" binding:"required"`
	EventId int     `json:"event_id" binding:"required"`
	Index   int     `json:"index" binding:"required"`
	Trees   [][]int `json:"trees" binding:"required"`
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

func toLadderResponse(entries []*repository.LadderEntry, characters []*repository.Character, stats map[string]*repository.CharacterStat, lastActivities map[int]time.Time) []*LadderEntry {
	response := make([]*LadderEntry, 0, len(entries))
	characterMap := make(map[string]*repository.Character)
	statsMap := make(map[string]*repository.CharacterStat)
	inLadder := make(map[string]bool)
	for _, character := range characters {
		characterMap[character.Name] = character
		statsMap[character.Name] = stats[character.Id]
	}
	for _, entry := range entries {
		inLadder[entry.Character] = true
		responseEntry := &LadderEntry{
			UserId:        entry.UserId,
			CharacterName: entry.Character,
			AccountName:   entry.Account,
			Level:         entry.Level,
			Class:         entry.Class,
			Experience:    entry.Experience,
			Delve:         entry.Delve,
			Rank:          entry.Rank,
			TwitchAccount: entry.TwitchAccount,
			Character:     toCharacterResponse(characterMap[entry.Character]),
			Stats:         toCharacterStatResponse(statsMap[entry.Character]),
		}
		if entry.UserId != nil && lastActivities[*entry.UserId] != (time.Time{}) {
			timestamp := lastActivities[*entry.UserId].Unix()
			responseEntry.LastActive = &timestamp
		}
		response = append(response, responseEntry)
	}

	// for name, character := range characterMap {
	// 	if !inLadder[name] {
	// 		stats := statsMap[name]
	// 		responseEntry := &LadderEntry{
	// 			CharacterName: name,
	// 			AccountName:   "",
	// 			Level:         character.Level,
	// 			Class:         character.Ascendancy,
	// 			Delve:         0,
	// 			Rank:          0,
	// 			TwitchAccount: nil,
	// 			Character:     toCharacterResponse(character),
	// 			Stats:         toCharacterStatResponse(stats),
	// 		}
	// 		if stats != nil {
	// 			responseEntry.Experience = int(stats.XP)
	// 		}
	// 		response = append(response, responseEntry)
	// 	}
	// }

	return response
}
