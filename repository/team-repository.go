package repository

import (
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type Team struct {
	ID             int            `gorm:"primaryKey"`
	Name           string         `gorm:"not null"`
	AllowedClasses pq.StringArray `gorm:"type:text[]"`
	EventID        int            `gorm:"not null references events(id)"`
	Users          []*User        `gorm:"many2many:team_users"`
}

type TeamUser struct {
	TeamID int `gorm:"primaryKey"`
	UserID int `gorm:"primaryKey"`
}

type TeamRepository struct {
	DB *gorm.DB
}

func NewTeamRepository(db *gorm.DB) *TeamRepository {
	return &TeamRepository{DB: db}
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
	result := r.DB.Create(team)
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
