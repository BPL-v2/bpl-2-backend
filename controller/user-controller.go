package controller

import (
	"bpl/auth"
	"bpl/repository"
	"bpl/service"
	"bpl/utils"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type UserController struct {
	userService  *service.UserService
	eventService *service.EventService
}

func NewUserController() *UserController {
	return &UserController{
		userService:  service.NewUserService(),
		eventService: service.NewEventService(),
	}
}

func setupUserController() []RouteInfo {
	e := NewUserController()
	basePath := ""
	routes := []RouteInfo{
		{Method: "GET", Path: "/events/:event_id/users", HandlerFunc: e.getUsersForEventHandler()},
		{Method: "GET", Path: "/users", HandlerFunc: e.getAllUsersHandler(), Authenticated: true, RequiredRoles: []repository.Permission{repository.PermissionAdmin}},
		{Method: "GET", Path: "/users/self", HandlerFunc: e.getUserHandler(), Authenticated: true},
		{Method: "PATCH", Path: "/users/self", HandlerFunc: e.updateUserHandler(), Authenticated: true},
		{Method: "PATCH", Path: "/users/:userId", HandlerFunc: e.changePermissionsHandler(), Authenticated: true, RequiredRoles: []repository.Permission{repository.PermissionAdmin}},
		{Method: "POST", Path: "/users/logout", HandlerFunc: e.logoutHandler(), Authenticated: true},
		{Method: "POST", Path: "/users/remove-auth", HandlerFunc: e.removeAuthHandler(), Authenticated: true},
	}
	for i, route := range routes {
		routes[i].Path = basePath + route.Path
	}
	return routes
}

// @id GetAllUsers
// @Description Fetches all users
// @Tags user
// @Produce json
// @Success 200 {array} User
// @Security ApiKeyAuth
// @Router /users [get]
func (e *UserController) getAllUsersHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		users, err := e.userService.GetAllUsers("OauthAccounts")
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, utils.Map(users, toUserResponse))
	}
}

// @id ChangePermissions
// @Description Changes the permissions of a user
// @Tags user
// @Accept json
// @Produce json
// @Param userId path int true "User Id"
// @Param permissions body repository.Permissions true "Permissions"
// @Success 200 {object} User
// @Security ApiKeyAuth
// @Router /users/{userId} [patch]
func (e *UserController) changePermissionsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		userId, err := strconv.Atoi(c.Param("userId"))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		var permissions repository.Permissions
		if err := c.BindJSON(&permissions); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		user, err := e.userService.ChangePermissions(userId, permissions)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, toUserResponse(user))
	}
}

// @id GetUser
// @Description Fetches the authenticated user
// @Tags user
// @Produce json
// @Success 200 {object} User
// @Security ApiKeyAuth
// @Router /users/self [get]
func (e *UserController) getUserHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := e.userService.GetUserFromAuthCookie(c)
		if err != nil {
			c.JSON(401, gin.H{"error": "Not authenticated"})
			return
		}
		authToken, _ := auth.CreateToken(user)
		c.SetSameSite(http.SameSiteStrictMode)
		c.SetCookie("auth", authToken, 60*60*24*7, "/", os.Getenv("PUBLIC_DOMAIN"), false, true)
		c.JSON(200, toUserResponse(user))
	}
}

// @id Logout
// @Description Logs out the authenticated user
// @Tags user
// @Produce json
// @Success 200
// @Security ApiKeyAuth
// @Router /users/logout [post]
func (e *UserController) logoutHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.SetCookie("auth", "", -1, "/", "", false, true)
		c.JSON(200, gin.H{"message": "Logged out"})
	}
}

// @id RemoveAuth
// @Description Removes an authentication provider from the authenticated user
// @Tags user
// @Produce json
// @Param provider query string true "Provider"
// @Success 200 {object} User
// @Security ApiKeyAuth
// @Router /users/remove-auth [post]
func (e *UserController) removeAuthHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		provider := repository.Provider(c.Request.URL.Query().Get("provider"))
		if provider == "" {
			c.JSON(400, gin.H{"error": "No provider specified"})
			return
		}
		user, err := e.userService.GetUserFromAuthCookie(c)
		if err != nil {
			c.JSON(401, gin.H{"error": "Not authenticated"})
			return
		}
		user, err = e.userService.RemoveProvider(user, provider)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		authToken, err := auth.CreateToken(user)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.SetSameSite(http.SameSiteStrictMode)
		c.SetCookie("auth", authToken, 60*60*24*7, "/", os.Getenv("PUBLIC_DOMAIN"), false, true)
		c.JSON(200, toUserResponse(user))
	}
}

