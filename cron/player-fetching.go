package cron

import (
	"bpl/client"
	"bpl/config"
	"bpl/parser"
	"bpl/repository"
	"bpl/service"
	"bpl/utils"
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	charQueue = make(chan *client.Character, 2000)
)
var pobQueueGauge = promauto.NewGauge(
	prometheus.GaugeOpts{
		Name: "bpl_pob_queue_size",
		Help: "Current size of the character queue to be processed by the pob server",
	},
)

var pobsCalculatedCounter = promauto.NewCounter(
	prometheus.CounterOpts{
		Name: "bpl_pobs_calculated",
		Help: "Number of PoBs calculated",
	},
)
var pobsSavedCounter = promauto.NewCounter(
	prometheus.CounterOpts{
		Name: "bpl_pobs_saved",
		Help: "Number of PoBs saved to the database",
	},
)

type PlayerFetchingService struct {
	userRepository        *repository.UserRepository
	objectiveMatchService *service.ObjectiveMatchService
	objectiveService      *service.ObjectiveService
	characterService      *service.CharacterService
	ladderService         *service.LadderService
	atlasService          *service.AtlasService
	timingRepository      *repository.TimingRepository
	characterRepository   *repository.CharacterRepository
	activityRepository    *repository.ActivityRepository
	timings               map[repository.TimingKey]time.Duration

	lastLadderUpdate time.Time
	client           *client.PoEClient
	event            *repository.Event
}

func (s *PlayerFetchingService) ReloadTimings() error {
	timings, err := s.timingRepository.GetTimings()
	if err != nil {
		return err
	}
	s.timings = timings
	return nil
}

func NewPlayerFetchingService(client *client.PoEClient, event *repository.Event) *PlayerFetchingService {
	return &PlayerFetchingService{
		userRepository:        repository.NewUserRepository(),
		objectiveMatchService: service.NewObjectiveMatchService(),
		objectiveService:      service.NewObjectiveService(),
		ladderService:         service.NewLadderService(),
		characterService:      service.NewCharacterService(),
		atlasService:          service.NewAtlasService(),
		timingRepository:      repository.NewTimingRepository(),
		characterRepository:   repository.NewCharacterRepository(),
		activityRepository:    repository.NewActivityRepository(),
		lastLadderUpdate:      time.Now().Add(-1 * time.Hour),
		client:                client,
		event:                 event,
	}
}

func (s *PlayerFetchingService) shouldUpdateLadder(timings map[repository.TimingKey]time.Duration) bool {
	return time.Since(s.lastLadderUpdate) > timings[repository.LadderUpdateInterval]
}

