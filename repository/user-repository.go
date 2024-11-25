package repository

import (
	"fmt"

	"gorm.io/gorm"
)

type User struct {
	ID                int      `gorm:"primaryKey"`
	AccountName       string   `gorm:"not null"`
	DiscordID         int64    `gorm:"not null"`
	PoEToken          string   `gorm:"not null"`
	PoeTokenExpiresAt int      `gorm:"not null"`
	Permissions       []string `gorm:"type:text[]"`
}

type UserRepository struct {
	DB *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{DB: db}
}

func (r *UserRepository) GetUserById(userId int) (*User, error) {
	var user User
	query := r.DB

	result := query.First(&user, userId)
	if result.Error != nil {
		return nil, fmt.Errorf("user with id %d not found", userId)
	}
	return &user, nil
}
