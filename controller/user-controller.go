package controller

import (
	"bpl/repository"
	"bpl/service"
	"net/http"

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
		DiscordID:   user.DiscordID,
		DiscordName: user.DiscordName,
		PoEToken:    user.PoeToken,
		Permissions: permissions,
	}
}

func setupUserController(db *gorm.DB) []RouteInfo {
	e := NewUserController(db)
	basePath := "/users"
	routes := []RouteInfo{
		{Method: "GET", Path: "", HandlerFunc: e.getUserHandler(), Authenticated: true},
		{Method: "POST", Path: "/logout", HandlerFunc: e.logoutHandler(), Authenticated: true},
	}
	for i, route := range routes {
		routes[i].Path = basePath + route.Path
	}
	return routes
}

func (e *UserController) getUserHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := e.userService.GetUserFromAuthCookie(c)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(404, gin.H{"error": "User not found"})
			} else if err == http.ErrNoCookie {
				c.JSON(401, gin.H{"error": "Not authenticated"})
			} else {
				c.JSON(500, gin.H{"error": err.Error()})
			}
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
	DiscordID   int64                   `json:"discord_id"`
	DiscordName string                  `json:"discord_name"`
	PoEToken    string                  `json:"poe_token"`
	Permissions []repository.Permission `json:"permissions"`
}
