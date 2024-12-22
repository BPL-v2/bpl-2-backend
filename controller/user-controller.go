package controller

import (
	"bpl/repository"
	"bpl/service"
	"bpl/utils"
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

func toUserResponse(user *repository.User) UserResponse {
	permissions := make([]repository.Permission, len(user.Permissions))
	for i, perm := range user.Permissions {
		permissions[i] = repository.Permission(perm)
	}
	return UserResponse{
		ID:          user.ID,
		AcountName:  user.AccountName,
		DiscordID:   strconv.FormatInt(user.DiscordID, 10),
		DiscordName: user.DiscordName,
		PoEToken:    user.PoeToken,
		Permissions: permissions,
	}
}

func toNonSensitiveUserResponse(user *repository.User) NonSensitiveUserResponse {
	return NonSensitiveUserResponse{
		ID:          user.ID,
		AcountName:  user.AccountName,
		DiscordID:   strconv.FormatInt(user.DiscordID, 10),
		DiscordName: user.DiscordName,
	}
}

func toUserAdminResponse(user *repository.User) UserAdminResponse {
	permissions := make([]repository.Permission, len(user.Permissions))
	for i, perm := range user.Permissions {
		permissions[i] = repository.Permission(perm)
	}
	return UserAdminResponse{
		ID:          user.ID,
		AcountName:  user.AccountName,
		DiscordID:   strconv.FormatInt(user.DiscordID, 10),
		DiscordName: user.DiscordName,
		Permissions: permissions,
	}
}

func setupUserController(db *gorm.DB) []RouteInfo {
	e := NewUserController(db)
	basePath := "/users"
	routes := []RouteInfo{
		{Method: "GET", Path: "", HandlerFunc: e.getUsersHandler(), Authenticated: true, RequiredRoles: []repository.Permission{repository.PermissionAdmin}},
		{Method: "GET", Path: "/self", HandlerFunc: e.getUserHandler(), Authenticated: true},
		{Method: "PATCH", Path: "/:userId", HandlerFunc: e.changePermissionsHandler(), Authenticated: true, RequiredRoles: []repository.Permission{repository.PermissionAdmin}},
		{Method: "POST", Path: "/logout", HandlerFunc: e.logoutHandler(), Authenticated: true},
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

type UserResponse struct {
	ID          int                     `json:"id"`
	AcountName  string                  `json:"account_name"`
	DiscordID   string                  `json:"discord_id"`
	DiscordName string                  `json:"discord_name"`
	PoEToken    string                  `json:"poe_token"`
	Permissions []repository.Permission `json:"permissions"`
}

type NonSensitiveUserResponse struct {
	ID          int    `json:"id"`
	AcountName  string `json:"account_name"`
	DiscordID   string `json:"discord_id"`
	DiscordName string `json:"discord_name"`
}

type UserAdminResponse struct {
	ID          int                     `json:"id"`
	AcountName  string                  `json:"account_name"`
	DiscordID   string                  `json:"discord_id"`
	DiscordName string                  `json:"discord_name"`
	Permissions []repository.Permission `json:"permissions"`
}
