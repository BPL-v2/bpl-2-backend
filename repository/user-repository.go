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
	ID          int         `gorm:"primaryKey autoIncrement"`
	DisplayName string      `gorm:"not null"`
	Permissions Permissions `gorm:"type:text[];not null;default:'{}'"`

	OauthAccounts []*Oauth `gorm:"foreignKey:UserID"`
}

type UserRepository struct {
	DB *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{DB: db}
}

func (r *UserRepository) GetUserById(userId int, preloads ...string) (*User, error) {
	var user User
	query := r.DB
	for _, preload := range preloads {
		query = query.Preload(preload)
	}
	result := query.First(&user, userId)
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

func (r *UserRepository) GetUsers(preloads ...string) ([]*User, error) {
	var users []*User
	query := r.DB
	for _, preload := range preloads {
		query = query.Preload(preload)
	}
	result := query.Find(&users)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get users: %v", result.Error)
	}
	return users, nil
}

type Streamer struct {
	UserID   int
	TwitchID string
}

func (r *UserRepository) GetStreamersForCurrentEvent() ([]*Streamer, error) {
	var streamers []*Streamer

	query := `
		SELECT 
			users.id as user_id, 
			oauths.account_id as twitch_id
		FROM users
		JOIN oauths ON oauths.user_id = users.id
		JOIN team_users ON team_users.user_id = users.id
		JOIN teams ON teams.id = team_users.team_id
		JOIN events ON events.id = teams.event_id
		WHERE events.is_current = true AND oauths.provider = 'twitch'
	`
	result := r.DB.Raw(query).Scan(&streamers)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get streamers for current event: %v", result.Error)
	}

	return streamers, nil

}
