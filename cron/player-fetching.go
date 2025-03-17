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

var ascendancyNodes = utils.ToSet([]int{193, 258, 409, 607, 662, 758, 869, 922, 982, 1105, 1675, 1697, 1729, 1731, 1734, 1945, 2060, 2336, 2521, 2598, 2872, 3184, 3554, 3651, 4194, 4242, 4494, 4849, 4917, 5082, 5087, 5415, 5443, 5643, 5819, 5865, 5926, 6028, 6038, 6052, 6064, 6728, 6778, 6982, 7618, 8281, 8419, 8592, 8656, 9014, 9271, 9327, 9971, 10099, 10143, 10238, 10635, 11046, 11412, 11490, 11597, 12146, 12475, 12597, 12738, 12850, 13219, 13374, 13851, 14103, 14156, 14603, 14726, 14870, 14996, 15286, 15550, 15616, 16023, 16093, 16212, 16306, 16745, 16848, 16940, 17018, 17315, 17445, 17765, 17988, 18309, 18378, 18574, 18635, 19083, 19417, 19488, 19587, 19595, 19598, 19641, 20050, 20480, 20954, 21264, 22551, 22637, 22852, 23024, 23169, 23225, 23509, 23572, 23972, 24214, 24432, 24528, 24538, 24704, 24755, 24848, 24984, 25111, 25167, 25309, 25651, 26067, 26298, 26446, 26714, 27038, 27055, 27096, 27536, 27604, 27864, 28535, 28782, 28884, 28995, 29026, 29294, 29630, 29662, 29825, 29994, 30690, 30919, 30940, 31316, 31344, 31364, 31598, 31667, 31700, 31984, 32115, 32249, 32251, 32364, 32417, 32640, 32662, 32730, 32816, 32947, 32992, 33167, 33179, 33645, 33795, 33875, 33940, 33954, 34215, 34434, 34484, 34567, 34774, 35185, 35598, 35750, 35754, 36017, 36242, 36958, 37114, 37127, 37191, 37419, 37486, 37492, 37623, 38180, 38387, 38689, 38918, 38999, 39598, 39728, 39790, 39818, 39834, 40010, 40059, 40104, 40510, 40631, 40810, 40813, 41081, 41433, 41534, 41891, 41996, 42144, 42264, 42293, 42546, 42659, 42671, 42861, 43122, 43193, 43195, 43215, 43242, 43336, 43725, 43962, 44297, 44482, 44797, 45313, 45403, 45696, 46952, 47366, 47486, 47630, 47778, 47873, 48124, 48214, 48239, 48480, 48719, 48760, 48904, 48999, 49153, 50024, 50692, 50845, 51101, 51462, 51492, 51998, 52575, 53086, 53095, 53123, 53421, 53816, 53884, 53992, 54159, 54279, 54877, 55146, 55236, 55509, 55646, 55686, 55867, 55985, 56134, 56461, 56722, 56789, 56856, 56967, 57052, 57197, 57222, 57429, 57560, 58029, 58229, 58427, 58454, 58650, 58827, 58998, 59800, 59837, 59920, 60462, 60508, 60547, 60769, 60791, 61072, 61259, 61355, 61372, 61393, 61478, 61627, 61761, 61805, 61871, 62067, 62136, 62162, 62349, 62504, 62595, 62817, 63135, 63293, 63357, 63417, 63490, 63583, 63673, 63908, 63940, 64028, 64111, 64768, 64842, 65153, 65296})

type PlayerFetchingService struct {
	userRepository        *repository.UserRepository
	objectiveMatchService *service.ObjectiveMatchService
	objectiveService      *service.ObjectiveService
	ladderService         *service.LadderService
	client                *client.PoEClient
	event                 *repository.Event
}

func NewPlayerFetchingService(client *client.PoEClient, event *repository.Event) *PlayerFetchingService {
	return &PlayerFetchingService{
		userRepository:        repository.NewUserRepository(),
		objectiveMatchService: service.NewObjectiveMatchService(),
		objectiveService:      service.NewObjectiveService(),
		ladderService:         service.NewLadderService(),
		client:                client,
		event:                 event,
	}
}

func (s *PlayerFetchingService) UpdateCharacterName(playerUpdate *parser.PlayerUpdate) {
	if s.event.GameVersion == repository.PoE2 {
		return
	}
	playerUpdate.Mu.Lock()
	defer playerUpdate.Mu.Unlock()
	if !playerUpdate.ShouldUpdateCharacterName() {
		return
	}
	charactersResponse, err := s.client.ListCharacters(playerUpdate.Token)
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
		}
	}
}

