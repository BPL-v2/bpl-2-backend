package cron

import (
	"bpl/client"
	"bpl/parser"
	"bpl/repository"
	"bpl/service"
	"bpl/utils"
	"context"
	"fmt"
	"log"
	"time"
)

type PlayerFetchingService struct {
	userRepository        *repository.UserRepository
	objectiveMatchService *service.ObjectiveMatchService
	objectiveService      *service.ObjectiveService

	client *client.PoEClient
	event  *repository.Event
}

func NewPlayerFetchingService(client *client.PoEClient, event *repository.Event) *PlayerFetchingService {
	return &PlayerFetchingService{
		userRepository:        repository.NewUserRepository(),
		objectiveMatchService: service.NewObjectiveMatchService(),
		objectiveService:      service.NewObjectiveService(),
		client:                client,
		event:                 event,
	}
}

func (s *PlayerFetchingService) UpdateCharacterName(player *parser.Player) {
	player.Mu.Lock()
	defer player.Mu.Unlock()
	if !player.ShouldUpdateCharacterName() {
		return
	}
	charactersResponse, err := s.client.ListCharacters(player.Token)
	player.LastUpdateTimes.CharacterName = time.Now()
	if err != nil {
		if err.StatusCode == 401 || err.StatusCode == 403 {
			player.RemovePlayer = true
			return
		}
		log.Print(err)
		return
	}
	characterLevel := 0
	for _, char := range charactersResponse.Characters {
		if char.League != nil && *char.League == s.event.Name && char.Level > characterLevel {
			characterLevel = char.Level
			player.CharacterName = &char.Name
		}
	}
}

func (s *PlayerFetchingService) UpdateCharacter(player *parser.Player) {
	player.Mu.Lock()
	defer player.Mu.Unlock()
	if !player.ShouldUpdateCharacter() {
		return
	}
	characterResponse, err := s.client.GetCharacter(player.Token, *player.CharacterName)
	player.LastUpdateTimes.Character = time.Now()
	if err != nil {
		if err.StatusCode == 401 || err.StatusCode == 403 {
			player.RemovePlayer = true
			return
		}
		if err.StatusCode == 404 {
			player.CharacterName = nil
			return
		}
		log.Print(err)
		return
	}

	player.Character = characterResponse.Character
}

func (s *PlayerFetchingService) UpdateLeagueAccount(player *parser.Player) {
	player.Mu.Lock()
	defer player.Mu.Unlock()
	if !player.ShouldUpdateLeagueAccount() {
		return
	}
	leagueAccount, err := s.client.GetLeagueAccount(player.Token, s.event.Name)
	player.LastUpdateTimes.LeagueAccount = time.Now()
	if err != nil {
		if err.StatusCode == 401 || err.StatusCode == 403 {
			player.RemovePlayer = true
			return
		}
		log.Print(err)
		return
	}
	player.LeagueAccount = &leagueAccount.LeagueAccount
}
func PlayerFetchLoop(ctx context.Context, event *repository.Event, poeClient *client.PoEClient) {
	service := NewPlayerFetchingService(poeClient, event)
	users, err := service.userRepository.GetAuthenticatedUsersForEvent(service.event.Id)
	if err != nil {
		log.Print(err)
		return
	}
	players := utils.Map(users, func(user *repository.TeamUserWithPoEToken) *parser.Player {
		return &parser.Player{
			UserId: user.UserId,
			TeamId: user.TeamId,
			Token:  user.Token,
		}
	})
	objectives, err := service.objectiveService.GetObjectivesByEventId(service.event.Id)
	if err != nil {
		log.Print(err)
		return
	}
	playerChecker, err := parser.NewPlayerChecker(objectives)
	if err != nil {
		log.Print(err)
		return
	}
	for {
		select {
		case <-ctx.Done():
			return
		default:
			for _, player := range players {
				fmt.Println("Updating player", player.UserId)
				service.UpdateCharacterName(player)
				fmt.Println("Updated character name", player.UserId)
				service.UpdateCharacter(player)
				fmt.Println("Updated character", player.UserId)
				service.UpdateLeagueAccount(player)
				fmt.Println("Updated league account", player.UserId)
			}
			players = utils.Filter(players, func(player *parser.Player) bool {
				return !player.RemovePlayer
			})
			matches := utils.FlatMap(players, func(player *parser.Player) []*repository.ObjectiveMatch {
				return service.GetPlayerMatches(player, playerChecker)
			})
			fmt.Println("Saving matches")
			fmt.Println(matches)
			service.objectiveMatchService.SaveMatches(matches, []int{})
			time.Sleep(1 * time.Minute)

		}
	}
}

func (m *PlayerFetchingService) GetPlayerMatches(player *parser.Player, playerChecker *parser.PlayerChecker) []*repository.ObjectiveMatch {
	return utils.Map(playerChecker.CheckForCompletions(player), func(result *parser.CheckResult) *repository.ObjectiveMatch {
		return &repository.ObjectiveMatch{
			ObjectiveId: result.ObjectiveId,
			UserId:      player.UserId,
			Number:      result.Number,
			Timestamp:   time.Now(),
			EventId:     m.event.Id,
		}
	})
}
