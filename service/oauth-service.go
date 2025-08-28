package service

import (
	"bpl/client"
	"bpl/config"
	"bpl/repository"
	"bpl/utils"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

type OauthState struct {
	Verifier    string
	Timeout     int64
	User        *repository.User
	LastUrl     string
	RedirectUrl string
}

type OauthService struct {
	Config                     map[repository.Provider]*oauth2.Config
	clientConfig               map[repository.Provider]*clientcredentials.Config
	stateMap                   map[string]OauthState
	userService                *UserService
	clientCredentialRepository *repository.ClientCredentialsRepository
	oauthRepository            *repository.OauthRepository
}

type DiscordUserResponse struct {
	Id                   string `json:"id"`
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
		Id              string `json:"id"`
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

func NewOauthService() *OauthService {
	return &OauthService{
		Config: map[repository.Provider]*oauth2.Config{
			repository.ProviderDiscord: {
				ClientID:     config.Env().DiscordClientID,
				ClientSecret: config.Env().DiscordClientSecret,
				Scopes:       []string{"identify"},
				Endpoint: oauth2.Endpoint{
					AuthURL:  "https://discord.com/oauth2/authorize",
					TokenURL: "https://discord.com/api/oauth2/token",
				},
			},
			repository.ProviderTwitch: {
				ClientID:     config.Env().TwitchClientID,
				ClientSecret: config.Env().TwitchClientSecret,
				Scopes:       []string{},
				Endpoint: oauth2.Endpoint{
					AuthURL:  "https://id.twitch.tv/oauth2/authorize",
					TokenURL: "https://id.twitch.tv/oauth2/token",
				},
			},
			repository.ProviderPoE: {
				ClientID:     config.Env().POEClientID,
				ClientSecret: config.Env().POEClientSecret,
				Scopes:       []string{"account:profile", "account:characters", "account:league_accounts", "account:guild:stashes"},
				Endpoint: oauth2.Endpoint{
					AuthURL:  "https://www.pathofexile.com/oauth/authorize",
					TokenURL: "https://www.pathofexile.com/oauth/token",
				},
			},
		},
		clientConfig: map[repository.Provider]*clientcredentials.Config{
			repository.ProviderTwitch: {
				ClientID:     config.Env().TwitchClientID,
				ClientSecret: config.Env().TwitchClientSecret,
				TokenURL:     "https://id.twitch.tv/oauth2/token",
			},
			repository.ProviderPoE: {
				ClientID:     config.Env().POEClientID,
				ClientSecret: config.Env().POEClientSecret,
				TokenURL:     "https://www.pathofexile.com/oauth/token",
				Scopes:       []string{"service:psapi"},
			},
		},

		stateMap:                   make(map[string]OauthState),
		userService:                NewUserService(),
		clientCredentialRepository: repository.NewClientCredentialsRepository(),
		oauthRepository:            repository.NewOauthRepository(),
	}
}

func (e *OauthService) GetNewVerifier(user *repository.User, lastUrl string, redirectUrl string) (string, string) {
	// clean up old verifiers
	for verifier, v := range e.stateMap {
		if v.Timeout < time.Now().Unix() {
			delete(e.stateMap, verifier)
		}
	}
	state := oauth2.GenerateVerifier()
	verifier := oauth2.GenerateVerifier()
	e.stateMap[state] = OauthState{
		Verifier:    verifier,
		Timeout:     time.Now().Add(1 * time.Minute).Unix(),
		User:        user,
		LastUrl:     lastUrl,
		RedirectUrl: redirectUrl,
	}
	return state, verifier
}

func (e *OauthService) GetOauthProviderUrl(user *repository.User, provider repository.Provider, lastUrl string, redirectUrl string) string {
	state, verifier := e.GetNewVerifier(user, lastUrl, redirectUrl)
	config := e.Config[provider]
	config.RedirectURL = redirectUrl
	return config.AuthCodeURL(
		state,
		oauth2.SetAuthURLParam("code_challenge", oauth2.S256ChallengeFromVerifier(verifier)),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)
}

func (e *OauthService) Verify(state string, code string, provider repository.Provider, oauthConfig oauth2.Config) (*OauthState, error) {
	switch provider {
	case repository.ProviderDiscord:
		return e.VerifyDiscord(state, code, oauthConfig)
	case repository.ProviderTwitch:
		return e.VerifyTwitch(state, code, oauthConfig)
	case repository.ProviderPoE:
		return e.VerifyPoE(state, code, oauthConfig)
	default:
		return nil, fmt.Errorf("not implemented")
	}
}

func (e *OauthService) addAccountToUser(authState *OauthState, accountId string, accountName string, token *oauth2.Token, provider repository.Provider) (*OauthState, error) {
	if authState.User == nil {
		user, err := e.userService.GetUserByOauthProviderAndAccountId(provider, accountId)
		if err != nil {
			user = &repository.User{
				Permissions:   []repository.Permission{},
				DisplayName:   accountName,
				OauthAccounts: []*repository.Oauth{},
			}
		}
		authState.User = user
	}
	authState.User.OauthAccounts = append(
		utils.Filter(authState.User.OauthAccounts, func(oauthAccount *repository.Oauth) bool {
			return oauthAccount.Provider != provider
		}),
		&repository.Oauth{
			UserId:       authState.User.Id,
			Provider:     provider,
			AccessToken:  token.AccessToken,
			RefreshToken: token.RefreshToken,
			AccountId:    accountId,
			Name:         accountName,
			Expiry:       token.Expiry,
		},
	)
	/* 		err := e.oauthRepository.DeleteOauthsByUserId(authState.User.Id)
	if err != nil {
		return nil, err
	} */
	_, err := e.userService.SaveUser(authState.User)
	return authState, err
}
func (e *OauthService) fetchToken(oauthConfig oauth2.Config, state string, code string) (*OauthState, *oauth2.Token, error) {
	authState, ok := e.stateMap[state]
	if !ok {
		return nil, nil, fmt.Errorf("state is unknown")
	}
	oauthConfig.RedirectURL = authState.RedirectUrl

	token, err := oauthConfig.Exchange(context.Background(), code, oauth2.SetAuthURLParam("code_verifier", authState.Verifier))
	if err != nil {
		return nil, nil, err
	}
	return &authState, token, nil
}

func (e *OauthService) VerifyDiscord(state string, code string, oauthConfig oauth2.Config) (*OauthState, error) {

	authState, token, err := e.fetchToken(oauthConfig, state, code)
	if err != nil {
		return nil, err
	}
	client := oauthConfig.Client(context.Background(), token)
	response, err := client.Get("https://discord.com/api/users/@me")
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	discordUser := &DiscordUserResponse{}
	err = json.NewDecoder(response.Body).Decode(discordUser)
	if err != nil {
		return nil, fmt.Errorf("failed to decode discord user response: %v", err)
	}
	return e.addAccountToUser(authState, discordUser.Id, discordUser.Username, token, repository.ProviderDiscord)
}

func (e *OauthService) VerifyTwitch(state string, code string, oauthConfig oauth2.Config) (*OauthState, error) {
	authState, token, err := e.fetchToken(oauthConfig, state, code)
	if err != nil {
		return nil, err
	}
	response, err := e.Config[repository.ProviderTwitch].Client(context.Background(), token).Get("https://id.twitch.tv/oauth2/userinfo")
	if err != nil {
		return nil, err
	}
	twitchUser := &TwitchUserResponse{}
	err = json.NewDecoder(response.Body).Decode(twitchUser)
	response.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to decode twitch user response: %v", err)
	}
	twitchId := twitchUser.Sub

	req := &http.Request{
		URL: &url.URL{
			Scheme:   "https",
			Host:     "api.twitch.tv",
			Path:     "/helix/users",
			RawQuery: "id=" + twitchId,
		},
		Header: http.Header{
			"Authorization": {"Bearer " + token.AccessToken},
			"Client-Id":     {config.Env().TwitchClientID},
		},
	}
	client := &http.Client{}
	response, err = client.Do(req)
	if err != nil {
		return nil, err
	}
	twitchExtendedUser := &TwitchExtendedUserResponse{}
	err = json.NewDecoder(response.Body).Decode(twitchExtendedUser)
	response.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to decode twitch extended user response: %v", err)
	}
	return e.addAccountToUser(authState, twitchId, twitchExtendedUser.Data[0].DisplayName, token, repository.ProviderTwitch)
}

