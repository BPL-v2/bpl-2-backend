package controller

import (
	"bpl/client"
	"bpl/repository"
	"bpl/service"
	"bpl/utils"
	"time"

	"github.com/gin-gonic/gin"
)

type GuildStashController struct {
	guildStashService *service.GuildStashService
	userService       *service.UserService
}

func NewGuildStashController(PoEClient *client.PoEClient) *GuildStashController {
	return &GuildStashController{
		guildStashService: service.NewGuildStashService(PoEClient),
		userService:       service.NewUserService(),
	}
}

func setupGuildStashController(PoEClient *client.PoEClient) []RouteInfo {
	e := NewGuildStashController(PoEClient)
	basePath := "/:event_id/guild-stash"
	routes := []RouteInfo{
		{Method: "GET", Path: "", HandlerFunc: e.getGuildStashForUser(), Authenticated: true},
		{Method: "POST", Path: "", HandlerFunc: e.updateGuildStash(), Authenticated: true},
		{Method: "PATCH", Path: "/:stash_id", HandlerFunc: e.switchStashFetch(), Authenticated: true},
		{Method: "POST", Path: "/:stash_id/update", HandlerFunc: e.updateStashTab(), Authenticated: true},
		{Method: "GET", Path: "/:stash_id/items", HandlerFunc: e.getGuildStashTabItems(), Authenticated: true},
	}
	for i, route := range routes {
		routes[i].Path = basePath + route.Path
	}
	return routes
}

// @id GetGuildStashForUser
// @Description Fetches all guild stash tabs for a user
// @Tags guild-stash
// @Produce json
// @Security BearerAuth
// @Param eventId path int true "Event Id"
// @Success 200 {array} GuildStashTab
// @Router /{eventId}/guild-stash [get]
func (e *GuildStashController) getGuildStashForUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		event := getEvent(c)
		if event == nil {
			return
		}
		team, _, err := e.userService.GetTeamForUser(c, event)
		if err != nil {
			c.JSON(403, gin.H{"error": err.Error()})
			return
		}
		tabs, err := e.guildStashService.GetGuildStashesForTeam(team.TeamId)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, utils.Map(tabs, toModel))
	}
}

// @id UpdateGuildStash
// @Description Updates the guild stash tabs for a user
// @Tags guild-stash
// @Security BearerAuth
// @Produce json
// @Param eventId path int true "Event Id"
// @Success 200 {array} GuildStashTab
// @Router /{eventId}/guild-stash [post]
func (e *GuildStashController) updateGuildStash() gin.HandlerFunc {
	return func(c *gin.Context) {
		event := getEvent(c)
		if event == nil {
			return
		}
		teamUser, user, err := e.userService.GetTeamForUser(c, event)
		if err != nil || !teamUser.IsTeamLead {
			c.JSON(403, "unauthorized")
			return
		}
		tabs, err := e.guildStashService.UpdateGuildStash(user, teamUser.TeamId, event)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, utils.Map(tabs, toModel))
	}
}

// @id UpdateStashTab
// @Description Fetches current items for specific guild stash tab
// @Tags guild-stash
// @Security BearerAuth
// @Produce json
// @Param eventId path int true "Event Id"
// @Param stash_id path string true "Stash Tab Id"
// @Success 200 {array} client.DisplayItem
// @Router /{eventId}/guild-stash/{stash_id}/update [post]
func (e *GuildStashController) updateStashTab() gin.HandlerFunc {
	return func(c *gin.Context) {
		event := getEvent(c)
		if event == nil {
			return
		}
		teamUser, user, err := e.userService.GetTeamForUser(c, event)
		if err != nil || !teamUser.IsTeamLead {
			c.JSON(403, "unauthorized")
			return
		}
		stashId := c.Param("stash_id")
		tab, err := e.guildStashService.UpdateStashTab(stashId, event, teamUser, user)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.Status(200)
		c.Writer.Header().Set("Content-Type", "application/json")
		c.Writer.Write([]byte(tab.Items))
	}
}

// @id SwitchStashFetching
// @Description Enables fetching for a specific guild stash tab
// @Tags guild-stash
// @Security BearerAuth
// @Produce json
// @Param eventId path int true "Event Id"
// @Param stash_id path string true "Stash Tab Id"
// @Success 200 {object} GuildStashTab
// @Router /{eventId}/guild-stash/{stash_id} [patch]
func (e *GuildStashController) switchStashFetch() gin.HandlerFunc {
	return func(c *gin.Context) {
		event := getEvent(c)
		if event == nil {
			return
		}
		teamUser, _, err := e.userService.GetTeamForUser(c, event)
		if err != nil || !teamUser.IsTeamLead {
			c.JSON(403, "unauthorized")
			return
		}
		stashId := c.Param("stash_id")
		tab, err := e.guildStashService.SwitchStashFetch(stashId, event.Id)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, toModel(tab))
	}
}

// @id GetGuildStashTabItems
// @Description Fetches all items in a specific guild stash tab
// @Tags guild-stash
// @Security BearerAuth
// @Produce json
// @Param eventId path int true "Event Id"
// @Param stash_id path string true "Stash Tab Id"
// @Success 200 {array} client.DisplayItem
// @Router /{eventId}/guild-stash/{stash_id}/items [get]
func (e *GuildStashController) getGuildStashTabItems() gin.HandlerFunc {
	return func(c *gin.Context) {
		event := getEvent(c)
		if event == nil {
			return
		}
		stashId := c.Param("stash_id")
		tab, err := e.guildStashService.GetGuildStash(stashId, event.Id)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.Status(200)
		c.Writer.Header().Set("Content-Type", "application/json")
		c.Writer.Write([]byte(tab.Items))
	}
}

type GuildStashTab struct {
	Id           string    `json:"id" binding:"required"`
	Name         string    `json:"name" binding:"required"`
	Type         string    `json:"type" binding:"required"`
	Index        *int      `json:"index"`
	Color        *string   `json:"color"`
	ParentId     *string   `json:"parent_id"`
	FetchEnabled bool      `json:"fetch_enabled" binding:"required"`
	LastFetch    time.Time `json:"last_fetch" binding:"required"`
}

func toModel(tab *repository.GuildStashTab) *GuildStashTab {
	if tab == nil {
		return nil
	}
	return &GuildStashTab{
		Id:           tab.Id,
		Name:         tab.Name,
		Type:         tab.Type,
		Index:        tab.Index,
		Color:        tab.Color,
		ParentId:     tab.ParentId,
		FetchEnabled: tab.FetchEnabled,
		LastFetch:    tab.LastFetch,
	}
}
