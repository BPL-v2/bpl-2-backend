package cron

import (
	"bpl/client"
	"bpl/parser"
	"bpl/repository"
	"bpl/service"
	"bpl/utils"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

var (
	charQueue = make(chan *client.Character, 100)
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
	return time.Since(s.lastLadderUpdate) > 30*time.Second
}

func (s *PlayerFetchingService) UpdateCharacterName(player *parser.PlayerUpdate, event *repository.Event) {
	// fmt.Println("Updating character name for player", player.UserId)
	charactersResponse, err := s.client.ListCharacters(player.Token, event.GetRealm())
	player.Mu.Lock()
	defer player.Mu.Unlock()
	player.LastUpdateTimes.CharacterName = time.Now()
	if err != nil {
		player.SuccessiveErrors++
		if err.StatusCode == 401 || err.StatusCode == 403 {
			player.TokenExpiry = time.Now()
			return
		}
		log.Printf("Error fetching characters for player %d: %v", player.UserId, err)
		return
	}
	player.SuccessiveErrors = 0
	for _, char := range charactersResponse.Characters {
		if char.League != nil && *char.League == s.event.Name && char.Level > player.New.CharacterLevel {
			player.New.CharacterName = char.Name
			player.New.CharacterLevel = char.Level
			player.New.CharacterXP = char.Experience
			player.New.Ascendancy = char.Class
		}
	}
}

func (s *PlayerFetchingService) UpdateCharacter(player *parser.PlayerUpdate, event *repository.Event) {
	fmt.Println("Updating character", player.New.CharacterName)
	characterResponse, err := s.client.GetCharacter(player.Token, player.New.CharacterName, event.GetRealm())
	player.Mu.Lock()
	defer player.Mu.Unlock()
	player.LastUpdateTimes.Character = time.Now()
	if err != nil {
		player.SuccessiveErrors++
		if err.StatusCode == 401 || err.StatusCode == 403 {
			fmt.Printf("Error fetching character for player %d: %d", player.UserId, err.StatusCode)
			player.TokenExpiry = time.Now()
			return
		}
		if err.StatusCode == 404 {
			fmt.Printf("Character not found for player %d: %s", player.UserId, player.New.CharacterName)
			player.New.CharacterName = ""
			return
		}
		log.Printf("Error fetching character for player %d: %v", player.UserId, err)
		return
	}

	charQueue <- characterResponse.Character

	player.SuccessiveErrors = 0
	player.New.CharacterId = characterResponse.Character.Id
	player.New.CharacterLevel = characterResponse.Character.Level
	player.New.CharacterXP = characterResponse.Character.Experience
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
		player.SuccessiveErrors++
		if err.StatusCode == 401 || err.StatusCode == 403 {
			fmt.Printf("Error fetching league account for player %d: %d", player.UserId, err.StatusCode)
			player.TokenExpiry = time.Now()
			return
		}
		log.Print(err)
		return
	}
	player.SuccessiveErrors = 0
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
		token := os.Getenv("OLD_POE_CLIENT_TOKEN")
		resp, clientError = s.client.GetFullLadder(token, s.event.Name)
	}
	if clientError != nil {
		log.Printf("Error fetching ladder: %v", clientError)
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
		entriesToPersist = append(entriesToPersist, &entry)
		if player, ok := charToUpdate[entry.Character.Name]; ok {
			foundInLadder[entry.Character.Name] = true
			player.Mu.Lock()
			player.New.CharacterLevel = entry.Character.Level
			if entry.Character.Depth != nil && entry.Character.Depth.Default != nil {
				player.New.DelveDepth = *entry.Character.Depth.Default
			}
			player.Mu.Unlock()
		}
	}
	// for charName, player := range charToUpdate {
	// 	if _, ok := foundInLadder[charName]; !ok {
	// 		entriesToPersist = append(entriesToPersist, &client.LadderEntry{
	// 			Character: client.LadderEntryCharacter{
	// 				Name:       charName,
	// 				Level:      player.New.CharacterLevel,
	// 				Experience: &player.New.CharacterXP,
	// 				Class:      player.New.Ascendancy,
	// 			},
	// 			Rank:    0,
	// 			Account: &client.Account{Name: player.AccountName},
	// 		})
	// 	}
	// }
	err := s.ladderService.UpsertLadder(entriesToPersist, s.event.Id, charToUserId)
	if err != nil {
		log.Print(clientError)
	}
}

