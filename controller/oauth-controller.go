package controller

import (
	"bpl/service"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
)

type Verifier struct {
	Verifier string
	Timeout  int64
}

type OauthController struct {
	discordConfig   *oauth2.Config
	stateToVerifyer map[string]Verifier
	UserService     *service.UserService
}

type DiscordUserResponse struct {
	ID                   string `json:"id"`
	Username             string `json:"username"`
	Avatar               string `json:"avatar"`
	Discriminator        string `json:"discriminator"`
	PublicFlags          int    `json:"public_flags"`
	Flags                int    `json:"flags"`
	Banner               string `json:"banner"`
	AccentColor          int    `json:"accent_color"`
	GlobalName           string `json:"global_name"`
	AvatarDecorationData string `json:"avatar_decoration_data"`
	BannerColor          string `json:"banner_color"`
	Clan                 string `json:"clan"`
	PrimaryGuild         string `json:"primary_guild"`
	MfaEnabled           bool   `json:"mfa_enabled"`
	Locale               string `json:"locale"`
	PremiumType          int    `json:"premium_type"`
}

func NewOauthController(db *gorm.DB) *OauthController {
	return &OauthController{
		discordConfig: &oauth2.Config{
			// todo: fill with env secrets
			ClientID:     "xxx",
			ClientSecret: "xxx",
			Scopes:       []string{"identify"},
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://discord.com/oauth2/authorize",
				TokenURL: "https://discord.com/api/oauth2/token",
			},
			RedirectURL: "https://redirectmeto.com/http://localhost:8000/oauth2/discord/redirect",
		},
		// small hashmap that is used to associate states with verifiers
		stateToVerifyer: make(map[string]Verifier),
		UserService:     service.NewUserService(db),
	}
}

func setupOauthController(db *gorm.DB) []gin.RouteInfo {
	e := NewOauthController(db)
	basePath := "/oauth2"
	routes := []gin.RouteInfo{
		{Method: "GET", Path: "/discord", HandlerFunc: e.getDiscordOauthHandler()},
		{Method: "GET", Path: "/discord/redirect", HandlerFunc: e.getDiscordRedirectHandler()},
	}
	for i, route := range routes {
		routes[i].Path = basePath + route.Path
	}
	return routes
}

func (e *OauthController) getNewVerifier() (string, string) {
	// clean up old verifiers
	for verifier, v := range e.stateToVerifyer {
		if v.Timeout < time.Now().Unix() {
			delete(e.stateToVerifyer, verifier)
		}
	}
	state := oauth2.GenerateVerifier()
	verifier := oauth2.GenerateVerifier()
	e.stateToVerifyer[state] = Verifier{
		Verifier: verifier,
		Timeout:  time.Now().Add(1 * time.Minute).Unix(),
	}
	return state, verifier
}

func (e *OauthController) getDiscordOauthHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		state, verifier := e.getNewVerifier()
		url := e.discordConfig.AuthCodeURL(state, oauth2.SetAuthURLParam("code_challenge", oauth2.S256ChallengeFromVerifier(verifier)))
		c.Redirect(http.StatusTemporaryRedirect, url)
	}
}

func (e *OauthController) getDiscordRedirectHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		code := c.Request.URL.Query().Get("code")
		state := c.Request.URL.Query().Get("state")
		verifier, ok := e.stateToVerifyer[state]
		if !ok {
			c.JSON(400, gin.H{"error": "state is unknown"})
			return
		}
		token, err := e.discordConfig.Exchange(c, code, oauth2.SetAuthURLParam("code_verifier", verifier.Verifier))
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
		}
		response, err := e.discordConfig.Client(c, token).Get("https://discord.com/api/users/@me")
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
		}
		defer response.Body.Close()
		discordUser := &DiscordUserResponse{}
		err = json.NewDecoder(response.Body).Decode(discordUser)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
		}
		discordId, err := strconv.ParseInt(discordUser.ID, 10, 64)
		if err != nil {
			c.JSON(500, gin.H{"error": "discord id is not a number"})
		}
		user, err := e.UserService.GetOrCreateUserByDiscordId(discordId)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
		}
		c.JSON(200, user)
	}
}
