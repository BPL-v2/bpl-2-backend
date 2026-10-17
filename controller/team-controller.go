package controller

import (
	"bpl/apperr"
	"bpl/client"
	"bpl/repository"
	"bpl/service"
	"bpl/utils"
	"errors"
	"io"
	"log/slog"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type TeamController struct {
	teamService  service.TeamService
	eventService service.EventService
}

func NewTeamController() *TeamController {
	return &TeamController{
		teamService:  service.NewTeamService(),
		eventService: service.NewEventService(),
	}
}

func toTeamResponse(team *repository.Team) *Team {
	iconUrl := ""
	if team.IconMimeType != nil {
		iconUrl = "/events/" + strconv.Itoa(team.EventId) + "/teams/" + strconv.Itoa(team.Id) + "/icon"
		if team.IconUpdatedAt != nil {
			// version the URL so a re-upload bypasses the browser cache
			iconUrl += "?v=" + strconv.FormatInt(team.IconUpdatedAt.UnixMilli(), 10)
		}
	}
	return &Team{
		IconUrl:        iconUrl,
		Id:             team.Id,
		Name:           team.Name,
		Abbreviation:   team.Abbreviation,
		AllowedClasses: team.AllowedClasses,
		EventId:        team.EventId,
		Color:          team.Color,
		DiscordRoleId:  team.DiscordRoleId,
	}
}

func setupTeamController() []RouteInfo {
	e := NewTeamController()
	basePath := "events/:event_id/teams"
	routes := []RouteInfo{
		{Method: "GET", Path: "", HandlerFunc: e.getTeamsHandler()},
		{Method: "PUT", Path: "", HandlerFunc: e.createTeamHandler(), Authenticated: true, RequiredRoles: []repository.Permission{repository.PermissionAdmin}},
		{Method: "GET", Path: "/users", HandlerFunc: e.getSortedUsersHandler(), Authenticated: true},
		{Method: "PUT", Path: "/users", HandlerFunc: e.addUsersToTeamsHandler(), Authenticated: true, RequiredRoles: []repository.Permission{repository.PermissionAdmin, repository.PermissionManager}},
		{Method: "GET", Path: "/:team_id", HandlerFunc: e.getTeamHandler()},
		{Method: "PUT", Path: "/:team_id/icon", HandlerFunc: e.uploadIconHandler(), Authenticated: true, RequiredRoles: []repository.Permission{repository.PermissionAdmin, repository.PermissionObjectiveDesigner}},
		{Method: "GET", Path: "/:team_id/icon", HandlerFunc: e.getIconHandler()},
		{Method: "DELETE", Path: "/:team_id/icon", HandlerFunc: e.deleteIconHandler(), Authenticated: true, RequiredRoles: []repository.Permission{repository.PermissionAdmin, repository.PermissionObjectiveDesigner}},
		{Method: "DELETE", Path: "/:team_id", HandlerFunc: e.deleteTeamHandler(), Authenticated: true, RequiredRoles: []repository.Permission{repository.PermissionAdmin}},
	}
	for i, route := range routes {
		routes[i].Path = basePath + route.Path
	}
	return routes
}

// @id GetTeams
// @Description Fetches all teams for an event
// @Tags team
// @Produce json
// @Param event_id path int true "Event Id"
// @Success 200 {array} Team
// @Router /events/{event_id}/teams [get]
func (e *TeamController) getTeamsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		event := getEvent(c)
		if event == nil {
			return
		}
		teams, err := e.teamService.GetTeamsForEvent(event.Id)
		if err != nil {
			respondError(c, err)
			return
		}

		c.JSON(200, utils.Map(teams, toTeamResponse))
	}
}

// @id CreateTeam
// @Description Creates a team for an event
// @Tags team
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param event_id path int true "Event Id"
// @Param teamCreate body TeamCreate true "Team to create"
// @Success 201 {object} Team
// @Router /events/{event_id}/teams [put]
func (e *TeamController) createTeamHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		event := getEvent(c)
		if event == nil {
			return
		}
		if event.Locked {
			c.JSON(400, gin.H{"error": "event is locked"})
			return
		}

		var team TeamCreate
		if err := c.BindJSON(&team); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		teamModel := team.toModel()
		teamModel.EventId = event.Id
		dbteam, err := e.teamService.SaveTeam(teamModel)
		if err != nil {
			respondError(c, err)
			return
		}
		c.JSON(201, toTeamResponse(dbteam))
	}
}

// @id GetTeam
// @Description Fetches a team by id
// @Tags team
// @Produce json
// @Param event_id path int true "Event Id"
// @Param team_id path int true "Team Id"
// @Success 200 {object} Team
// @Router /events/{event_id}/teams/{team_id} [get]
func (e *TeamController) getTeamHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		teamId, err := strconv.Atoi(c.Param("team_id"))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		team, err := e.teamService.GetTeamById(teamId)
		if err != nil {
			respondError(c, err)
			return
		}
		c.JSON(200, toTeamResponse(team))
	}
}

// @id UploadTeamIcon
// @Description Uploads an icon for a team
// @Security BearerAuth
// @Tags team
// @Accept multipart/form-data
// @Param event_id path int true "Event Id"
// @Param team_id path int true "Team Id"
// @Param icon formData file true "Icon file"
// @Success 204
// @Router /events/{event_id}/teams/{team_id}/icon [put]
func (e *TeamController) uploadIconHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		teamId, err := strconv.Atoi(c.Param("team_id"))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		file, header, err := c.Request.FormFile("icon")
		if err != nil {
			c.JSON(400, gin.H{"error": "icon file required"})
			return
		}
		defer file.Close()
		data, err := io.ReadAll(file)
		if err != nil {
			respondError(c, err)
			return
		}
		mimeType := header.Header.Get("Content-Type")
		if !strings.HasPrefix(mimeType, "image/") {
			c.JSON(400, gin.H{"error": "icon must be an image"})
			return
		}
		if err := e.teamService.UploadIcon(teamId, data, mimeType); err != nil {
			respondError(c, err)
			return
		}
		c.Status(204)
	}
}

