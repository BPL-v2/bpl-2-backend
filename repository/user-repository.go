package repository

import (
	"fmt"

	"github.com/lib/pq"
	"gorm.io/gorm"
)

type Permission string

const (
	PermissionAdmin       Permission = "admin"
	PermissionCommandTeam Permission = "command_team"
)

type User struct {
	ID                int            `gorm:"primaryKey autoIncrement"`
	AccountName       string         `gorm:"null"`
	DiscordID         int64          `gorm:"null"`
	DiscordName       string         `gorm:"null"`
	PoeToken          string         `gorm:"null"`
	PoeTokenExpiresAt int64          `gorm:"null"`
	Permissions       pq.StringArray `gorm:"type:text[];not null;default:'{}'"`
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
		return nil, fmt.Errorf("user with discord id %s not found", discordId)
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