func (s *PlayerFetchingService) UpdateCharacter(player *parser.PlayerUpdate) {
	if s.event.GameVersion == repository.PoE2 {
		return
	}
	player.Mu.Lock()
	defer player.Mu.Unlock()
	if !player.ShouldUpdateCharacter() {
		return
	}
	characterResponse, err := s.client.GetCharacter(player.Token, player.New.CharacterName)
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
	player.New.Pantheon = characterResponse.Character.Passives.PantheonMajor != nil && characterResponse.Character.Passives.PantheonMinor != nil
	player.New.AscendancyPoints = len(ascendancyNodes.Intersection(utils.ToSet(characterResponse.Character.Passives.Hashes)))
}

func (s *PlayerFetchingService) UpdateLeagueAccount(player *parser.PlayerUpdate) {
	if s.event.GameVersion == repository.PoE2 {
		return
	}
	player.Mu.Lock()
	defer player.Mu.Unlock()
	if !player.ShouldUpdateLeagueAccount() {
		return
	}
	leagueAccount, err := s.client.GetLeagueAccount(player.Token, s.event.Name)
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
	var resp *client.GetLeagueLadderResponse
	var clientError *client.ClientError
	if s.event.GameVersion == repository.PoE2 {
		// todo: get the ladder for the correct event
		resp, clientError = s.client.GetPoE2Ladder("Standard")
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
	charToUserId := map[string]int{}
	for _, player := range players {
		charToUpdate[player.New.CharacterName] = player
		charToUserId[player.New.CharacterName] = player.UserId
	}

	entriesToPersist := make([]*client.LadderEntry, 0, len(resp.Ladder.Entries))
	for _, entry := range resp.Ladder.Entries {
		if player, ok := charToUpdate[entry.Character.Name]; ok {
			entriesToPersist = append(entriesToPersist, &entry)
			player.Mu.Lock()
			player.New.CharacterLevel = entry.Character.Level
			if entry.Character.Depth != nil && entry.Character.Depth.Depth != nil {
				player.New.DelveDepth = *entry.Character.Depth.Depth
			}
			player.Mu.Unlock()
		}
	}
	err := s.ladderService.UpsertLadder(entriesToPersist, s.event.Id, charToUserId)
	if err != nil {
		log.Print(clientError)
	}
}

func PlayerFetchLoop(ctx context.Context, event *repository.Event, poeClient *client.PoEClient) {
	service := NewPlayerFetchingService(poeClient, event)
	users, err := service.userRepository.GetAuthenticatedUsersForEvent(service.event.Id)
	if err != nil {
		log.Print(err)
		return
	}
	players := utils.Map(users, func(user *repository.TeamUserWithPoEToken) *parser.PlayerUpdate {
		return &parser.PlayerUpdate{
			UserId:      user.UserId,
			TeamId:      user.TeamId,
			AccountName: user.AccountName,
			Token:       user.Token,
			TokenExpiry: user.TokenExpiry,
			New:         &parser.Player{},
			Old:         &parser.Player{},
			Mu:          sync.Mutex{},
			LastUpdateTimes: struct {
				CharacterName time.Time
				Character     time.Time
				LeagueAccount time.Time
			}{},
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
			wg := sync.WaitGroup{}
			for _, player := range players {
				if player.TokenExpiry.Before(time.Now()) {
					continue
				}
				wg.Add(3)
				go func(player *parser.PlayerUpdate) {
					defer wg.Done()
					service.UpdateCharacterName(player)
				}(player)
				go func(player *parser.PlayerUpdate) {
					defer wg.Done()
					service.UpdateCharacter(player)
				}(player)
				go func(player *parser.PlayerUpdate) {
					defer wg.Done()
					service.UpdateLeagueAccount(player)
				}(player)
			}
			wg.Add(1)
			go func() {
				defer wg.Done()
				service.UpdateLadder(players)
			}()
			wg.Wait()

			matches := utils.FlatMap(players, func(player *parser.PlayerUpdate) []*repository.ObjectiveMatch {
				return service.GetPlayerMatches(player, playerChecker)
			})
			service.objectiveMatchService.SaveMatches(matches, []int{})
			time.Sleep(1 * time.Minute)
			for _, player := range players {
				player.Old = player.New
			}

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
