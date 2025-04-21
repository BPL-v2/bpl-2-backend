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
		{Method: "POST", Path: "/callback", HandlerFunc: e.callbackHandler()},
		{Method: "GET", Path: "/discord", HandlerFunc: e.discordOauthHandler()},
		{Method: "GET", Path: "/twitch", HandlerFunc: e.twitchOauthHandler()},
		{Method: "POST", Path: "/discord/bot-login", HandlerFunc: e.loginDiscordBotHandler()},
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
				Id:          0,
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
		user, _ := e.userService.GetUserFromAuthHeader(c)
		lastUrl := c.Request.URL.Query().Get("last_url")
		url := e.oauthService.GetRedirectUrl(user, provider, lastUrl)
		c.Redirect(http.StatusTemporaryRedirect, url)
	}
}

// @Description Callback handler for oauth
// @Id OauthCallback
// @Tags oauth
// @Accept json
// @Param body body CallbackBody true "Callback body"
// @Success 200 {object} CallbackResponse
// @Router /oauth2/callback [post]
func (e *OauthController) callbackHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var body CallbackBody
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		verifier, err := e.oauthService.Verify(body.State, body.Code, body.Provider)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		authToken, _ := auth.CreateToken(verifier.User)
		c.JSON(200,
			CallbackResponse{
				LastPath:  verifier.Redirect,
				AuthToken: authToken,
				User:      *toUserResponse(verifier.User),
			},
		)
	}
}

type CallbackBody struct {
	Provider repository.Provider `json:"provider" binding:"required"`
	Code     string              `json:"code" binding:"required"`
	State    string              `json:"state" binding:"required"`
}

type CallbackResponse struct {
	LastPath  string `json:"last_path" binding:"required"`
	AuthToken string `json:"auth_token" binding:"required"`
	User      User   `json:"user" binding:"required"`
}
