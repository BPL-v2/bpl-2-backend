package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type PoEClient struct {
	Client         *AsyncHttpClient
	TimeOutSeconds int
}

type ClientError struct {
	StatusCode      int
	Error           any
	Description     string
	ResponseHeaders http.Header
}

type ErrorResponse struct {
	Error            any    `json:"error"`
	ErrorDescription string `json:"error_description"`
}

func NewPoEClient(userAgent string, maxRequestsPerSecond float64, raiseForStatus bool, timeOutSeconds int) *PoEClient {
	baseURL := &url.URL{Scheme: "https", Host: "www.pathofexile.com", Path: "/api"}
	return &PoEClient{
		Client:         NewAsyncHttpClient(baseURL, userAgent, maxRequestsPerSecond),
		TimeOutSeconds: timeOutSeconds,
	}
}

func sendRequest[T any](client *PoEClient, args RequestArgs) (*T, *ClientError) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(client.TimeOutSeconds)*time.Second)
	defer cancel()

	if args.Body == nil && args.BodyRaw != nil {
		bodyString, err := json.Marshal(args.BodyRaw)
		if err != nil {
			return nil, &ClientError{
				StatusCode:  0,
				Error:       "bpl2_client_request_body_error",
				Description: err.Error(),
			}
		}
		args.Body = strings.NewReader(string(bodyString))
	}
	response, err := client.Client.SendRequest(ctx, args)
	if err != nil {
		return nil, &ClientError{
			StatusCode:  0,
			Error:       "bpl2_client_request_error",
			Description: err.Error(),
		}
	}
	defer response.Body.Close()
	respBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, &ClientError{
			StatusCode:      0,
			Error:           "bpl2_client_response_body_read_error",
			Description:     err.Error(),
			ResponseHeaders: response.Header,
		}
	}

	if response.StatusCode >= 400 {
		fmt.Println(string(respBody))
		errorBody := &ErrorResponse{}
		err = json.Unmarshal(respBody, errorBody)
		if err != nil {
			return nil, &ClientError{
				StatusCode:      response.StatusCode,
				Error:           "bpl2_client_response_body_read_error",
				Description:     err.Error(),
				ResponseHeaders: response.Header,
			}
		}
		return nil, &ClientError{
			StatusCode:      response.StatusCode,
			Error:           errorBody.Error,
			Description:     errorBody.ErrorDescription,
			ResponseHeaders: response.Header,
		}
	}

	result := new(T)
	err = json.Unmarshal(respBody, result)
	if err != nil {
		return nil, &ClientError{
			StatusCode:      response.StatusCode,
			Error:           "bpl2_client_response_body_read_error",
			Description:     err.Error(),
			ResponseHeaders: response.Header,
		}
	}
	return result, nil
}

func (c *PoEClient) ListLeagues(token string, realm string, leagueType string, limit int, offset int) (*ListLeaguesResponse, *ClientError) {
	return sendRequest[ListLeaguesResponse](c, RequestArgs{
		Endpoint: "league",
		Token:    token,
		Method:   "GET",
		QueryParams: map[string]string{
			"realm":  realm,
			"type":   leagueType,
			"limit":  fmt.Sprintf("%d", limit),
			"offset": fmt.Sprintf("%d", offset),
		},
	},
	)

}

func (c *PoEClient) GetLeague(token string, league string, realm string) (*GetLeagueResponse, *ClientError) {
	return sendRequest[GetLeagueResponse](c, RequestArgs{
		Endpoint: fmt.Sprintf("league/%s", league),
		Token:    token,
		Method:   "GET",
		QueryParams: map[string]string{
			"realm": realm,
		},
	},
	)
}

func (c *PoEClient) GetLeagueLadder(token string, league string, realm string, sort string, limit int, offset int) (*GetLeagueLadderResponse, *ClientError) {
	return sendRequest[GetLeagueLadderResponse](c, RequestArgs{
		Endpoint: fmt.Sprintf("league/%s/ladder", league),
		Token:    token,
		Method:   "GET",
		QueryParams: map[string]string{
			"realm":  realm,
			"sort":   sort,
			"limit":  fmt.Sprintf("%d", limit),
			"offset": fmt.Sprintf("%d", offset),
		},
	},
	)
}

