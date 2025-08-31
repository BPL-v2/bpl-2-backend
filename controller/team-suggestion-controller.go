package controller

import (
	"bpl/repository"
	"bpl/service"
	"bpl/utils"
	"strconv"

	"github.com/gin-gonic/gin"
)

type TeamSuggestionController struct {
	teamSuggestionService *service.TeamSuggestionService
	teamService           *service.TeamService
	userService           *service.UserService
}

func NewTeamSuggestionController() *TeamSuggestionController {
	return &TeamSuggestionController{
		teamSuggestionService: service.NewTeamSuggestionService(),
		teamService:           service.NewTeamService(),
		userService:           service.NewUserService(),
	}
}

func setupTeamSuggestionController() []RouteInfo {
	e := NewTeamSuggestionController()
	basePath := "events/:event_id/suggestions"
	routes := []RouteInfo{
		{Method: "GET", Path: "", HandlerFunc: e.getTeamSuggestionsHandler(), Authenticated: true},
		{Method: "PUT", Path: "/:objective_id", HandlerFunc: e.createTeamSuggestionHandler(), Authenticated: true},
		{Method: "DELETE", Path: "/:objective_id", HandlerFunc: e.deleteTeamSuggestionHandler(), Authenticated: true},
	}
	for i, route := range routes {
		routes[i].Path = basePath + route.Path
	}
	return routes
}

func (e *TeamSuggestionController) GetTeamUser(c *gin.Context, requiresTeamLead bool) *repository.TeamUser {
	event := getEvent(c)
	if event == nil {
		return nil
	}
	teamUser, _, err := e.userService.GetTeamForUser(c, event)
	if err != nil {
		c.JSON(403, gin.H{"error": err.Error()})
		return nil
	}
	if requiresTeamLead && !teamUser.IsTeamLead {
		c.JSON(403, gin.H{"error": "You are not a team lead"})
		return nil
	}
	return teamUser
}

// @id GetTeamSuggestions
// @Description Fetches all suggestions for your team for an event
// @Tags team
// @Security BearerAuth
// @Produce json
// @Param event_id path int true "Event Id"
// @Success 200 {array} int
// @Router /events/{event_id}/suggestions [get]
func (e *TeamSuggestionController) getTeamSuggestionsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		teamUser := e.GetTeamUser(c, false)
		if teamUser == nil {
			return
		}
		suggestions, err := e.teamSuggestionService.GetSuggestionsForTeam(teamUser.TeamId)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, toSuggestionResponse(suggestions))
	}
}

// @id CreateObjectiveTeamSuggestion
// @Description Creates a suggestion for an objective for your team for an event
// @Tags team
// @Accept json
// @Security BearerAuth
// @Produce json
// @Param event_id path int true "Event Id"
// @Param objective_id path int true "Objective Id"
// @Param body body TeamSuggestion true "Suggestion data"
// @Success 201
// @Router /events/{event_id}/suggestions/{objective_id} [PUT]
func (e *TeamSuggestionController) createTeamSuggestionHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		teamUser := e.GetTeamUser(c, true)
		if teamUser == nil {
			return
		}
		objectiveId, err := strconv.Atoi(c.Param("objective_id"))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		var suggestion TeamSuggestion
		if err := c.ShouldBindJSON(&suggestion); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		err = e.teamSuggestionService.SaveSuggestion(objectiveId, teamUser.TeamId, suggestion.Extra)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(201, gin.H{"message": "Suggestion created successfully"})
	}
}

// @id DeleteObjectiveTeamSuggestion
// @Description Deletes a suggestion for an objective for your team for an event
// @Tags team
// @Security BearerAuth
// @Produce json
// @Param event_id path int true "Event Id"
// @Param objective_id path int true "Objective Id"
// @Success 204
// @Router /events/{event_id}/suggestions/{objective_id} [delete]
func (e *TeamSuggestionController) deleteTeamSuggestionHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		teamUser := e.GetTeamUser(c, true)
		if teamUser == nil {
			return
		}
		objectiveId, err := strconv.Atoi(c.Param("objective_id"))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		err = e.teamSuggestionService.DeleteSuggestion(objectiveId, teamUser.TeamId)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(204, nil)
	}
}

func toSuggestionResponse(suggestions []*repository.TeamSuggestion) []*TeamSuggestion {
	return utils.Map(suggestions, func(s *repository.TeamSuggestion) *TeamSuggestion {
		return &TeamSuggestion{
			ObjectiveId: s.Id,
			Extra:       s.Extra,
		}
	})
}

type TeamSuggestion struct {
	ObjectiveId int    `json:"objective_id"`
	Extra       string `json:"extra"`
}
