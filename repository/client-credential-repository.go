package repository

import (
	"bpl/config"
	"time"

	"gorm.io/gorm"
)

type ClientCredentials struct {
	Name        Provider   `gorm:"primaryKey"`
	AccessToken string     `json:"access_token"`
	Expiry      *time.Time `gorm:"null" json:"expiry"`
}

type ClientCredentialsRepository interface {
	SaveClientCredentials(credentials *ClientCredentials) error
	GetClientCredentialsByName(provider Provider) (*ClientCredentials, error)
}

type ClientCredentialsRepositoryImpl struct {
	DB *gorm.DB
}

func NewClientCredentialsRepository() ClientCredentialsRepository {
	return &ClientCredentialsRepositoryImpl{DB: config.DatabaseConnection()}
}

func (r *ClientCredentialsRepositoryImpl) SaveClientCredentials(credentials *ClientCredentials) error {
	return r.DB.Save(credentials).Error
}

func (r *ClientCredentialsRepositoryImpl) GetClientCredentialsByName(provider Provider) (*ClientCredentials, error) {
	var clientCredentials ClientCredentials
	result := r.DB.First(&clientCredentials, ClientCredentials{Name: provider})
	if result.Error != nil {
		return nil, result.Error
	}
	return &clientCredentials, nil
}