func (s *PlayerFetchingService) UpdateCharacterName(player *parser.PlayerUpdate, event *repository.Event) {
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
	characterResponse, clientError := s.client.GetCharacter(player.Token, player.New.CharacterName, event.GetRealm())
	player.Mu.Lock()
	defer player.Mu.Unlock()
	player.LastUpdateTimes.Character = time.Now()
	if clientError != nil {
		player.SuccessiveErrors++
		if clientError.StatusCode == 401 || clientError.StatusCode == 403 {
			fmt.Printf("Error fetching character for player %d: %d", player.UserId, clientError.StatusCode)
			player.TokenExpiry = time.Now()
			return
		}
		if clientError.StatusCode == 404 {
			fmt.Printf("Character not found for player %d: %s\n", player.UserId, player.New.CharacterName)
			player.New.CharacterName = ""
			return
		}
		log.Printf("Error fetching character for player %d: %v", player.UserId, clientError)
		return
	}

	player.SuccessiveErrors = 0
	player.New.CharacterName = characterResponse.Character.Name
	player.New.CharacterId = characterResponse.Character.Id
	player.New.CharacterLevel = characterResponse.Character.Level
	player.New.CharacterXP = characterResponse.Character.Experience
	player.New.Ascendancy = characterResponse.Character.Class
	player.New.Pantheon = characterResponse.Character.HasPantheon()
	player.New.AscendancyPoints = characterResponse.Character.GetAscendancyPoints()
	player.New.MainSkill = characterResponse.Character.GetMainSkill()
	player.New.EquipmentHash = characterResponse.Character.EquipmentHash()
	if player.New.EquipmentHash != player.Old.EquipmentHash || time.Since(player.LastUpdateTimes.PoB) > 15*time.Minute {
		charQueue <- characterResponse.Character
		player.LastUpdateTimes.PoB = time.Now()
	}
	character := &repository.Character{
		Id:               player.New.CharacterId,
		UserId:           &player.UserId,
		EventId:          event.Id,
		Name:             player.New.CharacterName,
		Level:            player.New.CharacterLevel,
		MainSkill:        player.New.MainSkill,
		Ascendancy:       player.New.Ascendancy,
		AscendancyPoints: player.New.AscendancyPoints,
		Pantheon:         player.New.Pantheon,
		AtlasPoints:      player.New.MaxAtlasTreeNodes(),
	}
	fmt.Printf("Saving character %s (%s) for user %d\n", character.Name, character.Id, player.UserId)
	fmt.Printf("Character details: Level %d, Main Skill %s, Ascendancy %s, Ascendancy Points %d, Pantheon %v, Atlas Points %d\n", characterResponse.Character.Level, characterResponse.Character.GetMainSkill(), characterResponse.Character.Class, characterResponse.Character.GetAscendancyPoints(), characterResponse.Character.HasPantheon(), character.New.AtlasPoints)
	err := s.characterRepository.Save(character)
	if err != nil {
		fmt.Printf("Error saving character %s (%s) for user %d: %v\n", character.Name, character.Id, player.UserId, err)
	}
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
	if len(player.New.AtlasPassiveTrees) > 0 {
		err := s.atlasService.SaveAtlasTrees(player.UserId, s.event.Id, player.New.AtlasPassiveTrees)
		if err != nil {
			fmt.Printf("Error saving atlas trees %d: %v\n", player.UserId, err)
		}
	}

}

