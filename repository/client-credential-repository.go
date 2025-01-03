package repository

import (
	"time"

	"gorm.io/gorm"
)

type OauthProvider string

const (
	OauthProviderDiscord OauthProvider = "discord"
	OauthProviderTwitch  OauthProvider = "twitch"
	OauthProviderPoE     OauthProvider = "poe"
)

type ClientCredentials struct {
	Name        OauthProvider `gorm:"primaryKey"`
	AccessToken string        `json:"access_token"`
	Expiry      time.Time     `json:"expiry"`
}

type ClientCredentialsRepository struct {
	DB *gorm.DB
}

func NewClientCredentialsRepository(db *gorm.DB) *ClientCredentialsRepository {
	return &ClientCredentialsRepository{DB: db}
}

func (r *ClientCredentialsRepository) GetClientCredentialsByName(provider OauthProvider) (*ClientCredentials, error) {
	var clientCredentials ClientCredentials
	result := r.DB.First(&clientCredentials, "name = ?", provider)
	if result.Error != nil {
		return nil, result.Error
	}
	return &clientCredentials, nil
}
