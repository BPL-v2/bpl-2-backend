package repository

import (
	"database/sql/driver"
	"fmt"

	"github.com/lib/pq"
	"gorm.io/gorm"
)

type Permission string

const (
	PermissionAdmin       Permission = "admin"
	PermissionCommandTeam Permission = "command_team"
)

type Permissions []Permission

func (p *Permissions) Scan(src interface{}) error {
	x := make(pq.StringArray, 0)
	x.Scan(src)
	permissions := make(Permissions, len(x))
	for i, perm := range x {
		permissions[i] = Permission(perm)
	}
	*p = permissions
	return nil
}

func (p Permissions) Value() (driver.Value, error) {
	permissions := make(pq.StringArray, len(p))
	for i, perm := range p {
		permissions[i] = string(perm)
	}
	return permissions.Value()
}

type User struct {
	ID                int         `gorm:"primaryKey autoIncrement"`
	DisplayName       string      `gorm:"not null"`
	POEAccount        *string     `gorm:"null"`
	DiscordID         *int64      `gorm:"null"`
	DiscordName       *string     `gorm:"null"`
	TwitchID          *string     `gorm:"null"`
	TwitchName        *string     `gorm:"null"`
	PoeToken          *string     `gorm:"null"`
	PoeTokenExpiresAt *int64      `gorm:"null"`
	Permissions       Permissions `gorm:"type:text[];not null;default:'{}'"`
}

type UserRepository struct {
	DB *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{DB: db}
}

func (r *UserRepository) GetUserById(userId int) (*User, error) {
	var user User
	result := r.DB.First(&user, userId)
	if result.Error != nil {
		return nil, fmt.Errorf("user with id %d not found", userId)
	}
	return &user, nil
}

func (r *UserRepository) GetUserByDiscordId(discordId int64) (*User, error) {
	var user User
	result := r.DB.First(&user, "discord_id = ?", discordId)
	if result.Error != nil {
		return nil, fmt.Errorf("user with discord id %d not found", discordId)
	}
	return &user, nil
}

func (r *UserRepository) GetUserByPoEAccount(poeAccount string) (*User, error) {
	var user User
	result := r.DB.First(&user, "poe_account = ?", poeAccount)
	if result.Error != nil {
		return nil, fmt.Errorf("user with poe account %s not found", poeAccount)
	}
	return &user, nil
}

func (r *UserRepository) GetUserByTwitchId(twitchId string) (*User, error) {
	var user User
	result := r.DB.First(&user, "twitch_id = ?", twitchId)
	if result.Error != nil {
		return nil, fmt.Errorf("user with twitch id %s not found", twitchId)
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

func (r *UserRepository) GetUsers() ([]*User, error) {
	var users []*User
	query := r.DB

	result := query.Find(&users)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get users: %v", result.Error)
	}
	return users, nil
}

func (r *UserRepository) GetStreamersForCurrentEvent() ([]*User, error) {
	var users []*User
	query := r.DB.Table("users").
		Select("users.*").
		Joins("JOIN team_users ON team_users.user_id = users.id").
		Joins("JOIN teams ON teams.id = team_users.team_id").
		Joins("JOIN events ON events.id = teams.event_id").
		Where("events.is_current = true").
		Where("users.twitch_id IS NOT NULL").
		Find(&users)
	if query.Error != nil {
		return nil, fmt.Errorf("failed to get streamers for current event: %v", query.Error)
	}
	return users, nil

}
