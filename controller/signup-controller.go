package controller

import (
	"bpl/client"
	"bpl/repository"
	"bpl/service"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SignupController struct {
	signupService *service.SignupService
	userService   *service.UserService
	teamService   *service.TeamService
}

func NewSignupController() *SignupController {
	return &SignupController{
		signupService: service.NewSignupService(),
		userService:   service.NewUserService(),
		teamService:   service.NewTeamService(),
	}
}

func setupSignupController() []RouteInfo {
	e := NewSignupController()
	basePath := "/events/:event_id/signups"
	routes := []RouteInfo{
		{Method: "GET", Path: "", HandlerFunc: e.getEventSignupsHandler(), Authenticated: true, RequiredRoles: []repository.Permission{repository.PermissionAdmin, repository.PermissionManager}},
		{Method: "GET", Path: "/self", HandlerFunc: e.getPersonalSignupHandler(), Authenticated: true},
		{Method: "PUT", Path: "/self", HandlerFunc: e.createSignupHandler(), Authenticated: true},
		{Method: "DELETE", Path: "/self", HandlerFunc: e.deleteSignupHandler(), Authenticated: true},
		{Method: "PUT", Path: "/self/actual-playtime", HandlerFunc: e.reportPlaytime(), Authenticated: true},
		{Method: "GET", Path: "/discord", HandlerFunc: getDiscordMembersHandler(), Authenticated: true, RequiredRoles: []repository.Permission{repository.PermissionAdmin, repository.PermissionManager}},
	}
	for i, route := range routes {
		routes[i].Path = basePath + route.Path
	}
	return routes
}

func getDiscordMembersHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		event := getEvent(c)
		if event == nil {
			return
		}
		discordClient := client.NewLocalDiscordClient()
		members, err := discordClient.GetServerMembers()
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to fetch discord members"})
			return
		}
		c.JSON(200, members)
	}
}

type ReportPlaytimeRequest struct {
	ActualPlaytime int `json:"actual_playtime" binding:"required"`
}

// @id ReportPlaytime
// @Description Reports the actual playtime for the authenticated user
// @Tags signup
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} Signup
// @Param event_id path int true "Event Id"
// @Param body body ReportPlaytimeRequest true "Actual Playtime"
// @Router /events/{event_id}/signups/self/actual-playtime [put]
func (e *SignupController) reportPlaytime() gin.HandlerFunc {
	return func(c *gin.Context) {
		event := getEvent(c)
		if event == nil {
			return
		}
		user, err := e.userService.GetUserFromAuthHeader(c)
		if err != nil {
			c.JSON(401, gin.H{"error": "Not authenticated"})
			return
		}
		actualPlaytimeRequest := ReportPlaytimeRequest{}
		if err := c.BindJSON(&actualPlaytimeRequest); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		signup, err := e.signupService.ReportPlaytime(user.Id, event.Id, actualPlaytimeRequest.ActualPlaytime)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, toSignupResponse(signup))
	}
}

// @id GetPersonalSignup
// @Description Fetches an authenticated user's signup for the event
// @Tags signup
// @Produce json
// @Security BearerAuth
// @Success 200 {object} Signup
// @Param event_id path int true "Event Id"
// @Router /events/{event_id}/signups/self [get]
func (e *SignupController) getPersonalSignupHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		event := getEvent(c)
		if event == nil {
			return
		}
		user, err := e.userService.GetUserFromAuthHeader(c)
		if err != nil {
			c.JSON(401, gin.H{"error": "Not authenticated"})
			return
		}
		signup, err := e.signupService.GetSignupForUser(user.Id, event.Id)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(404, gin.H{"error": "Not signed up"})
			} else {
				c.JSON(400, gin.H{"error": err.Error()})
			}
			return
		}
		c.JSON(200, toSignupResponse(signup))
	}
}