func (service *PlayerFetchingService) initPlayerUpdates() ([]*parser.PlayerUpdate, error) {
	users, err := service.userRepository.GetUsersForEvent(service.event.Id)
	if err != nil {
		return nil, err
	}
	players := utils.Map(users, func(user *repository.TeamUserWithPoEToken) *parser.PlayerUpdate {
		return &parser.PlayerUpdate{
			UserId:           user.UserId,
			TeamId:           user.TeamId,
			AccountName:      user.AccountName,
			Token:            user.Token,
			TokenExpiry:      user.TokenExpiry,
			SuccessiveErrors: 0,
			New:              parser.Player{},
			Old:              parser.Player{},
			Mu:               sync.Mutex{},
			LastUpdateTimes: struct {
				CharacterName time.Time
				Character     time.Time
				LeagueAccount time.Time
				PoB           time.Time
			}{},
		}
	})
	latestCharacters, err := service.characterService.GetLatestCharactersForEvent(service.event.Id)
	if err != nil {
		return nil, err
	}
	characterMap := make(map[int]*repository.Character, len(latestCharacters))
	for _, character := range latestCharacters {
		characterMap[character.UserId] = character
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

func (service *PlayerFetchingService) UpdatePlayerTokens(players []*parser.PlayerUpdate) []*parser.PlayerUpdate {
	users, err := service.userRepository.GetUsersForEvent(service.event.Id)
	usermap := make(map[int]*repository.TeamUserWithPoEToken, len(users))
	if err != nil {
		fmt.Printf("Error fetching users for event %d: %v", service.event.Id, err)
		return players
	}
	for _, player := range players {
		if user, ok := usermap[player.UserId]; ok {
			player.Token = user.Token
			player.TokenExpiry = user.TokenExpiry
		}
	}
	return players
}

type PlayerStatsCache struct {
	OldStats      *repository.CharacterStat
	OldPoBString  string
	LastPoBUpdate time.Time
}

func updateStats(character *client.Character, event *repository.Event, characterRepo *repository.CharacterRepository, cache map[string]*PlayerStatsCache, mu *sync.Mutex) {
	pob, export, err := client.GetPoBExport(character)
	if err != nil {
		fmt.Printf("Error fetching PoB export for character %s: %v\n", character.Name, err)
		// write local json with character data and filename character name
		file, err := os.Create(fmt.Sprintf("debug_pob_%s.json", character.Name))
		if err != nil {
			log.Printf("Error creating debug file for character %s: %v", character.Name, err)
			return
		}
		defer file.Close()
		data, err := json.MarshalIndent(character, "", "  ")
		if err != nil {
			log.Printf("Error marshalling character data for %s: %v", character.Name, err)
			return
		}
		_, err = file.Write(data)
		if err != nil {
			log.Printf("Error writing debug file for character %s: %v", character.Name, err)
			return
		}
		log.Printf("Debug file created for character %s", character.Name)
		return
	}
	stats := pob.Build.PlayerStats
	newStats := &repository.CharacterStat{
		Time:        time.Now(),
		EventId:     event.Id,
		CharacterId: character.Id,
		DPS:         int(stats.CombinedDPS),
		EHP:         int(stats.TotalEHP),
		PhysMaxHit:  int(stats.PhysicalMaximumHitTaken),
		EleMaxHit:   int(utils.Min(stats.FireMaximumHitTaken, stats.ColdMaximumHitTaken, stats.LightningMaximumHitTaken)),
		HP:          int(stats.Life),
		Mana:        int(stats.Mana),
		ES:          int(stats.EnergyShield),
		Armour:      int(stats.Armour),
		Evasion:     int(stats.Evasion),
		XP:          int(character.Experience),
	}
	mu.Lock()
	defer mu.Unlock()
	if cache[character.Name] == nil {
		cache[character.Name] = &PlayerStatsCache{
			OldStats: &repository.CharacterStat{},
		}
	}
	cacheItem := cache[character.Name]
	if time.Since(cacheItem.LastPoBUpdate) > 30*time.Minute && export != cacheItem.OldPoBString {
		cacheItem.OldPoBString = export
		cacheItem.LastPoBUpdate = time.Now()
		err := characterRepo.SavePoB(&repository.CharacterPob{
			CharacterId: character.Id,
			Level:       character.Level,
			MainSkill:   character.GetMainSkill(),
			Ascendancy:  character.Class,
			Export:      export,
			Timestamp:   time.Now(),
		})
		if err != nil {
			log.Printf("Error saving PoB for character %s: %v", character.Name, err)
		}
	}
	if !cacheItem.OldStats.IsEqual(newStats) {
		cacheItem.OldStats = newStats
		err := characterRepo.CreateCharacterStat(newStats)
		if err != nil {
			log.Printf("Error saving character stats for %s: %v", character.Name, err)
		}
	}
}

func InitCharacterStatsCache(eventId int, characterRepo *repository.CharacterRepository) map[string]*PlayerStatsCache {
	cache := make(map[string]*PlayerStatsCache)
	stats, err := characterRepo.GetLatestStatsForEvent(eventId)
	if err != nil {
		log.Printf("Error fetching latest stats for event %d: %v", eventId, err)
		return cache
	}
	pobs, err := characterRepo.GetLatestPoBsForEvent(eventId)
	if err != nil {
		log.Printf("Error fetching latest PoBs for event %d: %v", eventId, err)
		return cache
	}
	for _, stat := range stats {
		cache[stat.CharacterId] = &PlayerStatsCache{
			OldStats: stat,
		}
	}
	for _, pob := range pobs {
		if cache[pob.CharacterId] == nil {
			cache[pob.CharacterId] = &PlayerStatsCache{
				OldStats: &repository.CharacterStat{},
			}
		}
		cache[pob.CharacterId].OldPoBString = pob.Export
		cache[pob.CharacterId].LastPoBUpdate = pob.Timestamp
	}
	return cache

}

func PlayerStatsLoop(ctx context.Context, event *repository.Event) {
	characterRepo := repository.NewCharacterRepository()
	cache := InitCharacterStatsCache(event.Id, characterRepo)
	mu := sync.Mutex{}
	for {
		select {
		case <-ctx.Done():
			return
		default:
			character, ok := <-charQueue
			if !ok {
				log.Println("PoB queue closed, stopping player stats loop")
				return
			}
			// todo: make this a goroutine once the pob server is stable enough to handle concurrent requests
			go updateStats(character, event, characterRepo, cache, &mu)
		}
	}
}

func PlayerFetchLoop(ctx context.Context, event *repository.Event, poeClient *client.PoEClient) {
	service := NewPlayerFetchingService(poeClient, event)
	players, err := service.initPlayerUpdates()
	if err != nil {
		log.Print(err)
		return
	}
	objectives, err := service.objectiveService.GetObjectivesForEvent(service.event.Id)
	if err != nil {
		log.Print(err)
		return
	}
	playerChecker, err := parser.NewPlayerChecker(objectives)
	if err != nil {
		log.Print(err)
		return
	}
	teamChecker, err := parser.NewTeamChecker(objectives)
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
			for _, team := range event.Teams {
				teamPlayers := utils.Filter(players, func(player *parser.PlayerUpdate) bool {
					return player.TeamId == team.Id
				})
				teamMatches := service.GetTeamMatches(teamPlayers, teamChecker)
				matches = append(matches, teamMatches...)
			}
			service.objectiveMatchService.SaveMatches(matches, []int{})
			for _, player := range players {
				player.Old = player.New
			}
			players = service.UpdatePlayerTokens(players)
			time.Sleep(1 * time.Second)
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

func (m *PlayerFetchingService) GetTeamMatches(players []*parser.PlayerUpdate, teamChecker *parser.TeamChecker) []*repository.ObjectiveMatch {
	if len(players) == 0 {
		return []*repository.ObjectiveMatch{}
	}
	now := time.Now()
	return utils.Map(teamChecker.CheckForCompletions(players), func(result *parser.CheckResult) *repository.ObjectiveMatch {
		return &repository.ObjectiveMatch{
			ObjectiveId: result.ObjectiveId,
			UserId:      players[0].UserId,
			Number:      result.Number,
			Timestamp:   now,
			EventId:     m.event.Id,
		}
	})
}
