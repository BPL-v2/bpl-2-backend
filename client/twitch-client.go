package client

import (
	"bpl/utils"
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"
)

type TwitchClient struct {
	Token        string
	clientId     string
	clientSecret string
	client       *http.Client
	baseURL      string
	rateLimiter  *time.Ticker
	mu           sync.Mutex
}

type TwitchStream struct {
	Id           string   `json:"id"`
	UserId       string   `json:"user_id"`
	UserLogin    string   `json:"user_login"`
	UserName     string   `json:"user_name"`
	GameId       string   `json:"game_id"`
	GameName     string   `json:"game_name"`
	Type         string   `json:"type"`
	Title        string   `json:"title"`
	Tags         []string `json:"tags"`
	ViewerCount  int      `json:"viewer_count"`
	StartedAt    string   `json:"started_at"`
	Language     string   `json:"language"`
	ThumbnailURL string   `json:"thumbnail_url"`
	TagIds       []string `json:"tag_ids"`
	IsMature     bool     `json:"is_mature"`

	BackendUserId int `json:"backend_user_id"`
}

type StreamResponse struct {
	Data       []*TwitchStream `json:"data"`
	Pagination struct {
		Cursor string `json:"cursor"`
	} `json:"pagination"`
}

func NewTwitchClient(token string) *TwitchClient {
	return &TwitchClient{
		clientId:     os.Getenv("TWITCH_CLIENT_ID"),
		clientSecret: os.Getenv("TWITCH_CLIENT_SECRET"),
		Token:        token,
		client:       &http.Client{},
		baseURL:      "https://api.twitch.tv/helix",
		rateLimiter:  time.NewTicker((100 * time.Millisecond)),
	}
}

func (t *TwitchClient) GetAllStreams(userIds []string) ([]*TwitchStream, error) {
	streamChannel := make(chan []*TwitchStream)
	var wg sync.WaitGroup
	for userBatch := range utils.BatchIterator(userIds, 100) {
		func(ids []string) {
			wg.Add(1)
			go func() {
				defer wg.Done()
				streamChannel <- t.GetStreams(ids, nil, 10)
			}()
		}(userBatch)

	}

	go func() {
		wg.Wait()
		close(streamChannel)
	}()

	allStreams := make([]*TwitchStream, 0)
	for streams := range streamChannel {
		allStreams = append(allStreams, streams...)
	}
	return allStreams, nil
}

func (t *TwitchClient) GetStreams(userIds []string, cursor *string, limit int) []*TwitchStream {
	if limit == 0 {
		return make([]*TwitchStream, 0)
	}
	query := make(url.Values)
	for _, id := range userIds {
		query.Add("user_id", id)
	}
	query.Add("first", "100")
	if cursor != nil {
		query.Add("after", *cursor)
	}
	req := &http.Request{
		URL: &url.URL{
			Scheme:   "https",
			Host:     "api.twitch.tv",
			Path:     "/helix/streams",
			RawQuery: query.Encode(),
		},
		Header: http.Header{
			"Authorization": {"Bearer " + t.Token},
			"Client-Id":     {t.clientId},
		},
	}
	t.mu.Lock()
	<-t.rateLimiter.C
	t.mu.Unlock()

	resp, err := t.client.Do(req)
	if err != nil {
		return make([]*TwitchStream, 0)
	}
	defer resp.Body.Close()
	streams := &StreamResponse{}
	err = json.NewDecoder(resp.Body).Decode(&streams)
	if err != nil {
		return make([]*TwitchStream, 0)
	}
	data := streams.Data
	if len(data) == 100 && streams.Pagination.Cursor != "" {
		data = append(data, t.GetStreams(userIds, &streams.Pagination.Cursor, limit-1)...)
	}
	return data
}
