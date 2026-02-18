package service

import (
	"bpl/client"
	"bpl/repository"
	"bpl/utils"
	"fmt"
	"time"
)

type GuildStashService struct {
	GuildStashRepository *repository.GuildStashRepository
	TeamRepository       *repository.TeamRepository
	ObjectiveService     *ObjectiveService
	PoEClient            *client.PoEClient
}

func NewGuildStashService(PoEClient *client.PoEClient) *GuildStashService {
	return &GuildStashService{
		GuildStashRepository: repository.NewGuildStashRepository(),
		TeamRepository:       repository.NewTeamRepository(),
		ObjectiveService:     NewObjectiveService(),
		PoEClient:            PoEClient,
	}
}

func (s *GuildStashService) GetGuildStashesForUserForEvent(user repository.User, event repository.Event) ([]*repository.GuildStashTab, error) {
	return s.GuildStashRepository.GetByUserAndEvent(user.Id, event.Id)
}

func (s *GuildStashService) GetGuildStashesForTeam(teamId int) ([]*repository.GuildStashTab, error) {
	return s.GuildStashRepository.GetByTeam(teamId)
}
func (s *GuildStashService) GetGuildStash(tabId string, eventId int, preloads ...string) (*repository.GuildStashTab, error) {
	return s.GuildStashRepository.GetById(tabId, eventId, preloads...)
}

func (s *GuildStashService) UpdateGuildStash(user *repository.User, teamId int, event *repository.Event) ([]*repository.GuildStashTab, error) {
	token, found := utils.FindFirst(user.OauthAccounts, func(o *repository.Oauth) bool {
		return o.Provider == repository.ProviderPoE
	})
	if !found || token.AccessToken == "" || token.Expiry.Before(time.Now()) {
		return nil, fmt.Errorf("invalid PoE token")
	}
	resp, httpError := s.PoEClient.ListGuildStashes(token.AccessToken, event.Name)
	if httpError != nil {
		return nil, fmt.Errorf("failed to fetch guild stash tabs: %d - %s", httpError.StatusCode, httpError.Description)
	}
	existingStashes, err := s.GuildStashRepository.GetByTeam(teamId)
	if err != nil {
		return nil, err
	}
	stashMap := make(map[string]*repository.GuildStashTab)
	for _, stash := range existingStashes {
		stashMap[stash.Id] = stash
	}
	stashesToPersist := make([]*repository.GuildStashTab, 0)
	responseStashes := utils.FlatMap(resp.Stashes, func(stash client.GuildStashTabGGG) []*client.GuildStashTabGGG {
		return stash.FlatMap()
	})
	for _, stash := range responseStashes {
		if existingStash, exists := stashMap[stash.Id]; exists {
			existingStash.UserIds = append(utils.Filter(existingStash.UserIds, func(id int32) bool {
				return id != int32(user.Id)
			}), int32(user.Id))
			existingStash.Index = stash.Index
			existingStash.Name = stash.Name
			existingStash.Type = stash.Type
			existingStash.Color = stash.Metadata.Colour
			existingStash.Owner.Id = user.Id
		} else {
			newStash := &repository.GuildStashTab{
				Index:   stash.Index,
				Id:      stash.Id,
				EventId: event.Id,
				TeamId:  teamId,
				OwnerId: user.Id,
				Name:    stash.Name,
				Type:    stash.Type,
				Color:   stash.Metadata.Colour,
				UserIds: utils.ConvertIntSlice([]int{user.Id}),
				Raw:     "{}",
			}
			if stash.Parent != nil {
				newStash.ParentId = stash.Parent
				newStash.ParentEventId = &event.Id
			}
			stashMap[stash.Id] = newStash
		}
		stashesToPersist = append(stashesToPersist, stashMap[stash.Id])
	}

	err = s.GuildStashRepository.SaveAll(stashesToPersist)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert guild stash tabs: %w", err)
	}
	return stashesToPersist, nil
}

func (s *GuildStashService) SwitchStashFetch(stashId string, eventId int) (*repository.GuildStashTab, error) {
	return s.GuildStashRepository.SwitchStashFetch(stashId, eventId)
}

func (s *GuildStashService) SaveGuildstashLogs(stashLogs []*repository.GuildStashChangelog) error {
	return s.GuildStashRepository.SaveGuildstashLogs(stashLogs)
}

func (s *GuildStashService) GetLatestLogEntryTimestampForGuild(event *repository.Event, guildId int) (*int64, *int64) {
	return s.GuildStashRepository.GetLatestLogEntryTimestampForGuild(event, guildId)
}

func (s *GuildStashService) GetLogs(eventId, guildId int, limit, offset *int, userName, stashName, itemName *string) ([]*repository.GuildStashChangelog, error) {
	return s.GuildStashRepository.GetLogs(eventId, guildId, limit, offset, userName, stashName, itemName)
}

func (s *GuildStashService) SaveGuild(guild *repository.Guild) error {
	return s.GuildStashRepository.SaveGuild(guild)
}

func (s *GuildStashService) GetGuildsForEvent(event *repository.Event) ([]*repository.Guild, error) {
	teams, err := s.TeamRepository.GetTeamsForEvent(event.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to get teams for event: %w", err)
	}
	return s.GuildStashRepository.GetGuildsForTeams(utils.Map(teams, func(team *repository.Team) int {
		return team.Id
	}))
}

func (s *GuildStashService) GetGuildById(guildId int, eventId int) (*repository.Guild, error) {
	return s.GuildStashRepository.GetGuildById(guildId, eventId)
}

func (s *GuildStashService) GetEarliestDeposits(event *repository.Event) ([]*repository.PlayerCompletion, error) {
	return s.GuildStashRepository.GetEarliestDeposits(event)
}
