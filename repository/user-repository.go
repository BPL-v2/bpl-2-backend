package repository

import (
	"fmt"

	"gorm.io/gorm"
)

type User struct {
	ID                int      `gorm:"primaryKey autoIncrement"`
	AccountName       string   `gorm:"null"`
	DiscordID         int64    `gorm:"null"`
	PoEToken          string   `gorm:"null"`
	PoeTokenExpiresAt int64    `gorm:"null"`
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

func (r *UserRepository) GetUserByDiscordId(discordId int64) (*User, error) {
	var user User
	query := r.DB

	result := query.First(&user, "discord_id = ?", discordId)
	if result.Error != nil {
		return nil, fmt.Errorf("user with discord id %d not found", discordId)
	}
	return &user, nil
}

func (r *UserRepository) SaveUser(user *User) (*User, error) {
	result := r.DB.Save(user)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to create user: %v", result.Error)
	}
	return user, nil
}
