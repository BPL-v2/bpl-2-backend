package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
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

func NewPoEClient(maxRequestsPerSecond float64, raiseForStatus bool, timeOutSeconds int) *PoEClient {
	baseURL := &url.URL{Scheme: "https", Host: "api.pathofexile.com"}
	return &PoEClient{
		Client:         NewAsyncHttpClient(baseURL, os.Getenv("POE_CLIENT_AGENT"), maxRequestsPerSecond),
		TimeOutSeconds: timeOutSeconds,
	}
}

var poeRequestCounter = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "poe_request_total",
	Help: "The total number of requests by endpoint to the PoE API",
}, []string{"endpoint"})

var responseCounter = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "poe_response_total",
	Help: "The total number of responses by status code from the PoE API",
}, []string{"status_code"})

var requestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Name: "poe_request_duration_seconds",
	Help: "Duration of requests to the PoE API",
}, []string{"endpoint"})

func sendRequest[T any](client *PoEClient, requestKey string, args RequestArgs) (*T, *ClientError) {
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
	response, err := client.Client.SendRequest(ctx, requestKey, args)
	if err != nil {
		return nil, &ClientError{
			StatusCode:  0,
			Error:       "bpl2_client_request_error",
			Description: err.Error(),
		}
	}
	responseCounter.WithLabelValues(fmt.Sprintf("%d", response.StatusCode)).Inc()
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
		log.Print(string(respBody))
		errorBody := &ErrorResponse{}
		err = json.Unmarshal(respBody, errorBody)
		if err != nil {
			return nil, &ClientError{
				StatusCode:      response.StatusCode,
				Error:           "bpl2_client_response_error_body_parse_error",
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
		fmt.Println(string(respBody))
		return nil, &ClientError{
			StatusCode:      response.StatusCode,
			Error:           "bpl2_client_response_body_parse_error",
			Description:     err.Error(),
			ResponseHeaders: response.Header,
		}
	}
	return result, nil
}

func (c *PoEClient) ListLeagues(token string, realm string, leagueType string, limit int, offset int) (*ListLeaguesResponse, *ClientError) {
	timer := prometheus.NewTimer(requestDuration.WithLabelValues("ListLeagues"))
	defer timer.ObserveDuration()
	poeRequestCounter.WithLabelValues("ListLeagues").Inc()
	return sendRequest[ListLeaguesResponse](c,
		"ListLeagues",
		RequestArgs{
			Path:   "league",
			Token:  token,
			Method: "GET",
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
	timer := prometheus.NewTimer(requestDuration.WithLabelValues("GetLeague"))
	defer timer.ObserveDuration()
	poeRequestCounter.WithLabelValues("GetLeague").Inc()
	return sendRequest[GetLeagueResponse](c,
		"GetLeague",
		RequestArgs{
			Path:       "league/%s",
			PathParams: []string{league},
			Token:      token,
			Method:     "GET",
			QueryParams: map[string]string{
				"realm": realm,
			},
		},
	)
}

func (c *PoEClient) GetLeagueLadder(token string, league string, realm string, sort string, limit int, offset int) (*GetLeagueLadderResponse, *ClientError) {
	timer := prometheus.NewTimer(requestDuration.WithLabelValues("GetLeagueLadder"))
	defer timer.ObserveDuration()
	poeRequestCounter.WithLabelValues("GetLeagueLadder").Inc()
	return sendRequest[GetLeagueLadderResponse](c,
		"GetLeagueLadder",
		RequestArgs{
			Path:       "league/%s/ladder",
			PathParams: []string{league},
			Token:      token,
			Method:     "GET",
			QueryParams: map[string]string{
				"realm":  realm,
				"sort":   sort,
				"limit":  fmt.Sprintf("%d", limit),
				"offset": fmt.Sprintf("%d", offset),
			},
		},
	)
}

func (c *PoEClient) GetFullLadder(token string, league string) (*GetLeagueLadderResponse, *ClientError) {
	response, err := c.GetLeagueLadder(token, league, "pc", "xp", 500, 0)
	if err != nil {
		return nil, err
	}
	Total := response.Ladder.Total
	wg := sync.WaitGroup{}
	for i := 1; i < int(math.Ceil(float64(Total)/500)); i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			newResp, err := c.GetLeagueLadder(token, league, "pc", "xp", 500, i*500)
			if err != nil {
				return
			}
			response.Ladder.Entries = append(response.Ladder.Entries, newResp.Ladder.Entries...)
		}(i)
	}
	wg.Wait()
	return response, nil
}

func (c *PoEClient) GetPoE2Ladder(league string) (*GetLeagueLadderResponse, *ClientError) {
	timer := prometheus.NewTimer(requestDuration.WithLabelValues("GetPoE2Ladder"))
	defer timer.ObserveDuration()
	poeRequestCounter.WithLabelValues("GetPoE2Ladder").Inc()
	resp, err := sendRequest[GetPoE2LadderResponse](c,
		"GetPoE2Ladder",
		RequestArgs{
			Path:          "https://pathofexile2.com/internal-api/content/game-ladder/id/%s",
			PathParams:    []string{league},
			Method:        "GET",
			IgnoreBaseURL: true,
		},
	)
	if err != nil {
		return nil, err
	}
	return resp.Context, nil
}

func (c *PoEClient) GetLeagueEventLadder(token string, league string, realm string, limit int, offset int) (*GetLeagueEventLadderResponse, *ClientError) {
	timer := prometheus.NewTimer(requestDuration.WithLabelValues("GetLeagueEventLadder"))
	defer timer.ObserveDuration()
	poeRequestCounter.WithLabelValues("GetLeagueEventLadder").Inc()
	return sendRequest[GetLeagueEventLadderResponse](c,
		"GetLeagueEventLadder",
		RequestArgs{
			Path:       "league/%s/event-ladder",
			PathParams: []string{league},
			Token:      token,
			Method:     "GET",
			QueryParams: map[string]string{
				"realm":  realm,
				"limit":  fmt.Sprintf("%d", limit),
				"offset": fmt.Sprintf("%d", offset),
			},
		},
	)
}

func (c *PoEClient) GetPvPMatches(token string, realm string, matchType string) (*GetPvPMatchesResponse, *ClientError) {
	timer := prometheus.NewTimer(requestDuration.WithLabelValues("GetPvPMatches"))
	defer timer.ObserveDuration()
	poeRequestCounter.WithLabelValues("GetPvPMatches").Inc()
	return sendRequest[GetPvPMatchesResponse](c,
		"GetPvPMatches",
		RequestArgs{
			Path:   "pvp-match",
			Token:  token,
			Method: "GET",
			QueryParams: map[string]string{
				"realm": realm,
				"type":  matchType,
			},
		},
	)
}

func (c *PoEClient) GetPvPMatch(token string, match string, realm string) (*GetPvPMatchResponse, *ClientError) {
	timer := prometheus.NewTimer(requestDuration.WithLabelValues("GetPvPMatch"))
	defer timer.ObserveDuration()
	poeRequestCounter.WithLabelValues("GetPvPMatch").Inc()
	return sendRequest[GetPvPMatchResponse](c,
		"GetPvPMatch",
		RequestArgs{
			Path:       "pvp-match/%s",
			PathParams: []string{match},
			Token:      token,
			Method:     "GET",
			QueryParams: map[string]string{
				"realm": realm,
			},
		},
	)
}

func (c *PoEClient) GetPvPMatchLadder(token string, match string, realm string, limit int, offset int) (*GetPvPMatchLadderResponse, *ClientError) {
	timer := prometheus.NewTimer(requestDuration.WithLabelValues("GetPvPMatchLadder"))
	defer timer.ObserveDuration()
	poeRequestCounter.WithLabelValues("GetPvPMatchLadder").Inc()
	return sendRequest[GetPvPMatchLadderResponse](c,
		"GetPvPMatchLadder",
		RequestArgs{
			Path:       "pvp-match/%s/ladder",
			PathParams: []string{match},
			Token:      token,
			Method:     "GET",
			QueryParams: map[string]string{
				"realm":  realm,
				"limit":  fmt.Sprintf("%d", limit),
				"offset": fmt.Sprintf("%d", offset),
			},
		},
	)
}

func (c *PoEClient) GetAccountProfile(token string) (*GetAccountProfileResponse, *ClientError) {
	timer := prometheus.NewTimer(requestDuration.WithLabelValues("GetAccountProfile"))
	defer timer.ObserveDuration()
	poeRequestCounter.WithLabelValues("GetAccountProfile").Inc()
	return sendRequest[GetAccountProfileResponse](c,
		"GetAccountProfile",
		RequestArgs{
			Path:   "profile",
			Token:  token,
			Method: "GET",
		},
	)
}

func (c *PoEClient) GetAccountLeagues(token string) (*ListLeaguesResponse, *ClientError) {
	timer := prometheus.NewTimer(requestDuration.WithLabelValues("GetAccountLeagues"))
	defer timer.ObserveDuration()
	poeRequestCounter.WithLabelValues("GetAccountLeagues").Inc()
	return sendRequest[ListLeaguesResponse](c,
		"GetAccountLeagues",
		RequestArgs{
			Path:   "account/leagues",
			Token:  token,
			Method: "GET",
		},
	)
}

func (c *PoEClient) ListCharacters(token string, realm *Realm) (*ListCharactersResponse, *ClientError) {
	timer := prometheus.NewTimer(requestDuration.WithLabelValues("ListCharacters"))
	defer timer.ObserveDuration()
	poeRequestCounter.WithLabelValues("ListCharacters").Inc()
	endpoint := "character"
	if realm != nil {
		endpoint += fmt.Sprintf("/%s", *realm)
	}
	return sendRequest[ListCharactersResponse](c,
		"ListCharacters",
		RequestArgs{
			Path:   endpoint,
			Token:  token,
			Method: "GET",
		},
	)
}

func (c *PoEClient) GetCharacter(token string, character string, realm *Realm) (*GetCharacterResponse, *ClientError) {
	timer := prometheus.NewTimer(requestDuration.WithLabelValues("GetCharacter"))
	defer timer.ObserveDuration()
	endpoint := "character"
	if realm != nil {
		endpoint += fmt.Sprintf("/%s", *realm)
	}
	poeRequestCounter.WithLabelValues("GetCharacter").Inc()
	return sendRequest[GetCharacterResponse](c,
		"GetCharacter",
		RequestArgs{
			Path:       endpoint + "/%s",
			PathParams: []string{character},
			Token:      token,
			Method:     "GET",
		},
	)
}

func (c *PoEClient) ListAccountStashes(token string, league string) (*ListAccountStashesResponse, *ClientError) {
	timer := prometheus.NewTimer(requestDuration.WithLabelValues("ListAccountStashes"))
	defer timer.ObserveDuration()
	poeRequestCounter.WithLabelValues("ListAccountStashes").Inc()
	return sendRequest[ListAccountStashesResponse](c,
		"ListAccountStashes",
		RequestArgs{
			Path:       "stash/%s",
			PathParams: []string{league},
			Token:      token,
			Method:     "GET",
		},
	)
}

func (c *PoEClient) GetAccountStash(token string, league string, stashId string, substashId *string) (*GetAccountStashResponse, *ClientError) {
	timer := prometheus.NewTimer(requestDuration.WithLabelValues("GetAccountStash"))
	defer timer.ObserveDuration()
	poeRequestCounter.WithLabelValues("GetAccountStash").Inc()
	endpoint := fmt.Sprintf("stash/%s/%s", league, stashId)
	if substashId != nil {
		endpoint += fmt.Sprintf("/%s", *substashId)
	}
	return sendRequest[GetAccountStashResponse](c,
		"GetAccountStash",
		RequestArgs{
			Path:   endpoint,
			Token:  token,
			Method: "GET",
		},
	)
}

func (c *PoEClient) ListItemFilters(token string) (*ListItemFiltersResponse, *ClientError) {
	timer := prometheus.NewTimer(requestDuration.WithLabelValues("ListItemFilters"))
	defer timer.ObserveDuration()
	poeRequestCounter.WithLabelValues("ListItemFilters").Inc()
	return sendRequest[ListItemFiltersResponse](c,
		"ListItemFilters",
		RequestArgs{
			Path:   "item-filter",
			Token:  token,
			Method: "GET",
		},
	)
}

func (c *PoEClient) GetItemFilter(token string, filterId string) (*GetItemFilterResponse, *ClientError) {
	timer := prometheus.NewTimer(requestDuration.WithLabelValues("GetItemFilter"))
	defer timer.ObserveDuration()
	poeRequestCounter.WithLabelValues("GetItemFilter").Inc()
	return sendRequest[GetItemFilterResponse](c,
		"GetItemFilter",
		RequestArgs{
			Path:       "item-filter/%s",
			PathParams: []string{filterId},
			Token:      token,
			Method:     "GET",
		},
	)
}

func (c *PoEClient) CreateItemFilter(token string, body CreateFilterBody, validate string) (*CreateItemFilterResponse, *ClientError) {
	timer := prometheus.NewTimer(requestDuration.WithLabelValues("CreateItemFilter"))
	defer timer.ObserveDuration()
	poeRequestCounter.WithLabelValues("CreateItemFilter").Inc()
	return sendRequest[CreateItemFilterResponse](c,
		"CreateItemFilter",
		RequestArgs{
			Path:   "item-filter",
			Token:  token,
			Method: "POST",
			QueryParams: map[string]string{
				"validate": validate,
			},
			BodyRaw: body,
		},
	)
}

func (c *PoEClient) UpdateItemFilter(token string, filterId string, body UpdateFilterBody, validate string) (*UpdateItemFilterResponse, *ClientError) {
	timer := prometheus.NewTimer(requestDuration.WithLabelValues("UpdateItemFilter"))
	defer timer.ObserveDuration()
	poeRequestCounter.WithLabelValues("UpdateItemFilter").Inc()
	return sendRequest[UpdateItemFilterResponse](c,
		"UpdateItemFilter",
		RequestArgs{
			Path:       "item-filter/%s",
			PathParams: []string{filterId},
			Token:      token,
			Method:     "POST",
			QueryParams: map[string]string{
				"validate": validate,
			},
			BodyRaw: body,
		},
	)
}

func (c *PoEClient) GetLeagueAccount(token string, league string) (*GetLeagueAccountResponse, *ClientError) {
	timer := prometheus.NewTimer(requestDuration.WithLabelValues("GetLeagueAccount"))
	defer timer.ObserveDuration()
	poeRequestCounter.WithLabelValues("GetLeagueAccount").Inc()
	return sendRequest[GetLeagueAccountResponse](c,
		"GetLeagueAccount",
		RequestArgs{
			Path:       "league-account/%s",
			PathParams: []string{league},
			Token:      token,
			Method:     "GET",
		},
	)
}

func (c *PoEClient) ListGuildStashes(token string, league string) (*ListGuildStashesResponse, *ClientError) {
	timer := prometheus.NewTimer(requestDuration.WithLabelValues("ListGuildStashes"))
	defer timer.ObserveDuration()
	poeRequestCounter.WithLabelValues("ListGuildStashes").Inc()
	return sendRequest[ListGuildStashesResponse](c,
		"ListGuildStashes",
		RequestArgs{
			Path:       "guild/stash/%s",
			PathParams: []string{league},
			Token:      token,
			Method:     "GET",
		},
	)
}

func (c *PoEClient) GetGuildStash(token string, league string, stashId string, parentId *string) (*GetGuildStashResponse, *ClientError) {
	timer := prometheus.NewTimer(requestDuration.WithLabelValues("GetGuildStash"))
	defer timer.ObserveDuration()
	poeRequestCounter.WithLabelValues("GetGuildStash").Inc()
	endpoint := "guild/stash/%s/%s"
	pathParams := []string{league}
	if parentId != nil {
		endpoint += "/%s"
		pathParams = append(pathParams, *parentId)
	}
	pathParams = append(pathParams, stashId)
	return sendRequest[GetGuildStashResponse](c,
		"GetGuildStash",
		RequestArgs{
			Path:       endpoint,
			PathParams: pathParams,
			Token:      token,
			Method:     "GET",
		},
	)
}

func (c *PoEClient) GetPublicStashes(token string, realm string, id string) (*GetPublicStashTabsResponse, *ClientError) {
	timer := prometheus.NewTimer(requestDuration.WithLabelValues("GetPublicStashes"))
	defer timer.ObserveDuration()
	poeRequestCounter.WithLabelValues("GetPublicStashes").Inc()
	url := "public-stash-tabs"
	params := map[string]string{}
	if realm != "pc" {
		url += "/" + realm
	}
	if id != "" {
		params["id"] = id
	}
	return sendRequest[GetPublicStashTabsResponse](c,
		"GetPublicStashes",
		RequestArgs{
			Path:        url,
			Token:       token,
			Method:      "GET",
			QueryParams: params,
		},
	)
}

func (c *PoEClient) GetClientCredentials(clientId string, clientSecret string) (*ClientCredentialsGrantResponse, *ClientError) {
	timer := prometheus.NewTimer(requestDuration.WithLabelValues("GetClientCredentials"))
	defer timer.ObserveDuration()
	poeRequestCounter.WithLabelValues("GetClientCredentials").Inc()
	form := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {clientId},
		"client_secret": {clientSecret},
		"scope":         {"service:psapi"},
	}
	return sendRequest[ClientCredentialsGrantResponse](c,
		"GetClientCredentials",
		RequestArgs{
			Path:          "https://www.pathofexile.com/oauth/token",
			Token:         "",
			Method:        "POST",
			Body:          strings.NewReader(form.Encode()),
			IgnoreBaseURL: true,
			Headers: map[string]string{
				"Content-Type": "application/x-www-form-urlencoded",
			},
		},
	)
}

