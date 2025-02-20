package controller

import (
	"bpl/auth"
	"bpl/repository"
	"bpl/service"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
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

func NewOauthController() *OauthController {
	return &OauthController{
		oauthService: service.NewOauthService(),
		userService:  service.NewUserService(),
	}
}

func setupOauthController() []RouteInfo {
	e := NewOauthController()
	basePath := "/oauth2"
	routes := []RouteInfo{
		{Method: "GET", Path: "/discord", HandlerFunc: e.discordOauthHandler()},
		{Method: "GET", Path: "/discord/redirect", HandlerFunc: e.discordRedirectHandler()},
		{Method: "POST", Path: "/discord/bot-login", HandlerFunc: e.loginDiscordBotHandler()},

		{Method: "GET", Path: "/twitch", HandlerFunc: e.twitchOauthHandler()},
		{Method: "GET", Path: "/twitch/redirect", HandlerFunc: e.twitchRedirectHandler()},
	}
	for i, route := range routes {
		routes[i].Path = basePath + route.Path
	}
	return routes
}

type DiscordBotLoginBody struct {
	Token string `json:"token" binding:"required"`
}

// @id LoginDiscordBot
// @Description Logs in the discord bot (only for internal use)
// @Tags oauth
// @Accept json
// @Param body body DiscordBotLoginBody true "Discord bot login body"
// @Produce json
// @Success 200 {string} authToken
// @Router /oauth2/discord/bot-login [post]
func (e *OauthController) loginDiscordBotHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var body DiscordBotLoginBody
		c.BindJSON(&body)
		if body.Token != os.Getenv("DISCORD_BOT_TOKEN") {
			c.JSON(401, gin.H{"error": "Invalid token"})
			return
		}

		authToken, _ := auth.CreateToken(
			&repository.User{
				ID:          0,
				DisplayName: "bot",
				Permissions: []repository.Permission{repository.PermissionAdmin},
			},
		)
		c.JSON(200, authToken)
	}
}

// @Description Redirects to discord oauth
// @Tags oauth
// @Produce json
// @Success 302
// @Router /oauth2/discord [get]
func (e *OauthController) discordOauthHandler() gin.HandlerFunc {
	return e.redirectHandler(repository.ProviderDiscord)
}

// @Description Redirects to twitch oauth
// @Tags oauth
// @Produce json
// @Success 302
// @Router /oauth2/twitch [get]
func (e *OauthController) twitchOauthHandler() gin.HandlerFunc {
	return e.redirectHandler(repository.ProviderTwitch)
}

func (e *OauthController) redirectHandler(provider repository.Provider) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _ := e.userService.GetUserFromAuthCookie(c)
		url := e.oauthService.GetRedirectUrl(user, provider)
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
		e.handleOauthResponse(c, repository.ProviderDiscord)
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
		e.handleOauthResponse(c, repository.ProviderTwitch)
	}
}

func (e *OauthController) handleOauthResponse(c *gin.Context, provider repository.Provider) {
	code := c.Request.URL.Query().Get("code")
	state := c.Request.URL.Query().Get("state")
	user, err := e.oauthService.Verify(state, code, provider)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	authToken, _ := auth.CreateToken(user)
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie("auth", authToken, 60*60*24*7, "/", os.Getenv("PUBLIC_DOMAIN"), false, true)
	c.HTML(http.StatusOK, "auth-closing.html", gin.H{})
}
