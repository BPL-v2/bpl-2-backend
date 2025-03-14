package controller

import (
	"bpl/client"
	"bpl/repository"
	"bpl/service"
	"bpl/utils"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type TeamController struct {
	teamService  *service.TeamService
	eventService *service.EventService
}

func NewTeamController() *TeamController {
	return &TeamController{
		teamService:  service.NewTeamService(),
		eventService: service.NewEventService(),
	}
}

func toTeamResponse(team *repository.Team) *Team {
	return &Team{
		Id:             team.Id,
		Name:           team.Name,
		AllowedClasses: team.AllowedClasses,
		EventId:        team.EventId,
	}
}

func setupTeamController() []RouteInfo {
	e := NewTeamController()
	basePath := "events/:event_id/teams"
	routes := []RouteInfo{
		{Method: "GET", Path: "", HandlerFunc: e.getTeamsHandler()},
		{Method: "PUT", Path: "", HandlerFunc: e.createTeamHandler(), Authenticated: true, RequiredRoles: []repository.Permission{repository.PermissionAdmin}},
		{Method: "PUT", Path: "/users", HandlerFunc: e.addUsersToTeamsHandler(), Authenticated: true, RequiredRoles: []repository.Permission{repository.PermissionAdmin}},
		{Method: "GET", Path: "/:team_id", HandlerFunc: e.getTeamHandler()},
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
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, utils.Map(teams, toTeamResponse))
	}
}

// @id CreateTeam
// @Description Creates a team for an event
// @Tags team
// @Accept json
// @Produce json
// @Param event_id path int true "Event Id"
// @Param body body TeamCreate true "Team to create"
// @Success 201 {object} Team
// @Router /events/{event_id}/teams [put]
func (e *TeamController) createTeamHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		event := getEvent(c)
		if event == nil {
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
			c.JSON(500, gin.H{"error": err.Error()})
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
			if err == gorm.ErrRecordNotFound {
				c.JSON(404, gin.H{"error": "Team not found"})
			} else {
				c.JSON(500, gin.H{"error": err.Error()})
			}
			return
		}
		c.JSON(200, toTeamResponse(team))
	}
}

// @id DeleteTeam
// @Description Deletes a team
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
			if err == gorm.ErrRecordNotFound {
				c.JSON(404, gin.H{"error": "Team not found"})
			} else {
				c.JSON(500, gin.H{"error": err.Error()})
			}
			return
		}
		c.Status(204)
	}
}

// @id AddUsersToTeams
// @Description Adds users to teams
// @Tags team, user
// @Accept json
// @Produce json
// @Param event_id path int true "Event Id"
// @Param body body []TeamUserCreate true "Users to add to teams"
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
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		go client.NewLocalDiscordClient().AssignRoles()
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
}

type Team struct {
	Id             int      `json:"id" binding:"required"`
	Name           string   `json:"name" binding:"required"`
	AllowedClasses []string `json:"allowed_classes" binding:"required"`
	EventId        int      `json:"event_id" binding:"required"`
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
	}
	if e.Id != nil {
		team.Id = *e.Id
	}
	return team
}