func (c *PoEClient) GetLeagueEventLadder(token string, league string, realm string, limit int, offset int) (*GetLeagueEventLadderResponse, *ClientError) {
	return sendRequest[GetLeagueEventLadderResponse](c, RequestArgs{
		Endpoint: fmt.Sprintf("league/%s/event-ladder", league),
		Token:    token,
		Method:   "GET",
		QueryParams: map[string]string{
			"realm":  realm,
			"limit":  fmt.Sprintf("%d", limit),
			"offset": fmt.Sprintf("%d", offset),
		},
	},
	)
}

func (c *PoEClient) GetPvPMatches(token string, realm string, matchType string) (*GetPvPMatchesResponse, *ClientError) {
	return sendRequest[GetPvPMatchesResponse](c, RequestArgs{
		Endpoint: "pvp-match",
		Token:    token,
		Method:   "GET",
		QueryParams: map[string]string{
			"realm": realm,
			"type":  matchType,
		},
	},
	)
}

func (c *PoEClient) GetPvPMatch(token string, match string, realm string) (*GetPvPMatchResponse, *ClientError) {
	return sendRequest[GetPvPMatchResponse](c, RequestArgs{
		Endpoint: fmt.Sprintf("pvp-match/%s", match),
		Token:    token,
		Method:   "GET",
		QueryParams: map[string]string{
			"realm": realm,
		},
	},
	)
}

func (c *PoEClient) GetPvPMatchLadder(token string, match string, realm string, limit int, offset int) (*GetPvPMatchLadderResponse, *ClientError) {
	return sendRequest[GetPvPMatchLadderResponse](c, RequestArgs{
		Endpoint: fmt.Sprintf("pvp-match/%s/ladder", match),
		Token:    token,
		Method:   "GET",
		QueryParams: map[string]string{
			"realm":  realm,
			"limit":  fmt.Sprintf("%d", limit),
			"offset": fmt.Sprintf("%d", offset),
		},
	},
	)
}

func (c *PoEClient) GetAccountProfile(token string) (*GetAccountProfileResponse, *ClientError) {
	return sendRequest[GetAccountProfileResponse](c, RequestArgs{
		Endpoint: "profile",
		Token:    token,
		Method:   "GET",
	},
	)
}

func (c *PoEClient) GetAccountLeagues(token string) (*ListLeaguesResponse, *ClientError) {
	return sendRequest[ListLeaguesResponse](c,
		RequestArgs{
			Endpoint: "account/leagues",
			Token:    token,
			Method:   "GET",
		},
	)
}

func (c *PoEClient) ListCharacters(token string) (*ListCharactersResponse, *ClientError) {
	return sendRequest[ListCharactersResponse](c, RequestArgs{
		Endpoint: "character",
		Token:    token,
		Method:   "GET",
	},
	)
}

func (c *PoEClient) GetCharacter(token string, character string) (*GetCharacterResponse, *ClientError) {
	return sendRequest[GetCharacterResponse](c, RequestArgs{
		Endpoint: fmt.Sprintf("character/%s", character),
		Token:    token,
		Method:   "GET",
	},
	)
}

func (c *PoEClient) ListAccountStashes(token string, league string) (*ListAccountStashesResponse, *ClientError) {
	return sendRequest[ListAccountStashesResponse](c, RequestArgs{
		Endpoint: fmt.Sprintf("stash/%s", league),
		Token:    token,
		Method:   "GET",
	},
	)
}

func (c *PoEClient) GetAccountStash(token string, league string, stashID string, substashID *string) (*GetAccountStashResponse, *ClientError) {
	endpoint := fmt.Sprintf("stash/%s/%s", league, stashID)
	if substashID != nil {
		endpoint += fmt.Sprintf("/%s", *substashID)
	}
	return sendRequest[GetAccountStashResponse](c, RequestArgs{
		Endpoint: endpoint,
		Token:    token,
		Method:   "GET",
	},
	)
}

