package service

import (
	"bpl/repository"
	"bpl/utils"
)

type TeamService struct {
	teamRepository *repository.TeamRepository
	userRepository *repository.UserRepository
}

func NewTeamService() *TeamService {
	return &TeamService{
		teamRepository: repository.NewTeamRepository(),
		userRepository: repository.NewUserRepository(),
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

func (e *TeamService) GetTeamUsersForEvent(eventId int) ([]*repository.TeamUser, error) {
	return e.teamRepository.GetTeamUsersForEvent(eventId)
}

func (e *TeamService) GetTeamUserMapForEvent(event *repository.Event) (*map[int]int, error) {
	teamUsers, err := e.GetTeamUsersForEvent(event.Id)
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

type SortedUser struct {
	UserId      int    `json:"user_id" binding:"required"`
	DisplayName string `json:"display_name" binding:"required"`
	PoEName     string `json:"poe_name" binding:"required"`
	DiscordName string `json:"discord_name" binding:"required"`
	DiscordId   string `json:"discord_id" binding:"required"`
	TeamId      int    `json:"team_id" binding:"required"`
	IsTeamLead  bool   `json:"is_team_lead" binding:"required"`
}

func (e *TeamService) GetSortedUsersForEvent(eventId int) ([]*SortedUser, error) {
	teamUsers, err := e.GetTeamUsersForEvent(eventId)
	if err != nil {
		return nil, err
	}
	userIds := utils.Map(teamUsers, func(teamUser *repository.TeamUser) int {
		return teamUser.UserId
	})
	users, err := e.userRepository.GetUsersByIds(userIds, "OauthAccounts")
	if err != nil {
		return nil, err
	}
	userMap := make(map[int]*repository.User)
	for _, user := range users {
		userMap[user.Id] = user
	}
	sortedUsers := make([]*SortedUser, 0)
	for _, teamUser := range teamUsers {
		if user, ok := userMap[teamUser.UserId]; ok {
			sortedUser := &SortedUser{
				UserId:      user.Id,
				DisplayName: user.DisplayName,
				TeamId:      teamUser.TeamId,
				IsTeamLead:  teamUser.IsTeamLead,
			}
			for _, account := range user.OauthAccounts {
				if account.Provider == repository.ProviderPoE {
					sortedUser.PoEName = account.Name
				} else if account.Provider == repository.ProviderDiscord {
					sortedUser.DiscordName = account.Name
					sortedUser.DiscordId = account.AccountId
				}
			}
			sortedUsers = append(sortedUsers, sortedUser)
		}
	}
	return sortedUsers, nil
}
