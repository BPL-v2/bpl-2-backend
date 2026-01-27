package cron

import (
	"bpl/client"
	"bpl/metrics"
	"bpl/parser"
	"bpl/repository"
	"bpl/service"
	"bpl/utils"
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

type EventChar struct {
	EventId   int
	Character *client.Character
}

var (
	charQueue = make(chan EventChar, 2000)
	statQueue = make(chan *repository.CharacterStat, 2000)
)

type PlayerFetchingService struct {
	userRepository        *repository.UserRepository
	objectiveMatchService *service.ObjectiveMatchService
	objectiveService      *service.ObjectiveService
	characterService      *service.CharacterService
	ladderService         *service.LadderService
	atlasService          *service.AtlasService
	oauthService          *service.OauthService
	timingRepository      *repository.TimingRepository
	characterRepository   *repository.CharacterRepository
	activityRepository    *repository.ActivityRepository
	itemWishService       *service.ItemWishService
	timings               map[repository.TimingKey]time.Duration

	lastLadderUpdate time.Time
	poeClient        *client.PoEClient
}

func (s *PlayerFetchingService) ReloadTimings() error {
	timings, err := s.timingRepository.GetTimings()
	if err != nil {
		return err
	}
	s.timings = timings
	return nil
}

func NewPlayerFetchingService(poeClient *client.PoEClient) *PlayerFetchingService {
	return &PlayerFetchingService{
		userRepository:        repository.NewUserRepository(),
		objectiveMatchService: service.NewObjectiveMatchService(),
		objectiveService:      service.NewObjectiveService(),
		ladderService:         service.NewLadderService(),
		characterService:      service.NewCharacterService(poeClient),
		atlasService:          service.NewAtlasService(),
		oauthService:          service.NewOauthService(),
		itemWishService:       service.NewItemWishService(),
		timingRepository:      repository.NewTimingRepository(),
		characterRepository:   repository.NewCharacterRepository(),
		activityRepository:    repository.NewActivityRepository(),
		lastLadderUpdate:      time.Now().Add(-1 * time.Hour),
		poeClient:             poeClient,
	}
}

func (s *PlayerFetchingService) shouldUpdateLadder(timings map[repository.TimingKey]time.Duration) bool {
	return time.Since(s.lastLadderUpdate) > timings[repository.LadderUpdateInterval]
}

func (s *PlayerFetchingService) UpdateCharacterName(player *parser.PlayerUpdate, event *repository.Event) {
	charactersResponse, err := s.poeClient.ListCharacters(player.Token, event.GetRealm())
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
		if char.League != nil && *char.League == event.Name && char.Level > player.New.Character.Level {
			player.New.Character.Name = char.Name
			player.New.Character.Level = char.Level
			player.New.Character.Experience = char.Experience
			player.New.Character.Class = char.Class
		}
	}
}

func (s *PlayerFetchingService) UpdateCharacter(player *parser.PlayerUpdate, event *repository.Event) (*client.Character, error) {
	fmt.Println("Updating character", player.New.Character.Name)
	characterResponse, clientError := s.poeClient.GetCharacter(player.Token, player.New.Character.Name, event.GetRealm())
	player.Mu.Lock()
	defer player.Mu.Unlock()
	player.LastUpdateTimes.Character = time.Now()
	if clientError != nil {
		player.SuccessiveErrors++
		if clientError.StatusCode == 401 || clientError.StatusCode == 403 {
			player.TokenExpiry = time.Now()
			return nil, fmt.Errorf("error fetching character for player %d: %d", player.UserId, clientError.StatusCode)
		}
		if clientError.StatusCode == 404 {
			player.New.Character.Name = ""
			return nil, fmt.Errorf("character not found for player %d: %s", player.UserId, player.New.Character.Name)
		}
		return nil, fmt.Errorf("error fetching character for player %d: %v", player.UserId, clientError)
	}
	err := s.itemWishService.UpdateItemWishFulfillment(player.TeamId, player.UserId, characterResponse.Character)
	if err != nil {
		log.Printf("Error updating item wish fulfillment for player %d: %v", player.UserId, err)
	}
	player.SuccessiveErrors = 0
	player.New.Character = characterResponse.Character
	if player.Old.Character == nil || (player.New.Character.EquipmentHash() != player.Old.Character.EquipmentHash()) {
		log.Printf("Character equipment changed for player %d, queuing for PoB processing", player.UserId)
		charQueue <- EventChar{EventId: event.Id, Character: characterResponse.Character}
		player.LastUpdateTimes.PoB = time.Now()
	}
	character := &repository.Character{
		Id:               player.New.Character.Id,
		UserId:           &player.UserId,
		EventId:          event.Id,
		Name:             player.New.Character.Name,
		Level:            player.New.Character.Level,
		MainSkill:        player.New.Character.GetMainSkill(),
		Ascendancy:       player.New.Character.Class,
		AscendancyPoints: player.New.Character.GetAscendancyPoints(),
		AtlasPoints:      player.New.MaxAtlasTreeNodes(),
	}
	err = s.characterRepository.Save(character)
	if err != nil {
		return nil, fmt.Errorf("Error saving character %s (%s) for user %d: %v\n", character.Name, character.Id, player.UserId, err)
	}
	return characterResponse.Character, nil
}