func (c *PoEClient) ListItemFilters(token string) (*ListItemFiltersResponse, *ClientError) {
	return sendRequest[ListItemFiltersResponse](c, RequestArgs{
		Endpoint: "item-filter",
		Token:    token,
		Method:   "GET",
	},
	)
}

func (c *PoEClient) GetItemFilter(token string, filterID string) (*GetItemFilterResponse, *ClientError) {
	return sendRequest[GetItemFilterResponse](c, RequestArgs{
		Endpoint: fmt.Sprintf("item-filter/%s", filterID),
		Token:    token,
		Method:   "GET",
	},
	)
}

func (c *PoEClient) CreateItemFilter(token string, body CreateFilterBody, validate string) (*CreateItemFilterResponse, *ClientError) {
	return sendRequest[CreateItemFilterResponse](c, RequestArgs{
		Endpoint: "item-filter",
		Token:    token,
		Method:   "POST",
		QueryParams: map[string]string{
			"validate": validate,
		},
		BodyRaw: body,
	},
	)
}

func (c *PoEClient) UpdateItemFilter(token string, filterID string, body UpdateFilterBody, validate string) (*UpdateItemFilterResponse, *ClientError) {
	return sendRequest[UpdateItemFilterResponse](c, RequestArgs{
		Endpoint: fmt.Sprintf("item-filter/%s", filterID),
		Token:    token,
		Method:   "POST",
		QueryParams: map[string]string{
			"validate": validate,
		},
		BodyRaw: body,
	},
	)
}

func (c *PoEClient) GetLeagueAccount(token string, league string) (*GetLeagueAccountResponse, *ClientError) {
	return sendRequest[GetLeagueAccountResponse](c, RequestArgs{
		Endpoint: fmt.Sprintf("league-account/%s", league),
		Token:    token,
		Method:   "GET",
	},
	)
}

func (c *PoEClient) ListGuildStashes(token string, league string) (*ListGuildStashesResponse, *ClientError) {
	return sendRequest[ListGuildStashesResponse](c, RequestArgs{
		Endpoint: fmt.Sprintf("guild/stash/%s", league),
		Token:    token,
		Method:   "GET",
	},
	)
}

func (c *PoEClient) GetGuildStash(token string, league string, stashID string, substashID *string) (*GetGuildStashResponse, *ClientError) {
	endpoint := fmt.Sprintf("guild/stash/%s/%s", league, stashID)
	if substashID != nil {
		endpoint += fmt.Sprintf("/%s", *substashID)
	}
	return sendRequest[GetGuildStashResponse](c, RequestArgs{
		Endpoint: endpoint,
		Token:    token,
		Method:   "GET",
	},
	)
}

func (c *PoEClient) GetPublicStashes(token string, realm string, id string) (*GetPublicStashTabsResponse, *ClientError) {
	url := "public-stash-tabs"
	params := map[string]string{}
	if realm != "pc" {
		url += "/" + realm
	}
	if id != "" {
		params["id"] = id
	}
	return sendRequest[GetPublicStashTabsResponse](c, RequestArgs{
		Endpoint:    url,
		Token:       token,
		Method:      "GET",
		QueryParams: params,
	},
	)
}

func (c *PoEClient) GetClientCredentials(clientID string, clientSecret string, scope string) (*ClientCredentialsGrantResponse, *ClientError) {
	form := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"scope":         {scope},
	}
	return sendRequest[ClientCredentialsGrantResponse](c, RequestArgs{
		Endpoint:      "https://www.pathofexile.com/oauth/token",
		Token:         "",
		Method:        "POST",
		Body:          strings.NewReader(form.Encode()),
		IgnoreBaseURL: true,
	},
	)
}

func (c *PoEClient) RefreshAccessToken(clientID string, clientSecret string, refreshToken string) (*RefreshTokenGrantResponse, *ClientError) {
	form := url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"refresh_token": {refreshToken},
	}
	return sendRequest[RefreshTokenGrantResponse](c, RequestArgs{
		Endpoint: "https://www.pathofexile.com/oauth/token",
		Method:   "POST",
		Body:     strings.NewReader(form.Encode()),
		Headers: map[string]string{
			"Content-Type": "application/x-www-form-urlencoded",
		},
		IgnoreBaseURL: true,
	},
	)

}
