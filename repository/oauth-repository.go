package repository

import (
	"bpl/config"
	"time"

	"gorm.io/gorm"
)

type Provider string

const (
	ProviderPoE     Provider = "poe"
	ProviderTwitch  Provider = "twitch"
	ProviderDiscord Provider = "discord"
)

type Oauth struct {
	UserID       int       `gorm:"primaryKey"`
	Provider     Provider  `gorm:"primaryKey"`
	AccessToken  string    `gorm:"not null"`
	RefreshToken string    `gorm:"null"`
	Expiry       time.Time `gorm:"not null"`
	Name         string    `gorm:"not null"`
	AccountID    string    `gorm:"not null"`

	User *User `gorm:"foreignKey:UserID"`
}

type OauthRepository struct {
	DB *gorm.DB
}

func NewOauthRepository() *OauthRepository {
	return &OauthRepository{DB: config.DatabaseConnection()}
}

func (r *OauthRepository) GetOauthByProviderAndAccountID(provider Provider, accountID string) (*Oauth, error) {
	var oauth Oauth
	result := r.DB.Preload("User").Preload("User.OauthAccounts").First(&oauth, "provider = ? AND account_id = ?", provider, accountID)
	if result.Error != nil {
		return nil, result.Error
	}
	return &oauth, nil
}