func (s *PlayerFetchingService) UpdateLeagueAccount(player *parser.PlayerUpdate, event *repository.Event) {
	if event.GameVersion == repository.PoE2 {
		return
	}
	leagueAccount, err := s.poeClient.GetLeagueAccount(player.Token, event.Name)
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
	if len(player.New.AtlasPassiveTrees) > 0 {
		err := s.atlasService.SaveAtlasTrees(player.UserId, event.Id, player.New.AtlasPassiveTrees)
		if err != nil {
			fmt.Printf("Error saving atlas trees %d: %v\n", player.UserId, err)
		}
	}

}

func (s *PlayerFetchingService) UpdateLadder(players []*parser.PlayerUpdate, event *repository.Event) {
	if !s.shouldUpdateLadder(s.timings) {
		return
	}
	s.lastLadderUpdate = time.Now()
	var resp *client.GetLeagueLadderResponse
	var clientError *client.ClientError
	if event.GameVersion == repository.PoE2 {
		// todo: get the ladder for the correct event
		resp, clientError = s.poeClient.GetPoE2Ladder(event.Name)
	} else {
		token, err := s.oauthService.GetApplicationToken(repository.ProviderPoE)
		if err != nil {
			log.Printf("Error fetching application token: %v", err)
			return
		}
		resp, clientError = s.poeClient.GetFullLadder(token, event.Name)
	}
	if clientError != nil {
		log.Printf("Error fetching ladder: %v", clientError)
		return
	}

	charToUpdate := map[string]*parser.PlayerUpdate{}
	foundInLadder := make(map[string]bool)
	charToUserId := map[string]int{}
	for _, player := range players {
		charToUpdate[player.New.Character.Name] = player
		charToUserId[player.New.Character.Name] = player.UserId
	}

	entriesToPersist := make([]*client.LadderEntry, 0, len(resp.Ladder.Entries))
	for _, entry := range resp.Ladder.Entries {
		entriesToPersist = append(entriesToPersist, &entry)
		if player, ok := charToUpdate[entry.Character.Name]; ok {
			player.Mu.Lock()
			foundInLadder[entry.Character.Name] = true
			player.New.Character.Level = entry.Character.Level
			if entry.Character.Depth != nil && entry.Character.Depth.Default != nil {
				player.New.DelveDepth = *entry.Character.Depth.Default
			}
			if entry.Character.Experience != nil {
				player.New.Character.Experience = *entry.Character.Experience
			}
			player.Mu.Unlock()
		}
	}
	err := s.ladderService.UpsertLadder(entriesToPersist, event.Id, charToUserId)
	if err != nil {
		log.Print(clientError)
	}
}

func (service *PlayerFetchingService) initPlayerUpdates(event *repository.Event) ([]*parser.PlayerUpdate, error) {
	users, err := service.userRepository.GetUsersForEvent(event.Id)
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
			New: parser.Player{
				Character: &client.Character{},
				Stats:     &repository.CharacterStat{},
			},
			Old: parser.Player{
				Character: &client.Character{},
				Stats:     &repository.CharacterStat{},
			},
			Mu: sync.Mutex{},
			LastUpdateTimes: struct {
				CharacterName time.Time
				Character     time.Time
				LeagueAccount time.Time
				PoB           time.Time
			}{},
		}
	})
	latestCharacters, err := service.characterService.GetCharactersForEvent(event.Id)
	if err != nil {
		return nil, err
	}
	characterMap := make(map[int]*repository.Character, len(latestCharacters))
	for _, character := range latestCharacters {
		characterMap[*character.UserId] = character
	}

	for _, player := range players {
		if character, ok := characterMap[player.UserId]; ok {
			player.New.Character.Name = character.Name
			player.Old.Character.Name = character.Name
			player.New.Character.Id = character.Id
			player.Old.Character.Id = character.Id
			player.New.Character.Level = character.Level
			player.Old.Character.Level = character.Level
			player.New.Character.Class = character.Ascendancy
			player.Old.Character.Class = character.Ascendancy
			player.LastUpdateTimes.CharacterName = time.Now()

		}
	}
	return players, nil
}

