package repository

import (
	"bpl/config"
	"bpl/utils"
	"database/sql/driver"
	"fmt"
	"log"
	"time"

	"github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
	"gorm.io/gorm"
)

type Permission string

const (
	PermissionAdmin             Permission = "admin"
	PermissionManager           Permission = "manager"
	PermissionObjectiveDesigner Permission = "objective_designer"
	PermissionSubmissionJudge   Permission = "submission_judge"
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
	Id          int         `gorm:"primaryKey autoIncrement"`
	DisplayName string      `gorm:"not null"`
	Permissions Permissions `gorm:"type:text[];not null;default:'{}'"`

	OauthAccounts []*Oauth `gorm:"foreignKey:UserId"`
}

type UserRepository struct {
	DB *gorm.DB
}

func NewUserRepository() *UserRepository {
	return &UserRepository{DB: config.DatabaseConnection()}
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

func (r *UserRepository) SaveUser(user *User) (*User, error) {
	result := r.DB.Save(user)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to create user: %v", result.Error)
	}
	return user, nil
}

func (r *UserRepository) GetAllUsers() ([]*User, error) {
	var users []*User
	result := r.DB.Find(&users)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get users: %v", result.Error)
	}
	return users, nil
}

type Streamer struct {
	UserId   int
	TwitchId string
}

func (r *UserRepository) GetStreamersForCurrentEvent() (streamers []*Streamer, err error) {
	timer := prometheus.NewTimer(queryDuration.WithLabelValues("GetStreamersForCurrentEvent"))
	defer timer.ObserveDuration()
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

type UserTeam struct {
	UserId      int
	TeamId      int
	DisplayName string
}

func LoadUsersIntoEvent(DB *gorm.DB, event *Event) error {
	timer := prometheus.NewTimer(queryDuration.WithLabelValues("LoadUsersIntoEvent"))
	defer timer.ObserveDuration()
	var users []*UserTeam
	query := `
		SELECT
			users.id as user_id,
			users.display_name as display_name,	
			team_users.team_id as team_id
		FROM users
		JOIN team_users ON team_users.user_id = users.id
		WHERE team_users.team_id IN ?		
		`
	err := DB.Raw(query, utils.Map(event.Teams, func(team *Team) int {
		return team.Id
	})).Scan(&users).Error
	if err != nil {
		log.Print(err)
		return fmt.Errorf("failed to load users into event: %v", err)
	}
	for _, user := range users {
		for _, team := range event.Teams {
			if team.Id == user.TeamId {
				team.Users = append(team.Users, &User{Id: user.UserId, DisplayName: user.DisplayName})
			}
		}
	}
	return nil

}

type TeamUserWithPoEAccountName struct {
	TeamId      int
	UserId      int
	AccountName string
}

type TeamUserWithPoEToken struct {
	TeamId      int
	UserId      int
	AccountName string
	Token       string
	TokenExpiry time.Time
}

func (r *UserRepository) GetUsersForEvent(eventId int) ([]*TeamUserWithPoEAccountName, error) {
	timer := prometheus.NewTimer(queryDuration.WithLabelValues("GetUsersForEvent"))
	defer timer.ObserveDuration()
	var users []*TeamUserWithPoEAccountName
	query := `
		SELECT
			users.id as user_id,
			oauths.name as account_name,
			team_users.team_id as team_id
		FROM users
		JOIN oauths ON oauths.user_id = users.id
		JOIN team_users ON team_users.user_id = users.id
		JOIN teams ON teams.id = team_users.team_id
		WHERE teams.event_id = ? AND oauths.provider = 'poe'
	`
	result := r.DB.Raw(query, eventId).Scan(&users)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get users for event: %v", result.Error)
	}
	return users, nil
}

func (r *UserRepository) GetAuthenticatedUsersForEvent(eventId int) ([]*TeamUserWithPoEToken, error) {
	timer := prometheus.NewTimer(queryDuration.WithLabelValues("GetAuthenticatedUsersForEvent"))
	defer timer.ObserveDuration()
	var users []*TeamUserWithPoEToken
	query := `
		SELECT
			users.id as user_id,
			oauths.access_token as token,
			oauths.expiry as token_expiry,
			oauths.name as account_name,
			team_users.team_id as team_id
		FROM users
		JOIN oauths ON oauths.user_id = users.id
		JOIN team_users ON team_users.user_id = users.id
		JOIN teams ON teams.id = team_users.team_id
		WHERE teams.event_id = ? AND oauths.provider = 'poe'
	`
	result := r.DB.Raw(query, eventId).Scan(&users)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get users for event: %v", result.Error)
	}
	return users, nil
}
