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

type ClientCredentialsRepository struct {
	DB *gorm.DB
}

func NewClientCredentialsRepository() *ClientCredentialsRepository {
	return &ClientCredentialsRepository{DB: config.DatabaseConnection()}
}

func (r *ClientCredentialsRepository) GetClientCredentialsByName(provider Provider) (*ClientCredentials, error) {
	var clientCredentials ClientCredentials
	result := r.DB.First(&clientCredentials, ClientCredentials{Name: provider})
	if result.Error != nil {
		return nil, result.Error
	}
	return &clientCredentials, nil
}
