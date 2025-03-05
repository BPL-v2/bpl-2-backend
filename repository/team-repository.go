package repository

import (
	"bpl/config"
	"bpl/utils"

	"github.com/lib/pq"
	"gorm.io/gorm"
)

type Team struct {
	Id             int            `gorm:"primaryKey"`
	Name           string         `gorm:"not null"`
	AllowedClasses pq.StringArray `gorm:"not null;type:text[]"`
	EventId        int            `gorm:"not null;references events(id)"`
	Users          []*User        `gorm:"many2many:team_users"`
}

type TeamUser struct {
	TeamId     int  `gorm:"index;primaryKey"`
	UserId     int  `gorm:"index;primaryKey"`
	IsTeamLead bool `gorm:"not null;default:false"`
}

type TeamRepository struct {
	DB *gorm.DB
}

func NewTeamRepository() *TeamRepository {
	return &TeamRepository{DB: config.DatabaseConnection()}
}

func (r *TeamRepository) GetTeamById(teamId int) (*Team, error) {
	var team Team
	result := r.DB.First(&team, teamId)
	if result.Error != nil {
		return nil, result.Error
	}
	return &team, nil
}

func (r *TeamRepository) Save(team *Team) (*Team, error) {
	result := r.DB.Save(team)
	if result.Error != nil {
		return nil, result.Error
	}
	return team, nil
}

func (r *TeamRepository) Update(teamId int, updateTeam *Team) (*Team, error) {
	team, err := r.GetTeamById(teamId)
	if err != nil {
		return nil, err
	}
	if updateTeam.Name != "" {
		team.Name = updateTeam.Name
	}
	result := r.DB.Save(&team)
	if result.Error != nil {
		return nil, result.Error
	}
	return team, nil
}

func (r *TeamRepository) Delete(teamId int) error {
	result := r.DB.Delete(Team{}, teamId)
	return result.Error
}

func (r *TeamRepository) FindAll() ([]Team, error) {
	var teams []Team
	result := r.DB.Find(&teams)
	if result.Error != nil {
		return nil, result.Error
	}
	return teams, nil
}

func (r *TeamRepository) GetTeamUsersForEvent(event *Event) ([]*TeamUser, error) {
	teamUsers := make([]*TeamUser, 0)
	result := r.DB.Find(&teamUsers, "team_id in ?", utils.Map(event.Teams, func(team *Team) int {
		return team.Id
	}))
	if result.Error != nil {
		return nil, result.Error
	}
	return teamUsers, nil
}

func (r *TeamRepository) RemoveTeamUsersForEvent(teamUsers []*TeamUser, event *Event) error {
	result := r.DB.Where("team_id in ? AND user_id in ?", utils.Map(event.Teams, func(team *Team) int {
		return team.Id
	}), utils.Map(teamUsers, func(user *TeamUser) int {
		return user.UserId
	})).Delete(&TeamUser{})

	return result.Error
}

func (r *TeamRepository) AddUsersToTeams(teamUsers []*TeamUser) error {
	validTeamUsers := utils.Filter(teamUsers, func(teamUser *TeamUser) bool {
		return teamUser.TeamId != 0
	})
	result := r.DB.CreateInBatches(validTeamUsers, len(validTeamUsers))
	return result.Error
}

func (r *TeamRepository) GetTeamForUser(eventId int, userId int) (*Team, error) {
	team := &Team{}
	result := r.DB.Joins("JOIN bpl2.team_users ON team_users.team_id = teams.id").
		Where("team_users.user_id = ? AND teams.event_id = ?", userId, eventId).
		First(team)
	if result.Error != nil {
		return nil, result.Error
	}
	return team, nil
}
