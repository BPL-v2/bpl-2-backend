package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

type LocalDiscordClient struct {
	Client   *http.Client
	BaseURL  string
	ServerId string
}

func NewLocalDiscordClient() *LocalDiscordClient {
	url := os.Getenv("DISCORD_BOT_URL")
	serverId := os.Getenv("DISCORD_GUILD_ID")
	return &LocalDiscordClient{
		Client:   &http.Client{},
		BaseURL:  url,
		ServerId: serverId,
	}
}

func (c *LocalDiscordClient) AssignRoles() (*http.Response, error) {
	return c.Client.Post(fmt.Sprintf("%s/%s/assign-roles", c.BaseURL, c.ServerId), "application/json", nil)
}

func (c *LocalDiscordClient) GetServerMemberIds() ([]string, error) {
	resp, err := c.Client.Get(fmt.Sprintf("%s/%s/members", c.BaseURL, c.ServerId))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0)
	err = json.Unmarshal(respBody, &ids)
	if err != nil {
		return nil, err
	}
	return ids, nil
}
