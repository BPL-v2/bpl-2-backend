package service

import (
	"bpl/repository"
)

type TeamService struct {
	teamRepository *repository.TeamRepository
}

func NewTeamService() *TeamService {
	return &TeamService{
		teamRepository: repository.NewTeamRepository(),
	}
}

func (e *TeamService) GetTeamsForEvent(eventId int) ([]*repository.Team, error) {
	return e.teamRepository.GetTeamsForEvent(eventId)
}

func (e *TeamService) SaveTeam(team *repository.Team) (*repository.Team, error) {
	team, err := e.teamRepository.Save(team)
	if err != nil {
		return nil, err
	}
	return team, nil
}

func (e *TeamService) GetTeamById(teamId int) (*repository.Team, error) {
	return e.teamRepository.GetTeamById(teamId)
}

func (e *TeamService) DeleteTeam(teamId int) error {
	return e.teamRepository.Delete(teamId)
}

func (e *TeamService) AddUsersToTeams(teamUsers []*repository.TeamUser, event *repository.Event) error {
	err := e.teamRepository.RemoveTeamUsersForEvent(teamUsers, event)
	if err != nil {
		return err
	}
	return e.teamRepository.AddUsersToTeams(teamUsers)
}

func (e *TeamService) GetTeamUsersForEvent(event *repository.Event) ([]*repository.TeamUser, error) {
	return e.teamRepository.GetTeamUsersForEvent(event)
}

func (e *TeamService) GetTeamUserMapForEvent(event *repository.Event) (*map[int]int, error) {
	teamUsers, err := e.GetTeamUsersForEvent(event)
	if err != nil {
		return nil, err
	}
	userToTeam := make(map[int]int)
	for _, teamUser := range teamUsers {
		userToTeam[teamUser.UserId] = teamUser.TeamId
	}
	return &userToTeam, nil
}

func (e *TeamService) GetTeamForUser(eventId int, userId int) (*repository.TeamUser, error) {
	return e.teamRepository.GetTeamForUser(eventId, userId)
}

func (e *TeamService) GetTeamLeadsForEvent(eventId int) (map[int][]*repository.TeamUser, error) {
	leads, err := e.teamRepository.GetTeamLeadsForEvent(eventId)
	if err != nil {
		return nil, err
	}
	teamLeads := make(map[int][]*repository.TeamUser)
	for _, teamLead := range leads {
		if _, ok := teamLeads[teamLead.TeamId]; !ok {
			teamLeads[teamLead.TeamId] = make([]*repository.TeamUser, 0)
		}
		teamLeads[teamLead.TeamId] = append(teamLeads[teamLead.TeamId], teamLead)
	}
	return teamLeads, nil
}