func (c *PoEClient) GetAccessToken(clientId string, clientSecret string, code string, code_verifier string, scopes []string, redirect_uri string) (*AccessTokenGrantResponse, *ClientError) {
	timer := prometheus.NewTimer(requestDuration.WithLabelValues("GetAccessToken"))
	defer timer.ObserveDuration()
	poeRequestCounter.WithLabelValues("GetAccessToken").Inc()
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {clientId},
		"client_secret": {clientSecret},
		"redirect_uri":  {redirect_uri},
		"code":          {code},
		"scope":         {strings.Join(scopes, " ")},
		"code_verifier": {code_verifier},
	}
	return sendRequest[AccessTokenGrantResponse](c,
		"GetAccessToken",
		RequestArgs{
			Path:   "https://www.pathofexile.com/oauth/token",
			Method: "POST",
			Body:   strings.NewReader(form.Encode()),
			Headers: map[string]string{
				"Content-Type": "application/x-www-form-urlencoded",
			},
			IgnoreBaseURL: true,
		},
	)
}
func (c *PoEClient) RefreshAccessToken(clientId string, clientSecret string, refreshToken string) (*AccessTokenGrantResponse, *ClientError) {
	timer := prometheus.NewTimer(requestDuration.WithLabelValues("RefreshAccessToken"))
	defer timer.ObserveDuration()
	poeRequestCounter.WithLabelValues("RefreshAccessToken").Inc()
	form := url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {clientId},
		"client_secret": {clientSecret},
		"refresh_token": {refreshToken},
	}
	return sendRequest[AccessTokenGrantResponse](c,
		"RefreshAccessToken",
		RequestArgs{
			Path:   "https://www.pathofexile.com/oauth/token",
			Method: "POST",
			Body:   strings.NewReader(form.Encode()),
			Headers: map[string]string{
				"Content-Type": "application/x-www-form-urlencoded",
			},
			IgnoreBaseURL: true,
		},
	)
}
