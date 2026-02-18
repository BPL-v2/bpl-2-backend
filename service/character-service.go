package service

import (
	"bpl/client"
	"bpl/config"
	"bpl/metrics"
	"bpl/parser"
	"bpl/repository"
	"fmt"
	"strings"
	"sync"
	"time"
)

type CharacterService struct {
	characterRepository *repository.CharacterRepository
	eventRepository     *repository.EventRepository
	teamRepository      *repository.TeamRepository
	userRepository      *repository.UserRepository
	activityRepository  *repository.ActivityRepository
	atlasService        *AtlasService
	itemService         *ItemService
	poeClient           *client.PoEClient
}

func NewCharacterService(poeClient *client.PoEClient) *CharacterService {
	return &CharacterService{
		characterRepository: repository.NewCharacterRepository(),
		eventRepository:     repository.NewEventRepository(),
		teamRepository:      repository.NewTeamRepository(),
		userRepository:      repository.NewUserRepository(),
		activityRepository:  repository.NewActivityRepository(),
		atlasService:        NewAtlasService(),
		itemService:         NewItemService(),
		poeClient:           poeClient,
	}
}

func (c *CharacterService) TrackActivity(eventId int, update *parser.PlayerUpdate) error {
	if update.New.Character.Experience != update.Old.Character.Experience {
		err := c.activityRepository.SaveActivity(&repository.Activity{
			Time:    time.Now(),
			UserId:  update.UserId,
			EventId: eventId,
		})
		if err != nil {
			fmt.Println("Error saving activity")
		}
	}

	return nil
}

func (c *CharacterService) GetCharactersForUser(user *repository.User) ([]*repository.Character, error) {
	return c.characterRepository.GetCharactersForUser(user)
}

func (c *CharacterService) GetCharactersForEvent(eventId int) ([]*repository.Character, error) {
	return c.characterRepository.GetCharactersForEvent(eventId)
}

func (c *CharacterService) GetCharacterById(characterId string) (*repository.Character, error) {
	return c.characterRepository.GetCharacterById(characterId)
}

func (c *CharacterService) GetCharacterHistory(characterId string) ([]*repository.CharacterPob, error) {
	return c.characterRepository.GetCharacterHistory(characterId)
}
func (c *CharacterService) GetCharacterStatsForEvent(eventId int, cutoff time.Time) (map[string]*repository.CharacterPob, error) {
	return c.characterRepository.GetCharacterStatsForEvent(eventId, cutoff)
}

func (c *CharacterService) GetTeamAtlasesForEvent(eventId int, userId int) ([]*repository.AtlasTree, error) {
	team, err := c.teamRepository.GetTeamForUser(eventId, userId)
	if err != nil {
		return []*repository.AtlasTree{}, nil
	}
	return c.atlasService.GetLatestAtlasesForEventAndTeam(eventId, team.TeamId)
}

func (c *CharacterService) GetPobForIdBeforeTimestamp(characterId string, timestamp time.Time) (*repository.CharacterPob, error) {
	pob, err := c.characterRepository.GetPobByCharacterIdBeforeTimestamp(characterId, timestamp)
	if err != nil {
		return nil, err
	}
	return pob, nil
}

func (c *CharacterService) GetPobs(characterId string) ([]*repository.CharacterPob, error) {
	pob, err := c.characterRepository.GetPobs(characterId)
	if err != nil {
		return nil, err
	}
	return pob, nil
}

func (c *CharacterService) UpdateCharacter(characterId string) (*client.Character, error) {
	character, err := c.characterRepository.GetCharacterById(characterId)
	if err != nil {
		return nil, err
	}
	if character.UserId == nil {
		return nil, fmt.Errorf("character has no user")
	}
	user, err := c.userRepository.GetUserById(*character.UserId, "OauthAccounts")
	if err != nil {
		return nil, err
	}
	if user.GetPoEToken() == "" {
		return nil, fmt.Errorf("user has no poe token")
	}
	response, clientErr := c.poeClient.GetCharacter(user.GetPoEToken(), character.Name, nil)
	if clientErr != nil {
		return nil, fmt.Errorf("%s", clientErr.Description)
	}
	return response.Character, nil
}

type CharacterInfo struct {
	User      *repository.User
	Event     *repository.Event
	Character *repository.Character
	TeamId    int
}

func (ci *CharacterInfo) ToPlayerUpdate() (*parser.PlayerUpdate, error) {
	for _, oauth := range ci.User.OauthAccounts {
		if oauth.Provider == repository.ProviderPoE && oauth.Expiry.After(time.Now()) {
			return &parser.PlayerUpdate{
				UserId:      ci.User.Id,
				TeamId:      ci.TeamId,
				AccountName: *ci.User.GetAccountName(repository.ProviderPoE),
				Token:       oauth.AccessToken,
				TokenExpiry: oauth.Expiry,
				Mu:          sync.Mutex{},
				New: parser.Player{
					Character: &client.Character{
						Id:    ci.Character.Id,
						Name:  ci.Character.Name,
						Class: ci.Character.Ascendancy,
						Level: ci.Character.Level,
					},
				},
				Old: parser.Player{},
				LastUpdateTimes: struct {
					CharacterName time.Time
					Character     time.Time
					LeagueAccount time.Time
					PoB           time.Time
				}{},
			}, nil
		}
	}
	return nil, fmt.Errorf("no valid PoE oauth token found for user")
}

