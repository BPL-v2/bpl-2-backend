package cron

import (
	"bpl/client"
	"bpl/parser"
	"bpl/repository"
	"bpl/service"
	"bpl/utils"
	"context"
	"log"
	"os"
	"sync"
	"time"
)

type PlayerFetchingService struct {
	userRepository        *repository.UserRepository
	objectiveMatchService *service.ObjectiveMatchService
	objectiveService      *service.ObjectiveService
	characterService      *service.CharacterService
	ladderService         *service.LadderService
	lastLadderUpdate      time.Time
	client                *client.PoEClient
	event                 *repository.Event
}

func NewPlayerFetchingService(client *client.PoEClient, event *repository.Event) *PlayerFetchingService {
	return &PlayerFetchingService{
		userRepository:        repository.NewUserRepository(),
		objectiveMatchService: service.NewObjectiveMatchService(),
		objectiveService:      service.NewObjectiveService(),
		ladderService:         service.NewLadderService(),
		characterService:      service.NewCharacterService(),
		lastLadderUpdate:      time.Now().Add(-1 * time.Hour),
		client:                client,
		event:                 event,
	}
}

func (s *PlayerFetchingService) shouldUpdateLadder() bool {
	return time.Since(s.lastLadderUpdate) > 5*time.Minute
}

func (s *PlayerFetchingService) UpdateCharacterName(playerUpdate *parser.PlayerUpdate, event *repository.Event) {
	charactersResponse, err := s.client.ListCharacters(playerUpdate.Token, event.GetRealm())
	playerUpdate.Mu.Lock()
	defer playerUpdate.Mu.Unlock()
	playerUpdate.LastUpdateTimes.CharacterName = time.Now()
	if err != nil {
		if err.StatusCode == 401 || err.StatusCode == 403 {
			playerUpdate.TokenExpiry = time.Now()
			return
		}
		log.Print(err)
		return
	}
	for _, char := range charactersResponse.Characters {
		if char.League != nil && *char.League == s.event.Name && char.Level > playerUpdate.New.CharacterLevel {
			playerUpdate.New.CharacterName = char.Name
			playerUpdate.New.CharacterLevel = char.Level
			playerUpdate.New.Ascendancy = char.Class
		}
	}
	log.Printf("Player %s updated: %s (%d)", playerUpdate.AccountName, playerUpdate.New.CharacterName, playerUpdate.New.CharacterLevel)
}

func (s *PlayerFetchingService) UpdateCharacter(player *parser.PlayerUpdate, event *repository.Event) {
	characterResponse, err := s.client.GetCharacter(player.Token, player.New.CharacterName, event.GetRealm())
	player.Mu.Lock()
	defer player.Mu.Unlock()
	player.LastUpdateTimes.Character = time.Now()
	if err != nil {
		if err.StatusCode == 401 || err.StatusCode == 403 {
			player.TokenExpiry = time.Now()
			return
		}
		if err.StatusCode == 404 {
			player.New.CharacterName = ""
			return
		}
		log.Print(err)
		return
	}
	player.New.CharacterLevel = characterResponse.Character.Level
	player.New.Ascendancy = characterResponse.Character.Class
	player.New.Pantheon = characterResponse.Character.HasPantheon()
	player.New.AscendancyPoints = characterResponse.Character.GetAscendancyPoints()
	player.New.MainSkill = characterResponse.Character.GetMainSkill()
}

func (s *PlayerFetchingService) UpdateLeagueAccount(player *parser.PlayerUpdate) {
	if s.event.GameVersion == repository.PoE2 {
		return
	}
	leagueAccount, err := s.client.GetLeagueAccount(player.Token, s.event.Name)
	player.Mu.Lock()
	defer player.Mu.Unlock()
	player.LastUpdateTimes.LeagueAccount = time.Now()
	if err != nil {
		if err.StatusCode == 401 || err.StatusCode == 403 {
			player.TokenExpiry = time.Now()
			return
		}
		log.Print(err)
		return
	}
	player.New.AtlasPassiveTrees = leagueAccount.LeagueAccount.AtlasPassiveTrees
}

