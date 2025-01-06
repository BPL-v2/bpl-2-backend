package controller

import (
	"bpl/auth"
	"bpl/repository"
	"bpl/service"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Verifier struct {
	Verifier string
	Timeout  int64
	User     *repository.User
}

type OauthController struct {
	oauthService *service.OauthService
	userService  *service.UserService
}

func NewOauthController(db *gorm.DB) *OauthController {
	return &OauthController{
		oauthService: service.NewOauthService(db),
		userService:  service.NewUserService(db),
	}
}

func setupOauthController(db *gorm.DB) []RouteInfo {
	e := NewOauthController(db)
	basePath := "/oauth2"
	routes := []RouteInfo{
		{Method: "GET", Path: "/discord", HandlerFunc: e.discordOauthHandler()},
		{Method: "GET", Path: "/discord/redirect", HandlerFunc: e.discordRedirectHandler()},
		{Method: "GET", Path: "/twitch", HandlerFunc: e.twitchOauthHandler()},
		{Method: "GET", Path: "/twitch/redirect", HandlerFunc: e.twitchRedirectHandler()},
	}
	for i, route := range routes {
		routes[i].Path = basePath + route.Path
	}
	return routes
}

// @Description Redirects to discord oauth
// @Tags oauth
// @Produce json
// @Success 302
// @Router /oauth2/discord [get]
func (e *OauthController) discordOauthHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _ := e.userService.GetUserFromAuthCookie(c)
		url := e.oauthService.GetRedirectUrl(user, repository.ProviderDiscord)
		c.Redirect(http.StatusTemporaryRedirect, url)
	}
}

// @Description Redirects to twitch oauth
// @Tags oauth
// @Produce json
// @Success 302
// @Router /oauth2/twitch [get]
func (e *OauthController) twitchOauthHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _ := e.userService.GetUserFromAuthCookie(c)
		url := e.oauthService.GetRedirectUrl(user, repository.ProviderTwitch)
		c.Redirect(http.StatusTemporaryRedirect, url)
	}
}

// @Description Redirect handler for discord oauth
// @Tags oauth
// @Produce html
// @Success 200
// @Router /oauth2/discord/redirect [get]
func (e *OauthController) discordRedirectHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		code := c.Request.URL.Query().Get("code")
		state := c.Request.URL.Query().Get("state")
		user, err := e.oauthService.VerifyDiscord(state, code)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		authToken, _ := auth.CreateToken(user)
		c.SetSameSite(http.SameSiteStrictMode)
		c.SetCookie("auth", authToken, 60*60*24*7, "/", os.Getenv("PUBLIC_DOMAIN"), false, true)
		c.HTML(http.StatusOK, "auth-closing.html", gin.H{})
	}
}

// @Description Redirect handler for twitch oauth
// @Tags oauth
// @Produce html
// @Success 200
// @Router /oauth2/twitch/redirect [get]
func (e *OauthController) twitchRedirectHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		errorString := c.Request.URL.Query().Get("error")
		if errorString != "" {
			c.JSON(400, gin.H{"error": errorString + ": " + c.Request.URL.Query().Get("error_description")})
			return
		}
		code := c.Request.URL.Query().Get("code")
		state := c.Request.URL.Query().Get("state")
		user, err := e.oauthService.VerifyTwitch(state, code)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		authToken, _ := auth.CreateToken(user)
		c.SetSameSite(http.SameSiteStrictMode)
		c.SetCookie("auth", authToken, 60*60*24*7, "/", os.Getenv("PUBLIC_DOMAIN"), false, true)
		c.HTML(http.StatusOK, "auth-closing.html", gin.H{})
	}
}
