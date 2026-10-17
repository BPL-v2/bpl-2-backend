package repository

import (
	"bpl/config"
	"bpl/metrics"
	"bpl/utils"
	"time"

	"github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
	"gorm.io/gorm"
)

type Team struct {
	Id             int            `gorm:"primaryKey"`
	Name           string         `gorm:"not null"`
	Abbreviation   string         `gorm:"not null"`
	AllowedClasses pq.StringArray `gorm:"not null;type:text[]"`
	EventId        int            `gorm:"not null;references events(id)"`
	Color          string         `gorm:"not null"`
	DiscordRoleId  *string        `gorm:"null"`
	IconMimeType   *string        `gorm:"null"`
	IconUpdatedAt  *time.Time     `gorm:"null"`
	Users          []*User        `gorm:"many2many:team_users"`
}

type TeamIcon struct {
	TeamId   int    `gorm:"primaryKey;references teams(id)"`
	Icon     []byte `gorm:"type:bytea;not null"`
	MimeType string `gorm:"not null"`
}

type TeamUser struct {
	TeamId     int       `gorm:"index;primaryKey"`
	UserId     int       `gorm:"index;primaryKey"`
	IsTeamLead bool      `gorm:"not null;default:false"`
	SortedAt   time.Time `gorm:"not null;autoCreateTime;default:CURRENT_TIMESTAMP"`
}

type TeamRepository interface {
	GetTeamById(teamId int) (*Team, error)
	GetTeamsForEvent(eventId int) ([]*Team, error)
	Save(team *Team) (*Team, error)
	Delete(teamId int) error
	SaveIcon(teamId int, icon []byte, mimeType string) error
	GetIcon(teamId int) (*TeamIcon, error)
	DeleteIcon(teamId int) error
	GetTeamUsersForEvent(eventId int) ([]*TeamUser, error)
	GetTeamUsersForTeam(teamId int) ([]*TeamUser, error)
	GetTeamLeadsForEvent(eventId int) ([]*TeamUser, error)
	RemoveTeamUsersForEvent(teamUsers []*TeamUser, event *Event) error
	RemoveUserForEvent(userId int, eventId int) error
	AddUsersToTeams(teamUsers []*TeamUser) error
	GetTeamForUser(eventId int, userId int) (*TeamUser, error)
	GetAllTeamUsers() ([]*TeamUser, error)
	GetAllTeamLeadUserIds() ([]int, error)
	GetNumbersOfPastEventsParticipatedByUsers(userIds []int) (map[int]int, error)
}

type TeamRepositoryImpl struct {
	DB *gorm.DB
}

func NewTeamRepository() TeamRepository {
	return &TeamRepositoryImpl{DB: config.DatabaseConnection()}
}

func (r *TeamRepositoryImpl) GetTeamById(teamId int) (*Team, error) {
	var team Team
	result := r.DB.First(&team, teamId)
	if result.Error != nil {
		return nil, translate(result.Error, ErrTeamNotFound)
	}
	return &team, nil
}

func (r *TeamRepositoryImpl) GetTeamsForEvent(eventId int) ([]*Team, error) {
	var teams []*Team
	result := r.DB.Find(&teams, Team{EventId: eventId})
	if result.Error != nil {
		return nil, result.Error
	}
	return teams, nil
}

func (r *TeamRepositoryImpl) Save(team *Team) (*Team, error) {
	result := r.DB.Omit("IconMimeType", "IconUpdatedAt").Save(team)
	if result.Error != nil {
		return nil, result.Error
	}
	return team, nil
}

func (r *TeamRepositoryImpl) SaveIcon(teamId int, icon []byte, mimeType string) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&TeamIcon{TeamId: teamId, Icon: icon, MimeType: mimeType}).Error; err != nil {
			return err
		}
		return tx.Model(&Team{}).Where("id = ?", teamId).Updates(map[string]any{
			"icon_mime_type":  mimeType,
			"icon_updated_at": time.Now(),
		}).Error
	})
}

func (r *TeamRepositoryImpl) GetIcon(teamId int) (*TeamIcon, error) {
	var icon TeamIcon
	if err := r.DB.First(&icon, "team_id = ?", teamId).Error; err != nil {
		return nil, translate(err, ErrTeamIconNotFound)
	}
	return &icon, nil
}

func (r *TeamRepositoryImpl) DeleteIcon(teamId int) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&TeamIcon{}, "team_id = ?", teamId).Error; err != nil {
			return err
		}
		return tx.Model(&Team{}).Where("id = ?", teamId).Updates(map[string]any{
			"icon_mime_type":  nil,
			"icon_updated_at": nil,
		}).Error
	})
}

func (r *TeamRepositoryImpl) Delete(teamId int) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		err := r.DB.Delete(&TeamUser{}, "team_id = ?", teamId).Error
		if err != nil {
			return err
		}
		if err := r.DB.Delete(&TeamIcon{}, "team_id = ?", teamId).Error; err != nil {
			return err
		}
		result := r.DB.Delete(Team{Id: teamId})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrTeamNotFound
		}
		return nil
	})
}

