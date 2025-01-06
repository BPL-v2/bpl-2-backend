package controller

import (
	"bpl/auth"
	"bpl/repository"
	"bpl/service"
	"bpl/utils"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type UserController struct {
	userService  *service.UserService
	eventService *service.EventService
}

func NewUserController(db *gorm.DB) *UserController {
	return &UserController{
		userService:  service.NewUserService(db),
		eventService: service.NewEventService(db),
	}
}

func setupUserController(db *gorm.DB) []RouteInfo {
	e := NewUserController(db)
	basePath := ""
	routes := []RouteInfo{
		{Method: "GET", Path: "/events/:event_id/users", HandlerFunc: e.getUsersForEventHandler()},
		{Method: "GET", Path: "/users", HandlerFunc: e.getUsersHandler(), Authenticated: true, RequiredRoles: []repository.Permission{repository.PermissionAdmin}},
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

func (e *UserController) getUsersHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		users, err := e.userService.GetUsers()
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, utils.Map(users, toUserAdminResponse))
	}
}

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
		err = e.userService.ChangePermissions(userId, permissions)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, nil)
	}
}

func (e *UserController) getUserHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := e.userService.GetUserFromAuthCookie(c)
		if err != nil {
			c.JSON(401, gin.H{"error": "Not authenticated"})
			return
		}
		c.JSON(200, toUserResponse(user))
	}
}

func (e *UserController) logoutHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.SetCookie("auth", "", -1, "/", "", false, true)
		c.JSON(200, gin.H{"message": "Logged out"})
	}
}

func (e *UserController) removeAuthHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		provider := c.Request.URL.Query().Get("provider")
		if provider == "" {
			c.JSON(400, gin.H{"error": "No provider specified"})
			return
		}
		user, err := e.userService.GetUserFromAuthCookie(c)
		if err != nil {
			c.JSON(401, gin.H{"error": "Not authenticated"})
			return
		}
		user, err = e.userService.RemoveProvider(user, repository.OauthProvider(provider))
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
		teamUsers := make(map[int][]*MinimalUserResponse)
		for _, team := range event.Teams {
			teamUsers[team.ID] = make([]*MinimalUserResponse, 0)
			for _, user := range team.Users {
				teamUsers[team.ID] = append(teamUsers[team.ID], toMinimalUserResponse(user))
			}
		}
		c.JSON(200, teamUsers)
	}
}

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

type UserResponse struct {
	ID                   int     `json:"id"`
	DisplayName          string  `json:"display_name"`
	AcountName           *string `json:"account_name"`
	DiscordID            *string `json:"discord_id"`
	DiscordName          *string `json:"discord_name"`
	TwitchID             *string `json:"twitch_id"`
	TwitchName           *string `json:"twitch_name"`
	TokenExpiryTimestamp *int64  `json:"token_expiry_timestamp"`

	Permissions []repository.Permission `json:"permissions"`
}

type NonSensitiveUserResponse struct {
	ID          int     `json:"id"`
	DisplayName string  `json:"display_name"`
	AcountName  *string `json:"account_name"`
	DiscordID   *string `json:"discord_id"`
	DiscordName *string `json:"discord_name"`
	TwitchID    *string `json:"twitch_id"`
	TwitchName  *string `json:"twitch_name"`
}

type UserAdminResponse struct {
	ID          int                     `json:"id"`
	DisplayName string                  `json:"display_name"`
	AcountName  *string                 `json:"account_name"`
	DiscordID   *string                 `json:"discord_id"`
	DiscordName *string                 `json:"discord_name"`
	TwitchName  *string                 `json:"twitch_name"`
	TwitchID    *string                 `json:"twitch_id"`
	Permissions []repository.Permission `json:"permissions"`
}

type MinimalUserResponse struct {
	ID          int    `json:"id"`
	DisplayName string `json:"display_name"`
}

func toUserResponse(user *repository.User) UserResponse {
	permissions := make([]repository.Permission, len(user.Permissions))
	for i, perm := range user.Permissions {
		permissions[i] = repository.Permission(perm)
	}
	response := UserResponse{
		ID:                   user.ID,
		AcountName:           user.POEAccount,
		DisplayName:          user.DisplayName,
		DiscordName:          user.DiscordName,
		TwitchID:             user.TwitchID,
		TwitchName:           user.TwitchName,
		TokenExpiryTimestamp: user.PoeTokenExpiresAt,
		Permissions:          permissions,
	}
	if user.DiscordID != nil {
		discordIdString := strconv.FormatInt(*user.DiscordID, 10)
		response.DiscordID = &discordIdString
	}
	return response
}

func toNonSensitiveUserResponse(user *repository.User) *NonSensitiveUserResponse {
	if user == nil {
		return nil
	}
	response := &NonSensitiveUserResponse{
		ID:          user.ID,
		AcountName:  user.POEAccount,
		DisplayName: user.DisplayName,
		DiscordName: user.DiscordName,
		TwitchID:    user.TwitchID,
		TwitchName:  user.TwitchName,
	}
	if user.DiscordID != nil {
		discordIdString := strconv.FormatInt(*user.DiscordID, 10)
		response.DiscordID = &discordIdString
	}
	return response
}

func toUserAdminResponse(user *repository.User) UserAdminResponse {
	permissions := make([]repository.Permission, len(user.Permissions))
	for i, perm := range user.Permissions {
		permissions[i] = repository.Permission(perm)
	}
	response := UserAdminResponse{
		ID:          user.ID,
		AcountName:  user.POEAccount,
		DisplayName: user.DisplayName,
		DiscordName: user.DiscordName,
		TwitchName:  user.TwitchName,
		TwitchID:    user.TwitchID,
		Permissions: permissions,
	}
	if user.DiscordID != nil {
		discordIdString := strconv.FormatInt(*user.DiscordID, 10)
		response.DiscordID = &discordIdString
	}
	return response
}

func toMinimalUserResponse(user *repository.User) *MinimalUserResponse {
	return &MinimalUserResponse{
		ID:          user.ID,
		DisplayName: user.DisplayName,
	}
}
