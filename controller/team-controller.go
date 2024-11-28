package controller

import (
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

func NewTeamController(db *gorm.DB) *TeamController {
	return &TeamController{
		teamService:  service.NewTeamService(db),
		eventService: service.NewEventService(db),
	}
}

func toTeamResponse(team *repository.Team) TeamResponse {
	return TeamResponse{
		ID:             team.ID,
		Name:           team.Name,
		AllowedClasses: team.AllowedClasses,
		EventID:        team.EventID,
	}
}

func setupTeamController(db *gorm.DB) []RouteInfo {
	e := NewTeamController(db)
	basePath := "events/:event_id/teams"
	routes := []RouteInfo{
		{Method: "GET", Path: "", HandlerFunc: e.getTeamsHandler()},
		{Method: "POST", Path: "", HandlerFunc: e.createTeamHandler()},
		{Method: "GET", Path: "/:team_id", HandlerFunc: e.getTeamHandler()},
		{Method: "PATCH", Path: "/:team_id", HandlerFunc: e.updateTeamHandler()},
		{Method: "DELETE", Path: "/:team_id", HandlerFunc: e.deleteTeamHandler()},
	}
	for i, route := range routes {
		routes[i].Path = basePath + route.Path
	}
	return routes
}

func (e *TeamController) getTeamsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		event_id, err := strconv.Atoi(c.Param("event_id"))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		event, err := e.eventService.GetEventById(event_id, "Teams")
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(404, gin.H{"error": "Event not found"})
			} else {
				c.JSON(500, gin.H{"error": err.Error()})
			}
			return
		}

		c.JSON(200, utils.Map(event.Teams, toTeamResponse))
	}
}

func (e *TeamController) createTeamHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		event_id, err := strconv.Atoi(c.Param("event_id"))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		var team TeamCreate
		if err := c.BindJSON(&team); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		teamModel := team.toModel()
		teamModel.EventID = event_id
		dbteam, err := e.teamService.CreateTeam(teamModel)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(201, toTeamResponse(dbteam))
	}
}

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

func (e *TeamController) updateTeamHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		teamId, err := strconv.Atoi(c.Param("team_id"))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		var team TeamUpdate
		if err := c.BindJSON(&team); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		dbteam, err := e.teamService.UpdateTeam(teamId, team.toModel())
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(404, gin.H{"error": "Team not found"})
			} else {
				c.JSON(500, gin.H{"error": err.Error()})
			}
			return
		}
		c.JSON(200, toTeamResponse(dbteam))
	}
}

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

type TeamCreate struct {
	Name           string   `json:"name" binding:"required"`
	AllowedClasses []string `json:"allowed_classes"`
}

type TeamUpdate struct {
	Name           string   `json:"name"`
	AllowedClasses []string `json:"allowed_classes"`
}

type TeamResponse struct {
	ID             int      `json:"id"`
	Name           string   `json:"name"`
	AllowedClasses []string `json:"allowed_classes"`
	EventID        int      `json:"event_id"`
}

func (e *TeamCreate) toModel() *repository.Team {
	return &repository.Team{
		Name:           e.Name,
		AllowedClasses: e.AllowedClasses,
	}
}

func (e *TeamUpdate) toModel() *repository.Team {
	return &repository.Team{
		Name:           e.Name,
		AllowedClasses: e.AllowedClasses,
	}
}
