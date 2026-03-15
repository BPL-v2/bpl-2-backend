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
	UserId        int       `gorm:"primaryKey;references:user(id);constraint:OnDelete:CASCADE"`
	Provider      Provider  `gorm:"primaryKey"`
	AccessToken   string    `gorm:"not null"`
	RefreshToken  string    `gorm:"null"`
	Expiry        time.Time `gorm:"not null"`
	RefreshExpiry time.Time `gorm:"not null"`
	Name          string    `gorm:"not null"`
	AccountId     string    `gorm:"not null"`

	User *User `gorm:"foreignKey:UserId"`
}

type OauthRepository interface {
	GetOauthByProviderAndAccountId(provider Provider, accountId string) (*Oauth, error)
	GetOauthByProviderAndAccountName(provider Provider, accountName string) (*Oauth, error)
	GetAllOauths() ([]*Oauth, error)
	DeleteOauthsByUserIdAndProvider(userId int, provider Provider) error
	GetOauthForTokenRefresh(provider Provider) (*Oauth, error)
	SaveOauth(oauth *Oauth) (*Oauth, error)
}

type OauthRepositoryImpl struct {
	DB *gorm.DB
}

func NewOauthRepository() OauthRepository {
	return &OauthRepositoryImpl{DB: config.DatabaseConnection()}
}

func (r *OauthRepositoryImpl) GetOauthByProviderAndAccountId(provider Provider, accountId string) (*Oauth, error) {
	var oauth Oauth
	result := r.DB.Preload("User").Preload("User.OauthAccounts").First(&oauth, Oauth{Provider: provider, AccountId: accountId})
	if result.Error != nil {
		return nil, result.Error
	}
	return &oauth, nil
}

func (r *OauthRepositoryImpl) GetOauthByProviderAndAccountName(provider Provider, accountName string) (*Oauth, error) {
	var oauth Oauth
	result := r.DB.Preload("User").Preload("User.OauthAccounts").First(&oauth, Oauth{Provider: provider, Name: accountName})
	if result.Error != nil {
		return nil, result.Error
	}
	return &oauth, nil
}

func (r *OauthRepositoryImpl) GetAllOauths() ([]*Oauth, error) {
	var oauths []*Oauth
	result := r.DB.Find(&oauths)
	if result.Error != nil {
		return nil, result.Error
	}
	return oauths, nil
}

func (r *OauthRepositoryImpl) DeleteOauthsByUserIdAndProvider(userId int, provider Provider) error {
	query := r.DB.Where("user_id = ? AND provider = ?", userId, provider)
	result := query.Delete(&Oauth{})
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (r *OauthRepositoryImpl) GetOauthForTokenRefresh(provider Provider) (*Oauth, error) {
	var oauth *Oauth
	result := r.DB.Preload("User").Where(`
		provider = ? AND
		refresh_token != '' AND
		refresh_expiry > NOW() AND
		expiry <= NOW() + INTERVAL '1 day' AND
		ABS(EXTRACT(EPOCH FROM (refresh_expiry - expiry))) > 1
	`, provider).Order("expiry ASC").First(&oauth)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get user for token refresh: %v", result.Error)
	}
	return oauth, nil
}

func (r *OauthRepositoryImpl) SaveOauth(oauth *Oauth) (*Oauth, error) {
	result := r.DB.Save(oauth)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to save oauth: %v", result.Error)
	}
	return oauth, nil
}
