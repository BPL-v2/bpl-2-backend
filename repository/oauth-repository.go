package repository

import (
	"bpl/config"
	"fmt"
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
	UserId       int       `gorm:"primaryKey;references:user(id);constraint:OnDelete:CASCADE"`
	Provider     Provider  `gorm:"primaryKey"`
	AccessToken  string    `gorm:"not null"`
	RefreshToken string    `gorm:"null"`
	Expiry       time.Time `gorm:"not null"`
	Name         string    `gorm:"not null"`
	AccountId    string    `gorm:"not null"`

	User *User `gorm:"foreignKey:UserId"`
}

type OauthRepository struct {
	DB *gorm.DB
}

func NewOauthRepository() *OauthRepository {
	return &OauthRepository{DB: config.DatabaseConnection()}
}

func (r *OauthRepository) GetOauthByProviderAndAccountId(provider Provider, accountId string) (*Oauth, error) {
	var oauth Oauth
	result := r.DB.Preload("User").Preload("User.OauthAccounts").First(&oauth, Oauth{Provider: provider, AccountId: accountId})
	if result.Error != nil {
		return nil, result.Error
	}
	return &oauth, nil
}

func (r *OauthRepository) GetOauthByProviderAndAccountName(provider Provider, accountName string) (*Oauth, error) {
	var oauth Oauth
	result := r.DB.Preload("User").Preload("User.OauthAccounts").First(&oauth, Oauth{Provider: provider, Name: accountName})
	if result.Error != nil {
		return nil, result.Error
	}
	return &oauth, nil
}

func (r *OauthRepository) GetAllOauths() ([]*Oauth, error) {
	var oauths []*Oauth
	result := r.DB.Find(&oauths)
	if result.Error != nil {
		return nil, result.Error
	}
	return oauths, nil
}

func (r *OauthRepository) DeleteOauthsByUserIdAndProvider(userId int, provider Provider) error {
	query := r.DB.Where("user_id = ? AND provider = ?", userId, provider)
	result := query.Delete(&Oauth{})
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (r *OauthRepository) GetOauthForTokenRefresh(provider Provider) (*Oauth, error) {
	var oauth *Oauth
	result := r.DB.Preload("User").Where("provider = ? AND refresh_token != '' AND expiry > NOW()", provider).Order("expiry ASC").First(&oauth)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get user for token refresh: %v", result.Error)
	}
	return oauth, nil
}

func (r *OauthRepository) SaveOauth(oauth *Oauth) (*Oauth, error) {
	result := r.DB.Save(oauth)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to save oauth: %v", result.Error)
	}
	return oauth, nil
}
