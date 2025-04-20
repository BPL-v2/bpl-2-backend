package service

import (
	"bpl/repository"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

type Verifier struct {
	Verifier string
	Timeout  int64
	User     *repository.User
}

type OauthService struct {
	config                     map[repository.Provider]*oauth2.Config
	clientConfig               map[repository.Provider]*clientcredentials.Config
	stateToVerifyer            map[string]Verifier
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
		config: map[repository.Provider]*oauth2.Config{
			repository.ProviderDiscord: {
				ClientID:     os.Getenv("DISCORD_CLIENT_ID"),
				ClientSecret: os.Getenv("DISCORD_CLIENT_SECRET"),
				Scopes:       []string{"identify"},
				Endpoint: oauth2.Endpoint{
					AuthURL:  "https://discord.com/oauth2/authorize",
					TokenURL: "https://discord.com/api/oauth2/token",
				},
				RedirectURL: fmt.Sprintf("%s/api/oauth2/discord/redirect", os.Getenv("PUBLIC_URL")),
			},
			repository.ProviderTwitch: {
				ClientID:     os.Getenv("TWITCH_CLIENT_ID"),
				ClientSecret: os.Getenv("TWITCH_CLIENT_SECRET"),
				Scopes:       []string{},
				Endpoint: oauth2.Endpoint{
					AuthURL:  "https://id.twitch.tv/oauth2/authorize",
					TokenURL: "https://id.twitch.tv/oauth2/token",
				},
				RedirectURL: fmt.Sprintf("%s/api/oauth2/twitch/redirect", os.Getenv("PUBLIC_URL")),
			},
		},
		clientConfig: map[repository.Provider]*clientcredentials.Config{
			repository.ProviderTwitch: {
				ClientID:     os.Getenv("TWITCH_CLIENT_ID"),
				ClientSecret: os.Getenv("TWITCH_CLIENT_SECRET"),
				TokenURL:     "https://id.twitch.tv/oauth2/token",
			},
		},

		stateToVerifyer:            make(map[string]Verifier),
		userService:                NewUserService(),
		clientCredentialRepository: repository.NewClientCredentialsRepository(),
	}
}

func (e *OauthService) GetNewVerifier(user *repository.User) (string, string) {
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

func (e *OauthService) GetRedirectUrl(user *repository.User, provider repository.Provider) string {
	state, verifier := e.GetNewVerifier(user)
	return e.config[provider].AuthCodeURL(state, oauth2.SetAuthURLParam("code_challenge", oauth2.S256ChallengeFromVerifier(verifier)))
}

func (e *OauthService) Verify(state string, code string, provider repository.Provider) (*repository.User, error) {
	switch provider {
	case repository.ProviderDiscord:
		return e.VerifyDiscord(state, code)
	case repository.ProviderTwitch:
		return e.VerifyTwitch(state, code)
	default:
		return nil, fmt.Errorf("not implemented")
	}
}

func (e *OauthService) VerifyDiscord(state string, code string) (*repository.User, error) {
	verifier, ok := e.stateToVerifyer[state]
	if !ok {
		return nil, fmt.Errorf("state is unknown")
	}
	token, err := e.config[repository.ProviderDiscord].Exchange(context.Background(), code, oauth2.SetAuthURLParam("code_verifier", verifier.Verifier))
	if err != nil {
		return nil, err
	}
	client := e.config[repository.ProviderDiscord].Client(context.Background(), token)
	response, err := client.Get("https://discord.com/api/users/@me")
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	discordUser := &DiscordUserResponse{}
	json.NewDecoder(response.Body).Decode(discordUser)

	user := &repository.User{}
	if verifier.User != nil {
		user = verifier.User
	} else {
		user, err = e.userService.GetUserByDiscordId(discordUser.Id)
		if err != nil {
			user = &repository.User{
				Permissions:   []repository.Permission{},
				DisplayName:   discordUser.Username,
				OauthAccounts: []*repository.Oauth{},
			}
		}
	}
	oauthAccounts := []*repository.Oauth{}
	for _, oauthAccount := range user.OauthAccounts {
		if oauthAccount.Provider != repository.ProviderDiscord {
			oauthAccounts = append(oauthAccounts, oauthAccount)
		}
	}

	user.OauthAccounts = append(oauthAccounts, &repository.Oauth{
		Provider:     repository.ProviderDiscord,
		AccountId:    discordUser.Id,
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		Expiry:       token.Expiry,
		Name:         discordUser.Username,
	})

	user, err = e.userService.SaveUser(user)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (e *OauthService) VerifyTwitch(state string, code string) (*repository.User, error) {
	verifier, ok := e.stateToVerifyer[state]
	if !ok {
		return nil, fmt.Errorf("state is unknown")
	}
	token, err := e.config[repository.ProviderTwitch].Exchange(context.Background(), code, oauth2.SetAuthURLParam("code_verifier", verifier.Verifier))
	if err != nil {
		return nil, err
	}
	response, err := e.config[repository.ProviderTwitch].Client(context.Background(), token).Get("https://id.twitch.tv/oauth2/userinfo")
	if err != nil {
		return nil, err
	}
	twitchUser := &TwitchUserResponse{}
	json.NewDecoder(response.Body).Decode(twitchUser)
	response.Body.Close()
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
			"Client-Id":     {os.Getenv("TWITCH_CLIENT_ID")},
		},
	}
	client := &http.Client{}
	response, err = client.Do(req)
	if err != nil {
		return nil, err
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
				Permissions:   []repository.Permission{},
				DisplayName:   twitchExtendedUser.Data[0].DisplayName,
				OauthAccounts: []*repository.Oauth{},
			}
		}
	}
	oauthAccounts := []*repository.Oauth{}
	for _, oauthAccount := range user.OauthAccounts {
		if oauthAccount.Provider != repository.ProviderDiscord {
			oauthAccounts = append(oauthAccounts, oauthAccount)
		}
	}
	user.OauthAccounts = append(oauthAccounts, &repository.Oauth{
		Provider:     repository.ProviderTwitch,
		AccountId:    twitchId,
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		Expiry:       token.Expiry,
		Name:         twitchExtendedUser.Data[0].DisplayName,
	})

	return e.userService.SaveUser(user)
}

func (e *OauthService) GetApplicationToken(provider repository.Provider) (*string, error) {
	credentials, err := e.clientCredentialRepository.GetClientCredentialsByName(provider)
	if err != nil || credentials.Expiry.Before(time.Now()) {
		config, ok := e.clientConfig[provider]
		if !ok {
			return nil, fmt.Errorf("provider not found")
		}
		token, err := config.Token(context.Background())
		if err != nil {
			return nil, err
		}
		if credentials == nil {
			credentials = &repository.ClientCredentials{
				Name:        provider,
				AccessToken: token.AccessToken,
				Expiry:      token.Expiry,
			}
		} else {
			credentials.AccessToken = token.AccessToken
			credentials.Expiry = token.Expiry
		}
		e.clientCredentialRepository.DB.Save(credentials)
	}
	return &credentials.AccessToken, nil
}
