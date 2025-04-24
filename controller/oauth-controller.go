package controller

import (
	"bpl/auth"
	"bpl/repository"
	"bpl/service"
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
		{Method: "POST", Path: "/:provider/callback", HandlerFunc: e.callbackHandler()},
		{Method: "GET", Path: "/:provider/redirect", HandlerFunc: e.oauthRedirectHandler()},
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

// @Id OauthRedirect
// @Description Redirects to an oauth provider
// @Tags oauth
// @Security BearerAuth
// @Param provider path repository.Provider true "Provider name"
// @Param redirect_url query string false "Redirect URL for oauth provider"
// @Param last_url query string false "Last URL to redirect to after oauth is finished"
// @Success 200 {string} string
// @Router /oauth2/{provider}/redirect [get]
func (e *OauthController) oauthRedirectHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		provider := repository.Provider(c.Param("provider"))
		if provider == "" {
			c.JSON(400, gin.H{"error": "Invalid provider"})
			return
		}
		user, _ := e.userService.GetUserFromAuthHeader(c)
		lastUrl := c.Request.URL.Query().Get("last_url")
		redirectUrl := c.Request.URL.Query().Get("redirect_url")
		url := e.oauthService.GetOauthProviderUrl(user, provider, lastUrl, redirectUrl)
		c.JSON(200, url)
	}
}

// @Description Callback handler for oauth
// @Id OauthCallback
// @Tags oauth
// @Accept json
// @Param provider path repository.Provider true "Provider name"
// @Param body body CallbackBody true "Callback body"
// @Success 200 {object} CallbackResponse
// @Router /oauth2/{provider}/callback [post]
func (e *OauthController) callbackHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		provider := repository.Provider(c.Param("provider"))
		if provider == "" {
			c.JSON(400, gin.H{"error": "Invalid provider"})
			return
		}
		var body CallbackBody
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		config := *e.oauthService.Config[provider]
		config.RedirectURL = body.RedirectUrl
		verifier, err := e.oauthService.Verify(body.State, body.Code, provider, config)
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
	RedirectUrl string `json:"redirect_url" binding:"required"`
	Code        string `json:"code" binding:"required"`
	State       string `json:"state" binding:"required"`
}

type CallbackResponse struct {
	LastPath  string `json:"last_path" binding:"required"`
	AuthToken string `json:"auth_token" binding:"required"`
	User      User   `json:"user" binding:"required"`
}