func (e *OauthService) VerifyPoE(state string, code string, oauthConfig oauth2.Config) (*OauthState, error) {
	client := client.NewPoEClient(1, true, 10)
	authState, ok := e.stateMap[state]
	if !ok {
		return nil, fmt.Errorf("state is unknown")
	}
	resp, clientError := client.GetAccessToken(oauthConfig.ClientID, oauthConfig.ClientSecret, code, authState.Verifier, oauthConfig.Scopes, authState.RedirectUrl)
	if clientError != nil {
		return nil, fmt.Errorf("failed to get access token: %v", clientError)
	}
	token := &oauth2.Token{
		AccessToken:  resp.AccessToken,
		TokenType:    resp.TokenType,
		RefreshToken: resp.RefreshToken,
		Expiry:       time.Now().Add(time.Duration(resp.ExpiresIn) * time.Second),
	}
	profile, clientError := client.GetAccountProfile(token.AccessToken)
	if clientError != nil {
		return nil, fmt.Errorf("failed to get profile: %v", clientError)
	}
	return e.addAccountToUser(&authState, profile.UUId, profile.Name, token, repository.ProviderPoE)
}

func (e *OauthService) GetApplicationToken(provider repository.Provider) (string, error) {
	credentials, err := e.clientCredentialRepository.GetClientCredentialsByName(provider)
	if err != nil || (credentials.Expiry != nil && credentials.Expiry.Before(time.Now())) {
		token, expiry, err := e.GetToken(provider)
		if err != nil {
			return "", err
		}
		if credentials == nil {
			credentials = &repository.ClientCredentials{
				Name:        provider,
				AccessToken: token,
				Expiry:      expiry,
			}
		} else {
			credentials.AccessToken = token
			credentials.Expiry = expiry
		}
		e.clientCredentialRepository.DB.Save(credentials)
	}
	return credentials.AccessToken, nil
}

func (e *OauthService) GetToken(provider repository.Provider) (token string, expiry *time.Time, err error) {
	if provider == repository.ProviderPoE {
		poeClient := client.NewPoEClient(1, false, 10)
		tokenResponse, hhtpErr := poeClient.GetClientCredentials(config.Env().POEClientID, config.Env().POEClientSecret)
		if hhtpErr != nil {
			return "", nil, fmt.Errorf("failed to get PoE token: %s", hhtpErr.Description)
		}
		var expiry *time.Time = nil
		if tokenResponse.ExpiresIn != nil {
			x := time.Now().Add(time.Duration(*tokenResponse.ExpiresIn) * time.Second)
			expiry = &x
		}
		return tokenResponse.AccessToken, expiry, nil
	}

	config, ok := e.clientConfig[provider]
	if !ok {
		return "", nil, fmt.Errorf("provider not found")
	}
	if config.ClientID == "" || config.ClientSecret == "" {
		return "", nil, fmt.Errorf("client ID or secret not set")
	}
	oauthToken, err := config.Token(context.Background())
	if err != nil {
		return "", nil, err
	}
	return oauthToken.AccessToken, &oauthToken.Expiry, nil
}