func (s *PlayerFetchingService) UpdateLadder(players []*parser.PlayerUpdate) {
	if !s.shouldUpdateLadder() {
		return
	}
	s.lastLadderUpdate = time.Now()
	var resp *client.GetLeagueLadderResponse
	var clientError *client.ClientError
	if s.event.GameVersion == repository.PoE2 {
		// todo: get the ladder for the correct event
		resp, clientError = s.client.GetPoE2Ladder(s.event.Name)
	} else {
		// todo: once we have a token that allows us to request the ladder api
		return
		token := os.Getenv("POE_CLIENT_TOKEN")
		resp, clientError = s.client.GetFullLadder(token, s.event.Name)
	}
	if clientError != nil {
		log.Print(clientError)
		return
	}

	charToUpdate := map[string]*parser.PlayerUpdate{}
	foundInLadder := make(map[string]bool)
	charToUserId := map[string]int{}
	for _, player := range players {
		charToUpdate[player.New.CharacterName] = player
		charToUserId[player.New.CharacterName] = player.UserId
	}

	entriesToPersist := make([]*client.LadderEntry, 0, len(resp.Ladder.Entries))
	for _, entry := range resp.Ladder.Entries {
		if player, ok := charToUpdate[entry.Character.Name]; ok {
			foundInLadder[entry.Character.Name] = true
			entriesToPersist = append(entriesToPersist, &entry)
			player.Mu.Lock()
			player.New.CharacterLevel = entry.Character.Level
			if entry.Character.Depth != nil && entry.Character.Depth.Depth != nil {
				player.New.DelveDepth = *entry.Character.Depth.Depth
			}
			player.Mu.Unlock()
		}
	}
	for charName, player := range charToUpdate {
		if _, ok := foundInLadder[charName]; !ok {
			entriesToPersist = append(entriesToPersist, &client.LadderEntry{
				Character: client.LadderEntryCharacter{
					Name:  charName,
					Level: player.New.CharacterLevel,
					Class: player.New.Ascendancy,
				},
				Rank:    0,
				Account: &client.Account{Name: player.AccountName},
			})
		}
	}
	err := s.ladderService.UpsertLadder(entriesToPersist, s.event.Id, charToUserId)
	if err != nil {
		log.Print(clientError)
	}
}

func (service *PlayerFetchingService) initPlayerUpdates() ([]*parser.PlayerUpdate, error) {
	users, err := service.userRepository.GetAuthenticatedUsersForEvent(service.event.Id)
	if err != nil {
		return nil, err
	}
	players := utils.Map(users, func(user *repository.TeamUserWithPoEToken) *parser.PlayerUpdate {
		return &parser.PlayerUpdate{
			UserId:      user.UserId,
			TeamId:      user.TeamId,
			AccountName: user.AccountName,
			Token:       user.Token,
			TokenExpiry: user.TokenExpiry,
			New:         parser.Player{},
			Old:         parser.Player{},
			Mu:          sync.Mutex{},
			LastUpdateTimes: struct {
				CharacterName time.Time
				Character     time.Time
				LeagueAccount time.Time
			}{},
		}
	})
	latestCharacters, err := service.characterService.GetLatestCharactersForEvent(service.event.Id)
	if err != nil {
		return nil, err
	}
	characterMap := make(map[int]*repository.Character, len(latestCharacters))
	for _, character := range latestCharacters {
		characterMap[character.UserID] = character
	}

	for _, player := range players {
		if character, ok := characterMap[player.UserId]; ok {
			player.Old.CharacterName = character.Name
			player.Old.CharacterLevel = character.Level
			player.Old.MainSkill = character.MainSkill
			player.Old.Pantheon = character.Pantheon
			player.Old.Ascendancy = character.Ascendancy
			player.Old.AscendancyPoints = character.AscendancyPoints
		}
	}
	return players, nil
}

func PlayerFetchLoop(ctx context.Context, event *repository.Event, poeClient *client.PoEClient) {
	service := NewPlayerFetchingService(poeClient, event)
	players, err := service.initPlayerUpdates()
	if err != nil {
		log.Print(err)
		return
	}
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
			wg := sync.WaitGroup{}
			// handle character name updates first
			for _, player := range players {
				if player.TokenExpiry.Before(time.Now()) {
					continue
				}
				if player.ShouldUpdateCharacterName() {
					wg.Add(1)
					go func(player *parser.PlayerUpdate) {
						defer wg.Done()
						service.UpdateCharacterName(player, event)
					}(player)
				}
			}
			wg.Wait()
			wg = sync.WaitGroup{}
			for _, player := range players {
				if player.TokenExpiry.Before(time.Now()) {
					continue
				}
				if player.ShouldUpdateCharacter() {
					wg.Add(1)
					go func(player *parser.PlayerUpdate) {
						defer wg.Done()
						service.UpdateCharacter(player, event)
					}(player)
				}
				if player.ShouldUpdateLeagueAccount() {
					wg.Add(1)
					go func(player *parser.PlayerUpdate) {
						defer wg.Done()
						service.UpdateLeagueAccount(player)
					}(player)
				}
			}
			if service.shouldUpdateLadder() {
				wg.Add(1)
				go func() {
					defer wg.Done()
					service.UpdateLadder(players)
				}()
			}
			wg.Wait()

			for _, player := range players {
				err := service.characterService.SavePlayerUpdate(event.Id, player)
				if err != nil {
					log.Print(err)
				}
			}

			matches := utils.FlatMap(players, func(player *parser.PlayerUpdate) []*repository.ObjectiveMatch {
				return service.GetPlayerMatches(player, playerChecker)
			})
			service.objectiveMatchService.SaveMatches(matches, []int{})
			for _, player := range players {
				player.Old = player.New
			}
			time.Sleep(10 * time.Second)
		}
	}
}

func (m *PlayerFetchingService) GetPlayerMatches(player *parser.PlayerUpdate, playerChecker *parser.PlayerChecker) []*repository.ObjectiveMatch {
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
