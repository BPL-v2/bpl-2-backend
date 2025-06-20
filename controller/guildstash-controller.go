package controller

import (
	"bpl/client"
	"bpl/cron"
	"bpl/parser"
	"bpl/repository"
	"bpl/service"
	"bpl/utils"
	"context"
	"encoding/json"
	"time"

	"github.com/gin-gonic/gin"
)

type GuildStashController struct {
	guildStashService *service.GuildStashService
	userService       *service.UserService
	objectiveService  *service.ObjectiveService
	poeClient         *client.PoEClient
}

func NewGuildStashController(PoEClient *client.PoEClient) *GuildStashController {
	return &GuildStashController{
		guildStashService: service.NewGuildStashService(PoEClient),
		userService:       service.NewUserService(),
		objectiveService:  service.NewObjectiveService(),
		poeClient:         PoEClient,
	}
}

func setupGuildStashController(PoEClient *client.PoEClient) []RouteInfo {
	e := NewGuildStashController(PoEClient)
	basePath := "/:event_id/guild-stash"
	routes := []RouteInfo{
		{Method: "GET", Path: "", HandlerFunc: e.getGuildStashForUser(), Authenticated: true},
		{Method: "POST", Path: "", HandlerFunc: e.updateGuildStash(), Authenticated: true},
		{Method: "GET", Path: "/:stash_id", HandlerFunc: e.getGuildStashTab(), Authenticated: true},
		{Method: "PATCH", Path: "/:stash_id", HandlerFunc: e.switchStashFetch(), Authenticated: true},
		{Method: "POST", Path: "/:stash_id/update", HandlerFunc: e.updateStashTab(), Authenticated: true},
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
// @Success 204
// @Router /{eventId}/guild-stash/{stash_id}/update [post]
func (e *GuildStashController) updateStashTab() gin.HandlerFunc {
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
		tab, err := e.guildStashService.GetGuildStash(stashId, event.Id)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		if tab.TeamId != teamUser.TeamId || !teamUser.IsTeamLead {
			c.JSON(403, gin.H{"error": "unauthorized to update stash tab"})
			return
		}
		ctx, _ := context.WithTimeout(context.Background(), 1*time.Minute)
		err = cron.NewFetchingService(ctx, event, e.poeClient).FetchGuildStashTab(tab)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.Status(204)
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

// @id GetGuildStashTab
// @Description Fetches a specific guild stash tab
// @Tags guild-stash
// @Security BearerAuth
// @Produce json
// @Param eventId path int true "Event Id"
// @Param stash_id path string true "Stash Tab Id"
// @Success 200 {object} client.GuildStashTabGGG
// @Router /{eventId}/guild-stash/{stash_id}  [get]
func (e *GuildStashController) getGuildStashTab() gin.HandlerFunc {
	return func(c *gin.Context) {
		event := getEvent(c)
		if event == nil {
			return
		}
		stashId := c.Param("stash_id")
		tab, err := e.guildStashService.GetGuildStash(stashId, event.Id, "Children")
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		if tab.Raw == "" || tab.Raw == "{}" {
			c.JSON(404, gin.H{"error": "stash tab not found"})
			return
		}
		parser, err := e.objectiveService.GetParser(event.Id)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.Status(200)
		c.JSON(200, toGGGModel(tab, parser))
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

func toGGGModel(tab *repository.GuildStashTab, parser *parser.ItemChecker) *client.GuildStashTabGGG {
	if tab == nil {
		return nil
	}
	model := &client.GuildStashTabGGG{}
	err := json.Unmarshal([]byte(tab.Raw), &model)
	if err != nil {
		return nil
	}

	model.Name = tab.Name
	if model.Items != nil {
		items := make([]client.DisplayItem, 0, len(*model.Items))
		for _, item := range *model.Items {
			completions := parser.CheckForCompletions(item.Item)
			if len(completions) > 0 {
				item.ObjectiveId = completions[0].ObjectiveId
			}
			items = append(items, item)
		}
		model.Items = &items
	}

	children := make([]client.GuildStashTabGGG, 0, len(tab.Children))
	for _, child := range tab.Children {
		childModel := toGGGModel(child, parser)
		if childModel != nil {
			children = append(children, *childModel)
		}
	}
	model.Children = &children
	return model

}