func (s *PlayerFetchingService) UpdateLadder(players []*parser.PlayerUpdate) {
	if !s.shouldUpdateLadder(s.timings) {
		return
	}
	s.lastLadderUpdate = time.Now()
	var resp *client.GetLeagueLadderResponse
	var clientError *client.ClientError
	if s.event.GameVersion == repository.PoE2 {
		// todo: get the ladder for the correct event
		resp, clientError = s.client.GetPoE2Ladder(s.event.Name)
	} else {
		resp, clientError = s.client.GetFullLadder(config.Env().OldPOEToken, s.event.Name)
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
			if entry.Character.Experience != nil {
				player.New.CharacterXP = *entry.Character.Experience
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
	latestCharacters, err := service.characterService.GetCharactersForEvent(service.event.Id)
	if err != nil {
		return nil, err
	}
	characterMap := make(map[int]*repository.Character, len(latestCharacters))
	for _, character := range latestCharacters {
		characterMap[*character.UserId] = character
	}

	for _, player := range players {
		if character, ok := characterMap[player.UserId]; ok {
			player.New.CharacterName = character.Name
			player.Old.CharacterName = character.Name
			player.New.CharacterId = character.Id
			player.Old.CharacterId = character.Id
			player.New.CharacterLevel = character.Level
			player.Old.CharacterLevel = character.Level
			player.New.MainSkill = character.MainSkill
			player.Old.MainSkill = character.MainSkill
			player.New.Pantheon = character.Pantheon
			player.Old.Pantheon = character.Pantheon
			player.New.Ascendancy = character.Ascendancy
			player.Old.Ascendancy = character.Ascendancy
			player.New.AscendancyPoints = character.AscendancyPoints
			player.Old.AscendancyPoints = character.AscendancyPoints
			player.LastUpdateTimes.CharacterName = time.Now()

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

func updateStats(character *client.Character, event *repository.Event, characterRepo *repository.CharacterRepository, cache map[string]*PlayerStatsCache, mu *sync.Mutex) {
	pob, export, err := client.GetPoBExport(character)
	if err != nil {
		fmt.Printf("Error fetching PoB export for character %s: %v\n", character.Name, err)
		return
	}
	stats := pob.Build.PlayerStats
	newStats := &repository.CharacterStat{
		Time:          time.Now(),
		EventId:       event.Id,
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
	mu.Lock()
	defer mu.Unlock()
	if cache[character.Name] == nil {
		cache[character.Name] = &PlayerStatsCache{
			OldStats: &repository.CharacterStat{},
		}
		if character.Equipment != nil {
			cache[character.Name].NumFilledSlots = len(*character.Equipment)
		}
	}
	// trying to filter out saving stats for characters that are missing equipment pieces if their DPS went down
	cacheItem := cache[character.Name]
	if newStats.DPS < cacheItem.OldStats.DPS && character.Equipment != nil && cacheItem.NumFilledSlots > len(*character.Equipment) {
		return
	}
	if time.Since(cacheItem.LastPoBUpdate) > 5*time.Minute && export != cacheItem.OldPoBString {
		cacheItem.OldPoBString = export
		cacheItem.LastPoBUpdate = time.Now()
		p := repository.PoBExport{}
		p.FromString(export)

		err := characterRepo.SavePoB(&repository.CharacterPob{
			CharacterId: character.Id,
			Level:       character.Level,
			MainSkill:   character.GetMainSkill(),
			Ascendancy:  character.Class,
			Export:      p,
			Timestamp:   time.Now(),
		})
		pobsSavedCounter.Inc()
		if err != nil {
			log.Printf("Error saving PoB for character %s: %v", character.Name, err)
		}
	}
	if !cacheItem.OldStats.IsEqual(newStats) {
		cacheItem.OldStats = newStats
		err := characterRepo.CreateCharacterStat(newStats)
		if err != nil {
			log.Printf("Error saving character stats for %s: %v", character.Name, err)
			log.Printf("db stats: %+v", newStats)
			log.Printf("client stats: %+v", stats)
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
		cache[pob.CharacterId].OldPoBString = pob.Export.ToString()
		cache[pob.CharacterId].LastPoBUpdate = pob.Timestamp
	}
	return cache

}

func PlayerStatsLoop(ctx context.Context, event *repository.Event) {
	characterRepo := repository.NewCharacterRepository()
	cache := InitCharacterStatsCache(event.Id, characterRepo)
	mu := sync.Mutex{}
	// make sure that only 2 goroutines are running at the same time
	semaphore := make(chan struct{}, 2)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			character, ok := <-charQueue
			pobQueueGauge.Set(float64(len(charQueue)))
			if !ok {
				log.Println("PoB queue closed, stopping player stats loop")
				return
			}
			semaphore <- struct{}{}
			go func(character *client.Character) {
				defer func() { <-semaphore }() // Release the slot when done
				updateStats(character, event, characterRepo, cache, &mu)
				pobsCalculatedCounter.Inc()
			}(character)
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
						service.UpdateCharacter(player, event)
					}(player)
				}
				if player.ShouldUpdateLeagueAccount(service.timings) {
					wg.Add(1)
					go func(player *parser.PlayerUpdate) {
						defer wg.Done()
						service.UpdateLeagueAccount(player)
					}(player)
				}
			}
			if service.shouldUpdateLadder(service.timings) {
				wg.Add(1)
				go func() {
					defer wg.Done()
					service.UpdateLadder(players)
				}()
			}
			wg.Wait()

			for _, player := range players {
				if player.New.CharacterXP != player.Old.CharacterXP {
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
			players = service.UpdatePlayerTokens(players)
			fmt.Printf("PlayerFetchLoop for event %s completed, waiting for next iteration\n", event.Name)
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
	if len(players) == 0 {
		return []*repository.ObjectiveMatch{}
	}
	now := time.Now()
	return utils.Map(teamChecker.CheckForCompletions(players), func(result *parser.CheckResult) *repository.ObjectiveMatch {
		return &repository.ObjectiveMatch{
			ObjectiveId: result.ObjectiveId,
			UserId:      &players[0].UserId,
			Number:      result.Number,
			Timestamp:   now,
			TeamId:      players[0].TeamId,
		}
	})
}
