package controller

import (
	"bpl/repository"
	"bpl/service"
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
		{Method: "POST", Path: "/objectives", HandlerFunc: e.createObjectiveTeamSuggestionHandler(), Authenticated: true},
		{Method: "POST", Path: "/categories", HandlerFunc: e.createCategoryTeamSuggestionHandler(), Authenticated: true},
		{Method: "DELETE", Path: "/objectives/:objective_id", HandlerFunc: e.deleteObjectiveTeamSuggestionHandler(), Authenticated: true},
		{Method: "DELETE", Path: "/categories/:category_id", HandlerFunc: e.deleteCategoryTeamSuggestionHandler(), Authenticated: true},
	}
	for i, route := range routes {
		routes[i].Path = basePath + route.Path
	}
	return routes
}

func (e *TeamSuggestionController) getTeamForUser(c *gin.Context) *repository.TeamUser {
	event := getEvent(c)
	if event == nil {
		return nil
	}
	user, err := e.userService.GetUserFromAuthHeader(c)
	if err != nil {
		c.JSON(401, gin.H{"error": err.Error()})
		return nil
	}

	team, err := e.teamService.GetTeamForUser(event.Id, user.Id)
	if err != nil {
		c.JSON(404, gin.H{"error": err.Error()})
		return nil
	}
	return team
}

// @id GetTeamSuggestions
// @Description Fetches all suggestions for your team for an event
// @Tags team
// @Security BearerAuth
// @Produce json
// @Param event_id path int true "Event Id"
// @Success 200 {object} Suggestions
// @Router /events/{event_id}/suggestions [get]
func (e *TeamSuggestionController) getTeamSuggestionsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		team := e.getTeamForUser(c)
		if team == nil {
			return
		}
		suggestions, err := e.teamSuggestionService.GetSuggestionsForTeam(team.TeamId)
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
// @Param body body SuggestionCreate true "Suggestion to create"
// @Success 201
// @Router /events/{event_id}/suggestions/objectives [POST]
func (e *TeamSuggestionController) createObjectiveTeamSuggestionHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		team := e.getTeamForUser(c)
		if team == nil {
			return
		}
		if !team.IsTeamLead {
			c.JSON(403, gin.H{"error": "You are not a team lead"})
			return
		}
		var suggestionCreate SuggestionCreate
		if err := c.BindJSON(&suggestionCreate); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		err := e.teamSuggestionService.SaveSuggestion(suggestionCreate.Id, team.TeamId, true)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(201, suggestionCreate)
	}
}

// @id CreateCategoryTeamSuggestion
// @Description Creates a suggestion for a category for your team for an event
// @Tags team
// @Accept json
// @Security BearerAuth
// @Produce json
// @Param event_id path int true "Event Id"
// @Param body body SuggestionCreate true "Suggestion to create"
// @Success 201
// @Router /events/{event_id}/suggestions/categories [POST]
func (e *TeamSuggestionController) createCategoryTeamSuggestionHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		team := e.getTeamForUser(c)
		if team == nil {
			return
		}
		if !team.IsTeamLead {
			c.JSON(403, gin.H{"error": "You are not a team lead"})
			return
		}
		var suggestionCreate SuggestionCreate
		if err := c.BindJSON(&suggestionCreate); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		err := e.teamSuggestionService.SaveSuggestion(suggestionCreate.Id, team.TeamId, false)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(201, suggestionCreate)
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
// @Router /events/{event_id}/suggestions/objectives/{objective_id} [delete]
func (e *TeamSuggestionController) deleteObjectiveTeamSuggestionHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		team := e.getTeamForUser(c)
		if team == nil {
			return
		}
		if !team.IsTeamLead {
			c.JSON(403, gin.H{"error": "You are not a team lead"})
			return
		}
		objectiveId, err := strconv.Atoi(c.Param("objective_id"))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		err = e.teamSuggestionService.DeleteSuggestion(objectiveId, team.TeamId, true)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(204, nil)
	}
}

// @id DeleteCategoryTeamSuggestion
// @Description Deletes a suggestion for a category for your team for an event
// @Tags team
// @Security BearerAuth
// @Produce json
// @Param event_id path int true "Event Id"
// @Param category_id path int true "Category Id"
// @Success 204
// @Router /events/{event_id}/suggestions/categories/{category_id} [delete]
func (e *TeamSuggestionController) deleteCategoryTeamSuggestionHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		team := e.getTeamForUser(c)
		if team == nil {
			return
		}
		if !team.IsTeamLead {
			c.JSON(403, gin.H{"error": "You are not a team lead"})
			return
		}
		categoryId, err := strconv.Atoi(c.Param("category_id"))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		err = e.teamSuggestionService.DeleteSuggestion(categoryId, team.TeamId, false)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(204, nil)
	}
}

type SuggestionCreate struct {
	Id int `json:"id" binding:"required"`
}

type Suggestions struct {
	CategoryIds  []int `json:"category_ids" binding:"required"`
	ObjectiveIds []int `json:"objective_ids"  binding:"required"`
}

func toSuggestionResponse(suggestions []*repository.TeamSuggestion) *Suggestions {
	category_ids := make([]int, 0)
	objective_ids := make([]int, 0)
	for _, suggestion := range suggestions {
		if suggestion.IsObjective {
			objective_ids = append(objective_ids, suggestion.Id)
		} else {
			category_ids = append(category_ids, suggestion.Id)
		}
	}
	return &Suggestions{
		CategoryIds:  category_ids,
		ObjectiveIds: objective_ids,
	}
}