func (service *PlayerFetchingService) UpdatePlayerTokens(players []*parser.PlayerUpdate, event *repository.Event) []*parser.PlayerUpdate {
	users, err := service.userRepository.GetUsersForEvent(event.Id)
	usermap := make(map[int]*repository.TeamUserWithPoEToken, len(users))
	if err != nil {
		fmt.Printf("Error fetching users for event %d: %v", event.Id, err)
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
	OldStats       *repository.CharacterStat
	OldPoBString   string
	LastPoBUpdate  time.Time
	NumFilledSlots int
}

func float2Int64(f float64) int64 {
	if f < 0 {
		return -float2Int64(-f) // handle negative values
	}
	if f > float64(int(^uint(0)>>1)) {
		return int64(^uint(0) >> 1) // max int value
	}
	return int64(f)
}

func float2Int32(f float64) int32 {
	if f < 0 {
		return -float2Int32(-f) // handle negative values
	}
	if f > float64(int32(^uint32(0)>>1)) {
		return int32(^uint32(0) >> 1) // max int32 value
	}
	return int32(f)
}

func updateStats(character *client.Character, eventId int, characterRepo *repository.CharacterRepository) {
	pob, export, err := client.GetPoBExport(character)
	if err != nil {
		fmt.Printf("Error fetching PoB export for character %s: %v\n", character.Name, err)
		return
	}
	metrics.PobsCalculatedCounter.Inc()
	stats := pob.Build.PlayerStats
	newStats := &repository.CharacterStat{
		Time:          time.Now(),
		EventId:       eventId,
		CharacterId:   character.Id,
		DPS:           float2Int64(utils.Max(stats.CombinedDPS, stats.CullingDPS, stats.FullDPS, stats.FullDotDPS, stats.PoisonDPS, stats.ReservationDPS, stats.TotalDPS, stats.TotalDotDPS, stats.WithBleedDPS, stats.WithIgniteDPS, stats.WithPoisonDPS)),
		EHP:           float2Int32(stats.TotalEHP),
		PhysMaxHit:    float2Int32(stats.PhysicalMaximumHitTaken),
		EleMaxHit:     float2Int32(utils.Min(stats.FireMaximumHitTaken, stats.ColdMaximumHitTaken, stats.LightningMaximumHitTaken)),
		HP:            float2Int32(stats.Life),
		Mana:          float2Int32(stats.Mana),
		ES:            float2Int32(stats.EnergyShield),
		Armour:        float2Int32(stats.Armour),
		Evasion:       float2Int32(stats.Evasion),
		XP:            int64(character.Experience),
		MovementSpeed: float2Int32(stats.EffectiveMovementSpeedMod * 100),
	}
	statQueue <- newStats
	oldStats, _ := characterRepo.GetLatestCharacterStats(character.Id)
	if newStats.IsEqual(oldStats) {
		log.Printf("No changes in stats for character %s, skipping save", character.Name)
		return
	}
	p := repository.PoBExport{}
	p.FromString(export)
	err = characterRepo.SavePoB(&repository.CharacterPob{
		CharacterId: character.Id,
		Level:       character.Level,
		MainSkill:   character.GetMainSkill(),
		Ascendancy:  character.Class,
		Export:      p,
		CreatedAt:   time.Now(),
	})
	if err != nil {
		log.Printf("Error saving PoB for character %s: %v", character.Name, err)
	}
	metrics.PobsSavedCounter.Inc()
	err = characterRepo.CreateCharacterStat(newStats)
	if err != nil {
		log.Printf("Error saving character stats for %s: %v", character.Name, err)
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
		cache[pob.CharacterId].OldPoBString = pob.Export.ToString()
		cache[pob.CharacterId].LastPoBUpdate = pob.CreatedAt
	}
	return cache

}

func PlayerStatsLoop(ctx context.Context) {
	characterRepo := repository.NewCharacterRepository()
	// make sure that only 2 goroutines are running at the same time
	semaphore := make(chan struct{}, 2)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			eventCharacter, ok := <-charQueue
			metrics.PobQueueGauge.Set(float64(len(charQueue)))
			if !ok {
				log.Println("PoB queue closed, stopping player stats loop")
				return
			}
			semaphore <- struct{}{}
			go func(eventCharacter EventChar) {
				defer func() { <-semaphore }() // Release the slot when done
				updateStats(eventCharacter.Character, eventCharacter.EventId, characterRepo)
			}(eventCharacter)
		}
	}
}
func drainStatQueue() map[string]*repository.CharacterStat {
	statMap := make(map[string]*repository.CharacterStat)
	for {
		select {
		case stat := <-statQueue:
			statMap[stat.CharacterId] = stat
		default:
			return statMap
		}
	}
}

