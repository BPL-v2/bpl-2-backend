package service

import (
	"bpl/repository"

	"gorm.io/gorm"
)

type TeamService struct {
	team_repository *repository.TeamRepository
}

func NewTeamService(db *gorm.DB) *TeamService {
	return &TeamService{
		team_repository: repository.NewTeamRepository(db),
	}
}

func (e *TeamService) GetAllTeams() ([]repository.Team, error) {
	return e.team_repository.FindAll()
}

func (e *TeamService) CreateTeam(team *repository.Team) (*repository.Team, error) {
	team, err := e.team_repository.Save(team)
	if err != nil {
		return nil, err
	}
	return team, nil
}

func (e *TeamService) GetTeamById(teamId int) (*repository.Team, error) {
	return e.team_repository.GetTeamById(teamId)
}

func (e *TeamService) UpdateTeam(teamId int, updateTeam *repository.Team) (*repository.Team, error) {
	return e.team_repository.Update(teamId, updateTeam)
}

func (e *TeamService) DeleteTeam(teamId int) error {
	return e.team_repository.Delete(teamId)
}

func (e *TeamService) AddUsersToTeams(teamUsers []*repository.TeamUser) error {
	return e.team_repository.AddUsersToTeams(teamUsers)
}
