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
	PoEClient            *client.PoEClient
}

func NewGuildStashService(PoEClient *client.PoEClient) *GuildStashService {
	return &GuildStashService{
		GuildStashRepository: repository.NewGuildStashRepository(),
		PoEClient:            PoEClient,
	}
}

func (s *GuildStashService) GetGuildStashesForTeam(teamId int) ([]*repository.GuildStashTab, error) {
	return s.GuildStashRepository.GetByTeam(teamId)
}
func (s *GuildStashService) GetGuildStash(tabId string, eventId int) (*repository.GuildStashTab, error) {
	return s.GuildStashRepository.GetById(tabId, eventId)
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
	responseStashes := utils.FlatMap(resp.Stashes, func(stash client.GuildStashTab) []*client.GuildStashTab {
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
				Items:   "[]",
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
