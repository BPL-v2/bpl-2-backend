package service

import (
	"bpl/parser"
	"bpl/repository"
	"fmt"
	"time"
)

type CharacterService struct {
	repository     *repository.CharacterRepository
	teamRepository *repository.TeamRepository
}

func NewCharacterService() *CharacterService {
	return &CharacterService{
		repository:     repository.NewCharacterRepository(),
		teamRepository: repository.NewTeamRepository(),
	}
}

func (c *CharacterService) SavePlayerUpdate(eventId int, update *parser.PlayerUpdate) error {
	if len(update.New.AtlasPassiveTrees) > 0 {
		err := c.repository.SaveAtlasTrees(update.UserId, eventId, update.New.AtlasPassiveTrees)
		if err != nil {
			fmt.Println("Error saving atlas trees")
			return err
		}
	}

	if update.New.CharacterName != update.Old.CharacterName ||
		update.New.CharacterLevel != update.Old.CharacterLevel ||
		update.New.MainSkill != update.Old.MainSkill ||
		update.New.Pantheon != update.Old.Pantheon ||
		update.New.AscendancyPoints != update.Old.AscendancyPoints ||
		update.New.Ascendancy != update.Old.Ascendancy ||
		update.New.MaxAtlasTreeNodes() != update.Old.MaxAtlasTreeNodes() {

		character := &repository.Character{
			UserID:           update.UserId,
			EventID:          eventId,
			Name:             update.New.CharacterName,
			Level:            update.New.CharacterLevel,
			MainSkill:        update.New.MainSkill,
			Ascendancy:       update.New.Ascendancy,
			AscendancyPoints: update.New.AscendancyPoints,
			Pantheon:         update.New.Pantheon,
			AtlasNodeCount:   update.New.MaxAtlasTreeNodes(),
			Timestamp:        time.Now(),
		}
		err := c.repository.CreateCharacterCheckpoint(character)
		if err != nil {
			fmt.Println("Error saving character")
			return err
		}

	}
	return nil
}

func (c *CharacterService) GetLatestEventCharactersForUser(userId int) ([]*repository.Character, error) {
	return c.repository.GetLatestEventCharactersForUser(userId)
}

func (c *CharacterService) GetLatestCharactersForEvent(eventId int) ([]*repository.Character, error) {
	return c.repository.GetLatestCharactersForEvent(eventId)
}

func (c *CharacterService) GetEventCharacterHistoryForUser(userId int, eventId int) ([]*repository.Character, error) {
	return c.repository.GetEventCharacterHistoryForUser(userId, eventId)
}

func (c *CharacterService) GetTeamAtlasesForEvent(eventId int, userId int) ([]*repository.Atlas, error) {
	team, err := c.teamRepository.GetTeamForUser(eventId, userId)
	if err != nil {
		return []*repository.Atlas{}, nil
	}
	return c.repository.GetTeamAtlasesForEvent(eventId, team.Id)
}