func PlayerFetchLoop(ctx context.Context, event *repository.Event, poeClient *client.PoEClient) {
	service := NewPlayerFetchingService(poeClient)
	players, err := service.initPlayerUpdates(event)
	if err != nil {
		log.Print(err)
		return
	}
	objectives, err := service.objectiveService.GetObjectivesForEvent(event.Id)
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
	fmt.Printf("Starting PlayerFetchLoop for event: %s with %d players\n", event.Name, len(players))
	for {
		select {
		case <-ctx.Done():
			return
		default:
			err := service.ReloadTimings()
			if err != nil {
				log.Print(err)
				return
			}
			wg := sync.WaitGroup{}
			for _, player := range players {
				if player.ShouldUpdateCharacterName(service.timings) {
					wg.Add(1)
					go func(player *parser.PlayerUpdate) {
						defer wg.Done()
						service.UpdateCharacterName(player, event)
					}(player)
				}
				if player.ShouldUpdateCharacter(service.timings) {
					wg.Add(1)
					go func(player *parser.PlayerUpdate) {
						defer wg.Done()
						_, err := service.UpdateCharacter(player, event)
						if err != nil {
							fmt.Printf("Error updating character for player %d: %v\n", player.UserId, err)
						}
					}(player)
				}
				if player.ShouldUpdateLeagueAccount(service.timings) {
					wg.Add(1)
					go func(player *parser.PlayerUpdate) {
						defer wg.Done()
						service.UpdateLeagueAccount(player, event)
					}(player)
				}
			}
			if service.shouldUpdateLadder(service.timings) {
				wg.Go(func() {
					service.UpdateLadder(players, event)
				})
			}
			wg.Wait()

			statMap := drainStatQueue()
			for _, player := range players {
				player.Mu.Lock()
				player.New.Stats = statMap[player.New.Character.Id]
				if player.New.Character.Experience != player.Old.Character.Experience {
					player.LastActive = time.Now()
					err = service.activityRepository.SaveActivity(&repository.Activity{
						Time:    time.Now(),
						UserId:  player.UserId,
						EventId: event.Id,
					})
					if err != nil {
						fmt.Printf("Error saving activity for player %d: %v\n", player.UserId, err)
					}
				}
				player.Mu.Unlock()
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
			err = service.objectiveMatchService.SaveMatches(matches, []int{})
			if err != nil {
				log.Print(err)
			}
			for _, player := range players {
				player.Old = player.New
			}
			players = service.UpdatePlayerTokens(players, event)
			time.Sleep(1 * time.Second)
		}
	}
}

func (m *PlayerFetchingService) GetPlayerMatches(player *parser.PlayerUpdate, playerChecker *parser.PlayerChecker) []*repository.ObjectiveMatch {
	return utils.Map(playerChecker.CheckForCompletions(player), func(result *parser.CheckResult) *repository.ObjectiveMatch {
		return &repository.ObjectiveMatch{
			ObjectiveId: result.ObjectiveId,
			UserId:      &player.UserId,
			Number:      result.Number,
			Timestamp:   time.Now(),
			TeamId:      player.TeamId,
		}
	})
}

func (m *PlayerFetchingService) GetTeamMatches(players []*parser.PlayerUpdate, teamChecker *parser.TeamChecker) []*repository.ObjectiveMatch {
	matches := []*repository.ObjectiveMatch{}
	if len(players) == 0 {
		return matches
	}
	now := time.Now()
	for _, result := range teamChecker.CheckForCompletions(players) {
		if result.Number > 0 {
			matches = append(matches, &repository.ObjectiveMatch{
				ObjectiveId: result.ObjectiveId,
				Number:      result.Number,
				Timestamp:   now,
				TeamId:      players[0].TeamId,
			})
		}
	}
	return matches
}