// @id GetUsersForEvent
// @Description Fetches all users for an event
// @Tags user
// @Produce json
// @Param event_id path int true "Event Id"
// @Success 200 {object} map[int][]MinimalUser
// @Router /events/{event_id}/users [get]
func (e *UserController) getUsersForEventHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		eventId, err := strconv.Atoi(c.Param("event_id"))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		event, err := e.eventService.GetEventById(eventId, "Teams", "Teams.Users")
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(404, gin.H{"error": "Event not found"})
			} else {
				c.JSON(500, gin.H{"error": err.Error()})
			}
			return
		}
		teamUsers := make(map[int][]*MinimalUser)
		for _, team := range event.Teams {
			teamUsers[team.Id] = make([]*MinimalUser, 0)
			for _, user := range team.Users {
				teamUsers[team.Id] = append(teamUsers[team.Id], toMinimalUserResponse(user))
			}
		}
		c.JSON(200, teamUsers)
	}
}

// @id UpdateUser
// @Description Updates the authenticated users display name
// @Tags user
// @Accept json
// @Produce json
// @Param user body UserUpdate true "User"
// @Success 200 {object} User
// @Security ApiKeyAuth
// @Router /users/self [patch]
func (e *UserController) updateUserHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := e.userService.GetUserFromAuthCookie(c)
		if err != nil {
			c.JSON(401, gin.H{"error": "Not authenticated"})
			return
		}
		var userUpdate UserUpdate
		if err := c.BindJSON(&userUpdate); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		user.DisplayName = userUpdate.DisplayName
		user, err = e.userService.SaveUser(user)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, toUserResponse(user))
	}
}

type UserUpdate struct {
	DisplayName string `json:"display_name" binding:"required"`
}

type User struct {
	Id                   int        `json:"id" binding:"required"`
	DisplayName          string     `json:"display_name" binding:"required"`
	AcountName           *string    `json:"account_name"`
	DiscordId            *string    `json:"discord_id"`
	DiscordName          *string    `json:"discord_name"`
	TwitchId             *string    `json:"twitch_id"`
	TwitchName           *string    `json:"twitch_name"`
	TokenExpiryTimestamp *time.Time `json:"token_expiry_timestamp"`

	Permissions []repository.Permission `json:"permissions" binding:"required"`
}

type NonSensitiveUser struct {
	Id          int     `json:"id" binding:"required"`
	DisplayName string  `json:"display_name" binding:"required"`
	AcountName  *string `json:"account_name"`
	DiscordId   *string `json:"discord_id"`
	DiscordName *string `json:"discord_name"`
	TwitchId    *string `json:"twitch_id"`
	TwitchName  *string `json:"twitch_name"`
}

type MinimalUser struct {
	Id          int    `json:"id" binding:"required"`
	DisplayName string `json:"display_name" binding:"required"`
}

func toUserResponse(user *repository.User) *User {
	response := &User{
		Id:          user.Id,
		DisplayName: user.DisplayName,
		Permissions: user.Permissions,
	}
	for _, oauth := range user.OauthAccounts {
		switch oauth.Provider {
		case repository.ProviderDiscord:
			response.DiscordId = &oauth.AccountId
			response.DiscordName = &oauth.Name
		case repository.ProviderTwitch:
			response.TwitchId = &oauth.AccountId
			response.TwitchName = &oauth.Name
		case repository.ProviderPoE:
			response.AcountName = &oauth.AccountId
			response.TokenExpiryTimestamp = &oauth.Expiry

		}
	}

	return response
}

func toNonSensitiveUserResponse(user *repository.User) *NonSensitiveUser {
	if user == nil {
		return nil
	}
	response := &NonSensitiveUser{
		Id:          user.Id,
		DisplayName: user.DisplayName,
	}
	for _, oauth := range user.OauthAccounts {
		switch oauth.Provider {
		case repository.ProviderDiscord:
			response.DiscordId = &oauth.AccountId
			response.DiscordName = &oauth.Name
		case repository.ProviderTwitch:
			response.TwitchId = &oauth.AccountId
			response.TwitchName = &oauth.Name
		case repository.ProviderPoE:
			response.AcountName = &oauth.AccountId
		}
	}
	return response
}

func toMinimalUserResponse(user *repository.User) *MinimalUser {
	return &MinimalUser{
		Id:          user.Id,
		DisplayName: user.DisplayName,
	}
}
