package service

import (
	"bpl/repository"
)

type TeamService struct {
	team_repository *repository.TeamRepository
}

func NewTeamService() *TeamService {
	return &TeamService{
		team_repository: repository.NewTeamRepository(),
	}
}

func (e *TeamService) GetAllTeams() ([]repository.Team, error) {
	return e.team_repository.FindAll()
}

func (e *TeamService) SaveTeam(team *repository.Team) (*repository.Team, error) {
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

func (e *TeamService) AddUsersToTeams(teamUsers []*repository.TeamUser, event *repository.Event) error {
	err := e.team_repository.RemoveTeamUsersForEvent(teamUsers, event)
	if err != nil {
		return err
	}
	return e.team_repository.AddUsersToTeams(teamUsers)
}

func (e *TeamService) GetTeamUsersForEvent(event *repository.Event) ([]*repository.TeamUser, error) {
	return e.team_repository.GetTeamUsersForEvent(event)
}

func (e *TeamService) GetTeamUserMapForEvent(event *repository.Event) (*map[int]int, error) {
	teamUsers, err := e.GetTeamUsersForEvent(event)
	if err != nil {
		return nil, err
	}
	userToTeam := make(map[int]int)
	for _, teamUser := range teamUsers {
		userToTeam[teamUser.UserID] = teamUser.TeamID
	}
	return &userToTeam, nil
}

func (e *TeamService) GetTeamForUser(eventId int, userId int) (*repository.Team, error) {
	return e.team_repository.GetTeamForUser(eventId, userId)
}
