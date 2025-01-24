package client

import (
	"net/http"
	"os"
)

type LocalDiscordClient struct {
	Client  *http.Client
	BaseURL string
}

func NewLocalDiscordClient() *LocalDiscordClient {
	url := os.Getenv("DISCORD_BOT_URL")
	return &LocalDiscordClient{
		Client:  &http.Client{},
		BaseURL: url,
	}
}

func (c *LocalDiscordClient) AssignRoles() (resp *http.Response, err error) {
	return c.Client.Post(c.BaseURL+"/assign-roles", "application/json", nil)
}
