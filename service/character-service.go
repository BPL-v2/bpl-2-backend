package service

import (
	"bpl/client"
	"bpl/parser"
	"bpl/repository"
	"fmt"
	"time"
)

type CharacterService struct {
	characterRepository *repository.CharacterRepository
	teamRepository      *repository.TeamRepository
	userRepository      *repository.UserRepository
	activityRepository  *repository.ActivityRepository
	atlasService        *AtlasService
	poeClient           *client.PoEClient
}

func NewCharacterService(poeClient *client.PoEClient) *CharacterService {
	return &CharacterService{
		characterRepository: repository.NewCharacterRepository(),
		teamRepository:      repository.NewTeamRepository(),
		userRepository:      repository.NewUserRepository(),
		activityRepository:  repository.NewActivityRepository(),
		atlasService:        NewAtlasService(),
		poeClient:           poeClient,
	}
}

func (c *CharacterService) TrackActivity(eventId int, update *parser.PlayerUpdate) error {
	if update.New.CharacterXP != update.Old.CharacterXP {
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

func (c *CharacterService) GetCharacterHistory(characterId string) ([]*repository.CharacterStat, error) {
	return c.characterRepository.GetCharacterHistory(characterId)
}
func (c *CharacterService) GetLatestCharacterStatsForEvent(eventId int) (map[string]*repository.CharacterStat, error) {
	return c.characterRepository.GetLatestCharacterStatsForEvent(eventId)
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