// @id GetTeamIcon
// @Description Fetches the icon of a team
// @Tags team
// @Param event_id path int true "Event Id"
// @Param team_id path int true "Team Id"
// @Produce image/*
// @Success 200
// @Router /events/{event_id}/teams/{team_id}/icon [get]
func (e *TeamController) getIconHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		teamId, err := strconv.Atoi(c.Param("team_id"))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		icon, err := e.teamService.GetIcon(teamId)
		if err != nil {
			if errors.Is(err, apperr.ErrNotFound) {
				c.Header("Cache-Control", "no-store")
				c.Status(404)
			} else {
				respondError(c, err)
			}
			return
		}
		c.Header("Cache-Control", "public, max-age=86400")
		c.Data(200, icon.MimeType, icon.Icon)
	}
}

// @id DeleteTeamIcon
// @Description Removes the icon of a team
// @Security BearerAuth
// @Tags team
// @Param event_id path int true "Event Id"
// @Param team_id path int true "Team Id"
// @Success 204
// @Router /events/{event_id}/teams/{team_id}/icon [delete]
func (e *TeamController) deleteIconHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		teamId, err := strconv.Atoi(c.Param("team_id"))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		if err := e.teamService.DeleteIcon(teamId); err != nil {
			respondError(c, err)
			return
		}
		c.Status(204)
	}
}

// @id DeleteTeam
// @Description Deletes a team
// @Security BearerAuth
// @Tags team
// @Produce json
// @Param event_id path int true "Event Id"
// @Param team_id path int true "Team Id"
// @Success 204
// @Router /events/{event_id}/teams/{team_id} [delete]
func (e *TeamController) deleteTeamHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		teamId, err := strconv.Atoi(c.Param("team_id"))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		err = e.teamService.DeleteTeam(teamId)
		if err != nil {
			respondError(c, err)
			return
		}
		c.Status(204)
	}
}

// @id GetSortedUsers
// @Description Fetches all users of an event sorted by team and role
// @Tags team, user
// @Produce json
// @Param event_id path int true "Event Id"
// @Success 200 {array} service.SortedUser
// @Router /events/{event_id}/teams/users [get]
func (e *TeamController) getSortedUsersHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		event := getEvent(c)
		if event == nil {
			return
		}
		users, err := e.teamService.GetSortedUsersForEvent(event.Id)
		if err != nil {
			respondError(c, err)
			return
		}
		c.JSON(200, users)
	}
}

// @id AddUsersToTeams
// @Description Adds users to teams
// @Tags team, user
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param event_id path int true "Event Id"
// @Param teamUsers body []TeamUserCreate true "Users to add to teams"
// @Success 204
// @Router /events/{event_id}/teams/users [put]
func (e *TeamController) addUsersToTeamsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		event := getEvent(c)
		if event == nil {
			return
		}

		var teamUsers []TeamUserCreate
		if err := c.BindJSON(&teamUsers); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		teamUsersModel := utils.Map(teamUsers, teamUserCreateToModel)
		err := e.teamService.AddUsersToTeams(teamUsersModel, event)
		if err != nil {
			respondError(c, err)
			return
		}
		func() {
			_, err := client.NewLocalDiscordClient().AssignRoles()
			if err != nil {
				slog.Error("failed to assign roles in Discord", "error", err)
			}
		}()
		c.Status(204)
	}
}

type TeamUserCreate struct {
	TeamId     int  `json:"team_id"`
	UserId     int  `json:"user_id" binding:"required"`
	IsTeamLead bool `json:"is_team_lead"`
}

type TeamCreate struct {
	Id             *int     `json:"id"`
	Name           string   `json:"name" binding:"required"`
	AllowedClasses []string `json:"allowed_classes" binding:"required"`
	Color          string   `json:"color"`
	Abbreviation   string   `json:"abbreviation"`
	DiscordRoleId  *string  `json:"discord_role_id,omitempty"`
}

type Team struct {
	Id             int      `json:"id" binding:"required"`
	Name           string   `json:"name" binding:"required"`
	Abbreviation   string   `json:"abbreviation"`
	AllowedClasses []string `json:"allowed_classes" binding:"required"`
	EventId        int      `json:"event_id" binding:"required"`
	Color          string   `json:"color"`
	DiscordRoleId  *string  `json:"discord_role_id,omitempty"`
	IconUrl        string   `json:"icon_url,omitempty"`
}

func teamUserCreateToModel(teamUserCreate TeamUserCreate) *repository.TeamUser {
	return &repository.TeamUser{
		TeamId:     teamUserCreate.TeamId,
		UserId:     teamUserCreate.UserId,
		IsTeamLead: teamUserCreate.IsTeamLead,
	}
}

func (e *TeamCreate) toModel() *repository.Team {
	team := &repository.Team{
		Name:           e.Name,
		AllowedClasses: e.AllowedClasses,
		Color:          e.Color,
		Abbreviation:   e.Abbreviation,
		DiscordRoleId:  e.DiscordRoleId,
	}
	if e.Id != nil {
		team.Id = *e.Id
	}
	return team
}