func (r *TeamRepositoryImpl) GetTeamUsersForEvent(eventId int) ([]*TeamUser, error) {
	timer := prometheus.NewTimer(metrics.QueryDuration.WithLabelValues("GetTeamUsersForEvent"))
	defer timer.ObserveDuration()
	teamUsers := make([]*TeamUser, 0)
	query := `
		SELECT * 
		FROM team_users
		JOIN teams ON team_users.team_id = teams.id
		WHERE teams.event_id = ?
		ORDER BY teams.id ASC
	`
	result := r.DB.Raw(query, eventId).Scan(&teamUsers)
	if result.Error != nil {
		return nil, result.Error
	}
	return teamUsers, nil
}

func (r *TeamRepositoryImpl) GetTeamUsersForTeam(teamId int) ([]*TeamUser, error) {
	timer := prometheus.NewTimer(metrics.QueryDuration.WithLabelValues("GetTeamUsersForTeam"))
	defer timer.ObserveDuration()
	teamUsers := make([]*TeamUser, 0)
	result := r.DB.Where("team_id = ?", teamId).Find(&teamUsers)
	if result.Error != nil {
		return nil, result.Error
	}
	return teamUsers, nil

}

func (r *TeamRepositoryImpl) GetTeamLeadsForEvent(eventId int) ([]*TeamUser, error) {
	timer := prometheus.NewTimer(metrics.QueryDuration.WithLabelValues("GetTeamLeadsForEvent"))
	defer timer.ObserveDuration()
	teamUsers := make([]*TeamUser, 0)
	query := `
		SELECT * 
		FROM team_users
		JOIN teams ON team_users.team_id = teams.id
		WHERE teams.event_id = ?
		AND team_users.is_team_lead = true
	`
	result := r.DB.Raw(query, eventId).Scan(&teamUsers)
	if result.Error != nil {
		return nil, result.Error
	}
	return teamUsers, nil
}

func (r *TeamRepositoryImpl) RemoveTeamUsersForEvent(teamUsers []*TeamUser, event *Event) error {
	query := `
		DELETE FROM team_users
		WHERE team_id IN (
			SELECT id
			FROM teams
			WHERE event_id = ?
		)
		AND user_id IN (?)
	`
	result := r.DB.Exec(query, event.Id, utils.Map(teamUsers, func(teamUser *TeamUser) int {
		return teamUser.UserId
	}))
	return result.Error
}

func (r *TeamRepositoryImpl) RemoveUserForEvent(userId int, eventId int) error {
	query := `
		DELETE FROM team_users
		WHERE team_id IN (
			SELECT id
			FROM teams
			WHERE event_id = ?
		)
		AND user_id = ?
	`
	result := r.DB.Exec(query, eventId, userId)
	return result.Error
}

func (r *TeamRepositoryImpl) AddUsersToTeams(teamUsers []*TeamUser) error {
	validTeamUsers := utils.Filter(teamUsers, func(teamUser *TeamUser) bool {
		return teamUser.TeamId != 0
	})
	result := r.DB.CreateInBatches(validTeamUsers, len(validTeamUsers))
	return result.Error
}

func (r *TeamRepositoryImpl) GetTeamForUser(eventId int, userId int) (*TeamUser, error) {
	team := &TeamUser{}
	query := `
		SELECT team_users.*
		FROM team_users
		JOIN teams ON team_users.team_id = teams.id
		WHERE team_users.user_id = ?
		AND teams.event_id = ?
	`
	err := r.DB.Raw(query, userId, eventId).First(&team).Error
	if err != nil {
		return nil, translate(err, ErrTeamUserNotFound)
	}
	return team, nil
}

func (r *TeamRepositoryImpl) GetAllTeamUsers() ([]*TeamUser, error) {
	teamUsers := make([]*TeamUser, 0)
	result := r.DB.Find(&teamUsers)
	if result.Error != nil {
		return nil, result.Error
	}
	return teamUsers, nil
}

func (r *TeamRepositoryImpl) GetAllTeamLeadUserIds() ([]int, error) {
	var userIds []int
	result := r.DB.Model(&TeamUser{}).
		Where("is_team_lead = ?", true).
		Distinct("user_id").
		Pluck("user_id", &userIds)
	if result.Error != nil {
		return nil, result.Error
	}
	return userIds, nil
}

func (r *TeamRepositoryImpl) GetNumbersOfPastEventsParticipatedByUsers(userIds []int) (map[int]int, error) {
	type Result struct {
		UserId         int
		NumberOfEvents int
	}
	results := make([]*Result, 0)
	query := `
		SELECT user_id, COUNT(DISTINCT team_id) AS number_of_events FROM team_users
		JOIN teams ON team_users.team_id = teams.id
		JOIN events ON teams.event_id = events.id
		WHERE user_id IN (?) AND events.is_current = false
		GROUP BY user_id
	`
	err := r.DB.Raw(query, userIds).Scan(&results).Error
	if err != nil {
		return nil, err
	}
	resultMap := make(map[int]int, 0)
	for _, result := range results {
		resultMap[result.UserId] = result.NumberOfEvents
	}
	return resultMap, nil
}
