package controller

import (
	"bpl/auth"
	"bpl/repository"
	"bpl/service"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
)

type Verifier struct {
	Verifier string
	Timeout  int64
	User     *repository.User
}

type OauthController struct {
	discordConfig   *oauth2.Config
	twitchConfig    *oauth2.Config
	stateToVerifyer map[string]Verifier
	userService     *service.UserService
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

type TwitchUserResponse struct {
	Aud            string `json:"aud"`
	Exp            int64  `json:"exp"`
	Iat            int64  `json:"iat"`
	Iss            string `json:"iss"`
	Sub            string `json:"sub"`
	Email          string `json:"email"`
	Email_verified bool   `json:"email_verified"`
	Picture        string `json:"picture"`
	Updated_at     string `json:"updated_at"`
}

type TwitchExtendedUserResponse struct {
	Data []struct {
		ID              string `json:"id"`
		Login           string `json:"login"`
		DisplayName     string `json:"display_name"`
		Type            string `json:"type"`
		BroadcasterType string `json:"broadcaster_type"`
		Description     string `json:"description"`
		ProfileImageUrl string `json:"profile_image_url"`
		OfflineImageUrl string `json:"offline_image_url"`
		ViewCount       int    `json:"view_count"`
		Email           string `json:"email"`
		CreatedAt       string `json:"created_at"`
	} `json:"data"`
}

func NewOauthController(db *gorm.DB) *OauthController {
	return &OauthController{
		discordConfig: &oauth2.Config{
			ClientID:     os.Getenv("DISCORD_CLIENT_ID"),
			ClientSecret: os.Getenv("DISCORD_CLIENT_SECRET"),
			Scopes:       []string{"identify"},
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://discord.com/oauth2/authorize",
				TokenURL: "https://discord.com/api/oauth2/token",
			},
			RedirectURL: fmt.Sprintf("https://redirectmeto.com/%s/api/oauth2/discord/redirect", os.Getenv("PUBLIC_URL")),
		},
		twitchConfig: &oauth2.Config{
			ClientID:     os.Getenv("TWITCH_CLIENT_ID"),
			ClientSecret: os.Getenv("TWITCH_CLIENT_SECRET"),
			Scopes:       []string{},
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://id.twitch.tv/oauth2/authorize?claims={\"id_token\":{\"email\":true}}",
				TokenURL: "https://id.twitch.tv/oauth2/token",
			},
			RedirectURL: fmt.Sprintf("https://redirectmeto.com/%s/api/oauth2/twitch/redirect", os.Getenv("PUBLIC_URL")),
		},

		// small hashmap that is used to associate states with verifiers
		stateToVerifyer: make(map[string]Verifier),
		userService:     service.NewUserService(db),
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

func (e *OauthController) getNewVerifier(user *repository.User) (string, string) {
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
		User:     user,
	}
	return state, verifier
}

func (e *OauthController) discordOauthHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _ := e.userService.GetUserFromAuthCookie(c)
		state, verifier := e.getNewVerifier(user)
		url := e.discordConfig.AuthCodeURL(state, oauth2.SetAuthURLParam("code_challenge", oauth2.S256ChallengeFromVerifier(verifier)))
		c.Redirect(http.StatusTemporaryRedirect, url)
	}
}

func (e *OauthController) twitchOauthHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _ := e.userService.GetUserFromAuthCookie(c)
		state, verifier := e.getNewVerifier(user)
		url := e.twitchConfig.AuthCodeURL(state, oauth2.SetAuthURLParam("code_challenge", oauth2.S256ChallengeFromVerifier(verifier)))
		c.Redirect(http.StatusTemporaryRedirect, url)
	}
}

func (e *OauthController) discordRedirectHandler() gin.HandlerFunc {
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
			return
		}
		response, err := e.discordConfig.Client(c, token).Get("https://discord.com/api/users/@me")
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		defer response.Body.Close()
		discordUser := &DiscordUserResponse{}
		json.NewDecoder(response.Body).Decode(discordUser)
		discordId, err := strconv.ParseInt(discordUser.ID, 10, 64)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		user := &repository.User{}
		if verifier.User != nil {
			user = verifier.User
		} else {
			user, err = e.userService.GetUserByDiscordId(discordId)
			if err != nil {
				verifier.User = &repository.User{
					Permissions: []repository.Permission{},
					DisplayName: discordUser.Username,
				}
			}
		}
		user.DiscordID = discordId
		user.DiscordName = discordUser.Username
		user, err = e.userService.SaveUser(user)
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

func (e *OauthController) twitchRedirectHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		code := c.Request.URL.Query().Get("code")
		state := c.Request.URL.Query().Get("state")
		verifier, ok := e.stateToVerifyer[state]
		if !ok {
			c.JSON(400, gin.H{"error": "state is unknown"})
			return
		}
		token, err := e.twitchConfig.Exchange(c, code, oauth2.SetAuthURLParam("code_verifier", verifier.Verifier))
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		response, err := e.twitchConfig.Client(c, token).Get("https://id.twitch.tv/oauth2/userinfo")
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		twitchUser := &TwitchUserResponse{}
		json.NewDecoder(response.Body).Decode(twitchUser)
		response.Body.Close()
		twitchId := twitchUser.Sub
		// todo: figure out why this request is running into 401 when using the oauth client
		req := &http.Request{
			URL: &url.URL{
				Scheme:   "https",
				Host:     "api.twitch.tv",
				Path:     "/helix/users",
				RawQuery: "id=" + twitchId,
			},
			Header: http.Header{
				"Authorization": {"Bearer " + token.AccessToken},
				"Client-Id":     {os.Getenv("TWITCH_CLIENT_ID")},
			},
		}
		client := &http.Client{}
		response, err = client.Do(req)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		twitchExtendedUser := &TwitchExtendedUserResponse{}
		json.NewDecoder(response.Body).Decode(twitchExtendedUser)
		response.Body.Close()

		user := &repository.User{}
		if verifier.User != nil {
			user = verifier.User
		} else {
			user, err = e.userService.GetUserByTwitchId(twitchId)
			if err != nil {
				user = &repository.User{
					DisplayName: twitchExtendedUser.Data[0].DisplayName,
					Permissions: []repository.Permission{},
				}
			}
		}
		user.TwitchID = twitchId
		user.TwitchName = twitchExtendedUser.Data[0].DisplayName

		user, err = e.userService.SaveUser(user)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		authToken, _ := auth.CreateToken(user)
		c.SetSameSite(http.SameSiteStrictMode)
		// TODO: Check if we still need to set security flag to false when we are using https
		// for now it seems to be required for the cookie being set when the application is running on the server
		c.SetCookie("auth", authToken, 60*60*24*7, "/", os.Getenv("PUBLIC_DOMAIN"), false, true)
		c.HTML(http.StatusOK, "auth-closing.html", gin.H{})
	}
}