// @id CreateSignup
// @Description Creates a signup for the authenticated user
// @Tags signup
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 201 {object} Signup
// @Param event_id path int true "Event Id"
// @Param body body SignupCreate true "Signup"
// @Router /events/{event_id}/signups/self [put]
func (e *SignupController) createSignupHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		event := getEvent(c)
		if event == nil {
			return
		}
		user, err := e.userService.GetUserFromAuthHeader(c)
		if err != nil {
			c.JSON(401, gin.H{"error": "Not authenticated"})
			return
		}
		if event.ApplicationStartTime.After(time.Now()) || event.ApplicationEndTime.Before(time.Now()) {
			c.JSON(400, gin.H{"error": "Applications are not open"})
			return
		}
		_, err = e.teamService.GetTeamForUser(event.Id, user.Id)
		if err == nil {
			c.JSON(400, gin.H{"error": "Cannot change signup after being added to a team"})
			return
		}
		//  TODO: Uncomment this when discord server check is implemented
		// err = e.userService.DiscordServerCheck(user)
		// if err != nil {
		// 	c.JSON(403, gin.H{"error": err.Error()})
		// 	return
		// }
		var signupCreate SignupCreate
		if err := c.BindJSON(&signupCreate); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		signup, err := e.signupService.GetSignupForUser(user.Id, event.Id)
		if err != nil {
			signup = &repository.Signup{
				UserId:    user.Id,
				User:      user,
				EventId:   event.Id,
				Timestamp: time.Now(),
			}
		}
		signup.ExpectedPlayTime = signupCreate.ExpectedPlaytime
		signup.NeedsHelp = signupCreate.NeedsHelp
		signup.WantsToHelp = signupCreate.WantsToHelp
		if signupCreate.PartnerAccountName != "" {
			partner, err := e.userService.GetUserByOauthProviderAndAccountName(repository.ProviderPoE, signupCreate.PartnerAccountName)
			if err != nil {
				c.JSON(404, gin.H{"error": "Could not find partner account"})
				return
			}
			signup.PartnerId = &partner.Id
			signup.Partner = partner
		}
		signup, err = e.signupService.SaveSignup(signup)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		c.JSON(201, toSignupResponse(signup))
	}
}

// @id DeleteSignup
// @Description Deletes the authenticated user's signup for the event
// @Tags signup
// @Produce json
// @Security BearerAuth
// @Success 204
// @Param event_id path int true "Event Id"
// @Router /events/{event_id}/signups/self [delete]
func (e *SignupController) deleteSignupHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		event := getEvent(c)
		if event == nil {
			return
		}
		user, err := e.userService.GetUserFromAuthHeader(c)
		if err != nil {
			c.JSON(401, gin.H{"error": "Not authenticated"})
			return
		}
		err = e.signupService.RemoveSignup(user.Id, event.Id)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{})
	}
}

// @id GetEventSignups
// @Description Fetches all signups for the event
// @Tags signup
// @Security BearerAuth
// @Produce json
// @Success 200 {object} []Signup
// @Param event_id path int true "Event Id"
// @Router /events/{event_id}/signups [get]
func (e *SignupController) getEventSignupsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		event := getEvent(c)
		if event == nil {
			return
		}
		signups, err := e.signupService.GetSignupsForEvent(event)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		signups = signups[:min(event.MaxSize, len(signups))]
		teamUsers, err := e.teamService.GetTeamUsersForEvent(event)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		teamUsersMap := make(map[int]*repository.TeamUser, 0)
		for _, teamUser := range teamUsers {
			teamUsersMap[teamUser.UserId] = teamUser
		}
		signupsWithUsers := make([]*Signup, 0)
		for _, signup := range signups {
			resp := &Signup{
				User:             toNonSensitiveUserResponse(signup.User),
				PartnerId:        signup.PartnerId,
				Timestamp:        signup.Timestamp,
				ExpectedPlaytime: signup.ExpectedPlayTime,
				NeedsHelp:        signup.NeedsHelp,
				WantsToHelp:      signup.WantsToHelp,
				ActualPlaytime:   signup.ActualPlayTime,
			}
			if teamUser, ok := teamUsersMap[signup.UserId]; ok {
				resp.TeamId = &teamUser.TeamId
				resp.IsTeamLead = teamUser.IsTeamLead
			}
			signupsWithUsers = append(signupsWithUsers, resp)
		}
		c.JSON(200, signupsWithUsers)

	}
}

type Signup struct {
	User             *NonSensitiveUser `json:"user" binding:"required"`
	Partner          *NonSensitiveUser `json:"partner"`
	PartnerId        *int              `json:"partner_id"`
	Timestamp        time.Time         `json:"timestamp" binding:"required"`
	ExpectedPlaytime int               `json:"expected_playtime" binding:"required"`
	ActualPlaytime   int               `json:"actual_playtime" binding:"required"`
	TeamId           *int              `json:"team_id"`
	IsTeamLead       bool              `json:"team_lead" binding:"required"`
	NeedsHelp        bool              `json:"needs_help"`
	WantsToHelp      bool              `json:"wants_to_help"`
}

type SignupCreate struct {
	ExpectedPlaytime   int    `json:"expected_playtime" binding:"required"`
	NeedsHelp          bool   `json:"needs_help"`
	WantsToHelp        bool   `json:"wants_to_help"`
	PartnerAccountName string `json:"partner_account_name"`
}

func toSignupResponse(signup *repository.Signup) *Signup {
	if signup == nil {
		return nil
	}

	return &Signup{
		User:             toNonSensitiveUserResponse(signup.User),
		Partner:          toNonSensitiveUserResponse(signup.Partner),
		PartnerId:        signup.PartnerId,
		Timestamp:        signup.Timestamp,
		ExpectedPlaytime: signup.ExpectedPlayTime,
		ActualPlaytime:   signup.ActualPlayTime,
		NeedsHelp:        signup.NeedsHelp,
		WantsToHelp:      signup.WantsToHelp,
	}
}