func (c *CharacterService) GetInfoForCharacter(characterId string) (*CharacterInfo, error) {
	character, err := c.characterRepository.GetCharacterById(characterId)
	if err != nil {
		return nil, err
	}
	if character.UserId == nil {
		return nil, fmt.Errorf("character has no associtated user")
	}
	user, err := c.userRepository.GetUserById(*character.UserId, "OauthAccounts")
	if err != nil {
		return nil, err
	}
	teamUser, err := c.teamRepository.GetTeamForUser(character.EventId, *character.UserId)
	if err != nil {
		return nil, err
	}
	event, err := c.eventRepository.GetEventById(character.EventId)
	if err != nil {
		return nil, err
	}
	return &CharacterInfo{
		Character: character,
		User:      user,
		Event:     event,
		TeamId:    teamUser.TeamId,
	}, nil

}

func (c *CharacterService) UpdatePoB(pob *repository.CharacterPob) error {
	newExport, err := client.UpdatePoBExport(pob.Export.ToString())
	if err != nil {
		metrics.PobsCalculatedErrorCounter.Inc()
		return err
	}
	metrics.PobsCalculatedCounter.Inc()
	p := repository.PoBExport{}
	err = p.FromString(newExport)
	if err != nil {
		return err
	}
	pob.Export = p
	pob.UpdatedAt = time.Now()
	pobDecoded, err := pob.Export.Decode()
	if err == nil {
		pob.UpdateStats(pobDecoded)
	} else {
		fmt.Printf("Error decoding updated PoB for character %s: %v\n", pob.CharacterId, err)
	}
	return c.characterRepository.SavePoB(pob)
}

func (c *CharacterService) UpdateLatestPoBs() error {
	semaphore := make(chan struct{}, config.Env().NumberOfPoBReplicas)
	updateStart := time.Date(2026, 01, 29, 12, 0, 0, 0, time.Local)
	startId := 0

	for {
		fmt.Printf("Fetching PoBs starting from ID %d\n", startId)
		pobs, err := c.characterRepository.GetPobsFromIdWithLimit(startId+1, 100)

		if err != nil {
			fmt.Printf("Error getting PoBs from id %d: %v\n", startId, err)
			return err
		}
		if len(pobs) == 0 {
			break
		}
		for _, characterPob := range pobs {
			fmt.Printf("Processing PoB ID %d\n", characterPob.Id)
			startId = characterPob.Id
			if characterPob.UpdatedAt.After(updateStart) {
				continue
			}
			semaphore <- struct{}{}
			go func(characterPob *repository.CharacterPob) {
				defer func() { <-semaphore }() // Release the slot when done
				err := c.UpdatePoB(characterPob)
				if err != nil {
					fmt.Printf("Error updating PoB for character %s: %v\n", characterPob.CharacterId, err)
				}
			}(characterPob)
		}
	}
	return nil
}

func (c *CharacterService) UpdatePoBStats() error {
	startId := 0
	itemMap, err := c.itemService.GetItemMap()
	if err != nil {
		return err
	}

	for {
		pobs, err := c.characterRepository.GetPobsFromIdWithLimit(startId+1, 1000)

		if err != nil {
			fmt.Printf("Error getting PoBs from id %d: %v\n", startId, err)
			return err
		}
		if len(pobs) == 0 {
			break
		}
		for _, characterPob := range pobs {
			startId = characterPob.Id
			pob, err := characterPob.Export.Decode()
			if err != nil {
				fmt.Printf("Error decoding PoB for character %s: %v\n", characterPob.CharacterId, err)
				continue
			}
			itemIndexes := make(map[int]bool)
			for _, item := range pob.Items {
				if item.Rarity == "UNIQUE" {
					itemId, ok := itemMap["unique"][item.Name]
					if ok {
						itemIndexes[itemId] = true
					} else {
						savedItem, err := c.itemService.SaveItem(item.Name, "unique")
						if err != nil {
							fmt.Printf("Error saving unique item %s: %v\n", item.Name, err)
						} else {
							itemMap["unique"][item.Name] = savedItem.Id
							itemIndexes[savedItem.Id] = true
						}
					}
				}
			}
			for _, skillset := range pob.Skills.SkillSets {
				for _, skill := range skillset.Skills {
					for _, gem := range skill.Gems {
						name := gem.NameSpec
						if strings.Contains(gem.GemID, "Support") {
							name += " Support"
						}
						itemId, ok := itemMap["gem"][name]
						if ok {
							itemIndexes[itemId] = true
						} else {
							savedItem, err := c.itemService.SaveItem(name, "gem")
							if err != nil {
								fmt.Printf("Error saving gem item %s: %v\n", name, err)
							} else {
								itemMap["gem"][name] = savedItem.Id
								itemIndexes[savedItem.Id] = true
							}
						}
					}
				}
			}
			if len(itemIndexes) == 0 {
				continue
			}
			characterPob.Items = make([]int32, 0, len(itemIndexes))
			for itemId := range itemIndexes {
				characterPob.Items = append(characterPob.Items, int32(itemId))
			}
		}
		err = c.characterRepository.SavePoBs(pobs)
		if err != nil {
			fmt.Printf("Error saving PoBs: %v\n", err)
		}
		fmt.Printf("Updated PoBs up to ID %d\n", startId)
	}
	return nil
}
