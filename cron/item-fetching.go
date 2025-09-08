package cron

import (
	"bpl/client"
	"bpl/config"
	"bpl/parser"
	"bpl/repository"
	"bpl/service"
	"bpl/utils"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/segmentio/kafka-go"
)

var stashCounterTotal = promauto.NewCounter(prometheus.CounterOpts{
	Name: "stash_counter_total",
	Help: "The total number of stashes processed",
})

var stashCounterFiltered = promauto.NewCounter(prometheus.CounterOpts{
	Name: "stash_counter_filtered",
	Help: "The total number of stashes filtered",
})

var changeIdGauge = promauto.NewGauge(prometheus.GaugeOpts{
	Name: "change_id",
	Help: "The current change id",
})

var ninjaChangeIdGauge = promauto.NewGauge(prometheus.GaugeOpts{
	Name: "ninja_change_id",
	Help: "The current change id from the poe.ninja api",
})

type FetchingService struct {
	ctx                  context.Context
	event                *repository.Event
	poeClient            *client.PoEClient
	stashChangeService   *service.StashChangeService
	stashChannel         chan repository.StashChangeMessage
	oauthService         *service.OauthService
	userRepository       *repository.UserRepository
	guildStashRepository *repository.GuildStashRepository
	activityRepository   *repository.ActivityRepository
}

var (
	fetchingService *FetchingService
	once            sync.Once
)

func NewFetchingService(ctx context.Context, event *repository.Event, poeClient *client.PoEClient) *FetchingService {
	once.Do(func() {
		stashChangeService := service.NewStashChangeService()
		fetchingService = &FetchingService{
			ctx:                  ctx,
			event:                event,
			poeClient:            poeClient,
			stashChangeService:   stashChangeService,
			oauthService:         service.NewOauthService(),
			stashChannel:         make(chan repository.StashChangeMessage),
			userRepository:       repository.NewUserRepository(),
			guildStashRepository: repository.NewGuildStashRepository(),
			activityRepository:   repository.NewActivityRepository(),
		}
	})
	return fetchingService
}

func (f *FetchingService) FetchStashChanges() error {
	fmt.Println("Starting stash change fetch loop")
	token, err := f.oauthService.GetApplicationToken(repository.ProviderPoE)
	if err != nil {
		log.Printf("Failed to get PoE token: %v", err)
		return fmt.Errorf("failed to get PoE token: %w", err)
	}
	fmt.Printf("Using token %s for event %d\n", token, f.event.Id)
	initialStashChange, err := f.stashChangeService.GetInitialChangeId(f.event)
	fmt.Printf("Initial stash change ID: %s\n", initialStashChange)
	if err != nil {
		log.Print(err)
		return nil
	}

	changeId := initialStashChange
	count := 0
	consecutiveErrors := 0
	for {
		select {
		case <-f.ctx.Done():
			return nil
		default:
			fmt.Printf("Fetching stash changes for event %d, change ID: %s\n", f.event.Id, changeId)
			response, err := f.poeClient.GetPublicStashes(token, "pc", changeId)
			if err != nil {
				consecutiveErrors++
				if consecutiveErrors > 5 {
					log.Print("Too many consecutive errors, exiting")
					return fmt.Errorf("too many consecutive errors")
				}
				if err.StatusCode == 429 {
					log.Print(err.ResponseHeaders)
					retryAfter, err := strconv.Atoi(err.ResponseHeaders.Get("Retry-After"))
					if err != nil {
						retryAfter = 60
					}
					<-time.After((time.Duration(retryAfter) + 1) * time.Second)
				} else {
					log.Print(err)
					<-time.After(60 * time.Second)
				}
				continue
			}
			consecutiveErrors = 0
			f.stashChannel <- repository.StashChangeMessage{ChangeId: changeId, NextChangeId: response.NextChangeId, Stashes: response.Stashes}
			changeId = response.NextChangeId
			changeIdGauge.Set(float64(service.ChangeIdToInt(changeId)))
			if count%20 == 0 {
				ninjaId, err := f.stashChangeService.GetNinjaChangeId()
				if err == nil {
					ninjaChangeIdGauge.Set(float64(service.ChangeIdToInt(ninjaId)))
				}
			}
			count++
		}
	}
}

type GuildStashFetcher struct {
	UserId      int
	Token       string
	TokenExpiry time.Time
	LastUse     time.Time
}

type GuildStashFetchers struct {
	Fetchers map[int]*GuildStashFetcher // key is team ID
	mu       sync.Mutex
}

func (f *GuildStashFetchers) GetToken(stash *repository.GuildStashTab) (*string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	stashId := stash.Id
	if stash.ParentId != nil {
		stashId = *stash.ParentId
	}
	fetchersForStashTab := []*GuildStashFetcher{}
	for _, userId := range stash.UserIds {
		if fetcher, exists := f.Fetchers[int(userId)]; exists {
			fetchersForStashTab = append(fetchersForStashTab, fetcher)
		}
	}
	if len(fetchersForStashTab) > 0 {
		// Return the token of the first fetcher for the team
		sort.Slice(fetchersForStashTab, func(i, j int) bool {
			return fetchersForStashTab[i].LastUse.Before(fetchersForStashTab[j].LastUse)
		})
		fetcher, found := utils.FindFirst(fetchersForStashTab, func(f *GuildStashFetcher) bool {
			return f.TokenExpiry.After(time.Now())
		})
		if found {
			fetcher.LastUse = time.Now()
			return &fetcher.Token, nil
		}
	}
	return nil, fmt.Errorf("no fetcher found for stash %s", stashId)
}

func InitFetchers(users []*repository.TeamUserWithPoEToken, stashes []*repository.GuildStashTab) *GuildStashFetchers {
	fetcherMap := make(map[int]*GuildStashFetcher)
	for _, user := range users {
		fetcher := &GuildStashFetcher{
			UserId:      user.UserId,
			Token:       user.Token,
			TokenExpiry: user.TokenExpiry,
			LastUse:     time.Now(),
		}
		fetcherMap[user.UserId] = fetcher
	}
	return &GuildStashFetchers{
		Fetchers: fetcherMap,
		mu:       sync.Mutex{},
	}
}

func (f *FetchingService) FilterStashChanges() error {
	err := config.CreateTopic(f.event.Id)
	if err != nil {
		return fmt.Errorf("failed to create kafka topic: %w", err)
	}
	users, err := f.userRepository.GetUsersForEvent(f.event.Id)
	if err != nil {
		return fmt.Errorf("failed to get users for event %d: %w", f.event.Id, err)
	}
	userMap := make(map[string]int)
	for _, user := range users {
		userMap[user.AccountName] = user.UserId
	}

	writer, err := config.GetWriter(f.event.Id)
	if err != nil {
		return fmt.Errorf("failed to get kafka writer: %w", err)

	}
	defer writer.Close()

	for stashChange := range f.stashChannel {
		select {
		case <-f.ctx.Done():
			return fmt.Errorf("context canceled")
		default:
			fmt.Println("filtering stash change", stashChange.ChangeId)
			stashes := make([]client.PublicStashChange, 0)
			now := time.Now()
			for _, stash := range stashChange.Stashes {
				stashCounterTotal.Inc()
				if stash.League != nil && *stash.League == f.event.Name {
					stashes = append(stashes, stash)
					stashCounterFiltered.Inc()
					if stash.AccountName != nil && userMap[*stash.AccountName] != 0 {
						err = f.activityRepository.SaveActivity(&repository.Activity{
							Time:    now.Add(-5 * time.Minute),
							UserId:  userMap[*stash.AccountName],
							EventId: f.event.Id,
						})
						if err != nil {
							log.Printf("Failed to save activity for user %s: %v", *stash.AccountName, err)
						}
					}
				}
			}
			message := repository.StashChangeMessage{
				ChangeId:     stashChange.ChangeId,
				NextChangeId: stashChange.NextChangeId,
				Stashes:      stashes,
				Timestamp:    now,
			}
			log.Printf("Writing %d stashes message to kafka: %s\n", len(stashes), stashChange.ChangeId)
			// make sure that stash changes are only saved if the messages are successfully written to kafka
			err = f.stashChangeService.SaveStashChangesConditionally(message, f.event.Id,
				func(data []byte) error {
					return writer.WriteMessages(context.Background(),
						kafka.Message{
							Value: data,
						},
					)
				})
			if err != nil {
				log.Printf("Failed to save stash changes conditionally: %v", err)
			}
		}
	}
	return nil
}

func (f *FetchingService) InitGuildStashFetching() (kafkaWriter *kafka.Writer, fetchers *GuildStashFetchers, userNameMap map[int]*string, err error) {
	users, err := f.userRepository.GetUsersForEvent(f.event.Id)
	if err != nil {
		log.Printf("Failed to get users for event %d: %v", f.event.Id, err)
		return
	}
	userNameMap = make(map[int]*string)
	for _, user := range users {
		userNameMap[user.UserId] = &user.AccountName
	}
	// todo: move this redundant db call
	stashes, err := f.guildStashRepository.GetByEvent(f.event.Id)
	if err != nil {
		log.Printf("Failed to get stashes for event %d: %v", f.event.Id, err)
		return
	}
	fetchers = InitFetchers(users, stashes)
	err = config.CreateTopic(f.event.Id)
	if err != nil {
		log.Print(err)
		return
	}
	kafkaWriter, err = config.GetWriter(f.event.Id)
	if err != nil {
		log.Print(err)
		return nil, nil, nil, fmt.Errorf("failed to get kafka writer: %w", err)
	}
	return kafkaWriter, fetchers, userNameMap, nil
}

func (f *FetchingService) FetchGuildStashTab(tab *repository.GuildStashTab) error {
	kafkaWriter, fetchers, userNameMap, err := f.InitGuildStashFetching()
	if err != nil {
		fmt.Printf("Failed to initialize guild stash fetching: %v\n", err)
		return err
	}
	stashChanges, persistedStashes, err := f.fetchStash(*tab, fetchers, userNameMap)
	if err != nil {
		fmt.Printf("Failed to fetch stash %s for team %d: %v\n", tab.Id, tab.TeamId, err)
		return err
	}
	addGuildStashesToQueue(kafkaWriter, stashChanges)
	err = f.guildStashRepository.SaveAll(persistedStashes)
	if err != nil {
		fmt.Printf("Failed to save guild stashes: %v\n", err)
		return err
	}
	return nil
}

func (f *FetchingService) AccessDeterminationLoop() {
	for {
		err := f.DetermineStashAccess()
		if err != nil {
			fmt.Printf("Failed to determine stash access: %v\n", err)
		}
		select {
		case <-f.ctx.Done():
			return
		case <-time.After(30 * time.Minute):
		}
	}
}

func (f *FetchingService) DetermineStashAccess() error {
	users, err := f.userRepository.GetUsersForEvent(f.event.Id)
	if err != nil {
		return err
	}
	stashToUsers := make(map[string]pq.Int32Array)
	stashMap := make(map[string]client.GuildStashTabGGG)
	userMap := make(map[int]*repository.TeamUserWithPoEToken)
	for _, user := range users {
		userMap[user.UserId] = user
	}
	mu := sync.Mutex{}
	wg := sync.WaitGroup{}
	for _, user := range users {
		wg.Add(1)
		go func(user *repository.TeamUserWithPoEToken) {
			defer wg.Done()
			stashes, err := f.GetAvailableStashes(user)
			if err != nil {
				return
			}
			mu.Lock()
			for _, stash := range *stashes {
				stashToUsers[stash.Id] = append(stashToUsers[stash.Id], int32(user.UserId))
				stashMap[stash.Id] = stash
				if stash.Children != nil {
					for _, stashChild := range *stash.Children {
						stashToUsers[stashChild.Id] = append(stashToUsers[stashChild.Id], int32(user.UserId))
						stashMap[stashChild.Id] = stashChild
					}
				}
			}
			mu.Unlock()
		}(user)
	}
	wg.Wait()

	existingStashes, err := f.guildStashRepository.GetByEvent(f.event.Id)
	if err != nil {
		return err
	}
	existingStashMap := make(map[string]*repository.GuildStashTab)
	for _, stash := range existingStashes {
		existingStashMap[stash.Id] = stash
	}

	persistedStashes := make([]*repository.GuildStashTab, 0)
	for _, stash := range stashMap {
		users := stashToUsers[stash.Id]
		existingStash, exists := existingStashMap[stash.Id]
		if !exists {
			user := userMap[int(users[0])]
			persistedStashes = append(persistedStashes, &repository.GuildStashTab{
				Id:           stash.Id,
				EventId:      f.event.Id,
				TeamId:       user.TeamId,
				OwnerId:      user.UserId,
				Name:         stash.Name,
				Type:         stash.Type,
				Index:        stash.Index,
				Color:        stash.Metadata.Colour,
				ParentId:     stash.Parent,
				FetchEnabled: false,
				LastFetch:    time.Now(),
				UserIds:      stashToUsers[stash.Id],
				Raw:          "{}",
			})
		} else {
			existingStash.UserIds = stashToUsers[stash.Id]
			existingStash.Color = stash.Metadata.Colour
			existingStash.Name = stash.Name
			existingStash.Type = stash.Type
			existingStash.Index = stash.Index
			persistedStashes = append(persistedStashes, existingStash)
		}
	}
	err = f.guildStashRepository.SaveAll(persistedStashes)
	if err != nil {
		return err
	}
	return nil
}

func (f *FetchingService) GetAvailableStashes(user *repository.TeamUserWithPoEToken) (*[]client.GuildStashTabGGG, error) {
	if user.TokenExpiry.Before(time.Now()) {
		return nil, fmt.Errorf("token expired")
	}
	response, err := f.poeClient.ListGuildStashes(user.Token, f.event.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to list guild stashes for user %d: %v", user.UserId, err)
	}
	return &response.Stashes, nil
}

func (f *FetchingService) FetchGuildStashes() error {
	kafkaWriter, fetchers, userNameMap, err := f.InitGuildStashFetching()
	if err != nil {
		return fmt.Errorf("failed to initialize guild stash fetching: %w", err)
	}

	defer kafkaWriter.Close()

	for {
		fmt.Printf("Fetching guild stashes for event %d\n", f.event.Id)
		guildStashes, err := f.guildStashRepository.GetActiveByEvent(f.event.Id)
		if err != nil {
			return fmt.Errorf("failed to get guild stashes for event %d: %w", f.event.Id, err)
		}
		stashChanges := []*client.PublicStashChange{}
		persistedStashes := make([]*repository.GuildStashTab, 0)
		mu := sync.Mutex{}
		wg := sync.WaitGroup{}
		for _, stash := range guildStashes {
			// child stashes are handled by their parent
			if !stash.FetchEnabled || stash.ParentId != nil {
				continue
			}
			fmt.Printf("Fetching guild stash %s for team %d\n", stash.Id, stash.TeamId)
			wg.Add(1)
			go func(stash repository.GuildStashTab) {
				defer wg.Done()
				changes, updatedStashes, err := f.fetchStash(stash, fetchers, userNameMap)
				if err != nil {
					fmt.Printf("Failed to fetch stash %s for team %d: %v\n", stash.Id, stash.TeamId, err)
					return
				}
				mu.Lock()
				defer mu.Unlock()
				stashChanges = append(stashChanges, changes...)
				persistedStashes = append(persistedStashes, updatedStashes...)
			}(*stash)
		}
		wg.Wait()
		addGuildStashesToQueue(kafkaWriter, stashChanges)
		err = f.guildStashRepository.SaveAll(persistedStashes)
		if err != nil {
			fmt.Printf("Failed to save guild stashes: %v\n", err)
		}
		select {
		case <-f.ctx.Done():
			return fmt.Errorf("context canceled")
		case <-time.After(10 * time.Second):
		}
	}
}

func (f *FetchingService) fetchStash(stash repository.GuildStashTab, fetchers *GuildStashFetchers, userNameMap map[int]*string) ([]*client.PublicStashChange, []*repository.GuildStashTab, error) {
	updatedStashes := make([]*repository.GuildStashTab, 0)
	stashChanges := make([]*client.PublicStashChange, 0)
	token, err := fetchers.GetToken(&stash)
	if err != nil {
		fmt.Printf("No token found for team %d: %v\n", stash.TeamId, err)
		updatedStashes = append(updatedStashes, &stash)
		return stashChanges, updatedStashes, nil
	}
	response, httpError := f.poeClient.GetGuildStash(*token, f.event.Name, stash.Id, stash.ParentId)
	if httpError != nil {
		return nil, nil, fmt.Errorf("failed to fetch guild stash %s for team %d: %d - %s", stash.Id, stash.TeamId, httpError.StatusCode, httpError.Description)
	}
	stash.LastFetch = time.Now()
	stash.Index = response.Stash.Index
	stash.Name = response.Stash.Name
	stash.Type = response.Stash.Type
	stash.Color = response.Stash.Metadata.Colour
	if response.Stash.Items != nil {
		stashChanges = append(stashChanges, &client.PublicStashChange{
			Id:          stash.Id,
			Public:      true,
			AccountName: userNameMap[stash.OwnerId],
			League:      &f.event.Name,
			Items: utils.Map(
				*response.Stash.Items,
				func(item client.DisplayItem) client.Item { return *item.Item }),
			StashType: stash.Type,
		})
		if response.Stash.Type == "UniqueStash" && len(*response.Stash.Items) > 0 {
			stash.Name = parser.ItemClasses[(*response.Stash.Items)[0].BaseType]
		}
	}

	raw, err := json.Marshal(response.Stash)
	if err != nil {
		fmt.Printf("Failed to marshal items for stash %s: %v\n", stash.Id, err)
		return nil, nil, fmt.Errorf("failed to marshal items for stash %s: %w", stash.Id, err)
	}
	stash.Raw = string(raw)

	if response.Stash.Children != nil {
		fmt.Printf("Found %d child stashes for stash %s\n", len(*response.Stash.Children), stash.Id)
		wg := sync.WaitGroup{}
		mu := sync.Mutex{}
		for _, child := range *response.Stash.Children {
			wg.Add(1)
			go func(child client.GuildStashTabGGG) {
				defer wg.Done()
				childChanges, childStashes, err := f.fetchStash(repository.GuildStashTab{
					Id:            child.Id,
					EventId:       f.event.Id,
					TeamId:        stash.TeamId,
					OwnerId:       stash.OwnerId,
					Name:          child.Name,
					Type:          child.Type,
					Index:         child.Index,
					Color:         child.Metadata.Colour,
					ParentId:      &stash.Id,
					ParentEventId: &f.event.Id,
					LastFetch:     time.Now(),
					Raw:           "",
					FetchEnabled:  stash.FetchEnabled,
					UserIds:       stash.UserIds,
				}, fetchers, userNameMap)
				if err != nil {
					fmt.Printf("Failed to fetch child stash %s for team %d: %v\n", child.Id, stash.TeamId, err)
					return
				}
				mu.Lock()
				defer mu.Unlock()
				stashChanges = append(stashChanges, childChanges...)
				updatedStashes = append(updatedStashes, childStashes...)
			}(child)
		}
		wg.Wait()
	}
	updatedStashes = append(updatedStashes, &stash)
	return stashChanges, updatedStashes, nil

}

var (
	GuildStashHashMap      = make(map[string][32]byte)
	GuildStashHashMapMutex = sync.Mutex{}
)

func addGuildStashesToQueue(kafkaWriter *kafka.Writer, changes []*client.PublicStashChange) {
	realChanges := make([]*client.PublicStashChange, 0)
	for _, change := range changes {
		hash := change.GetHash()
		if GuildStashHashMap[change.Id] == hash {
			fmt.Printf("Skipping stash %s, already processed\n", change.Id)
			continue
		}
		GuildStashHashMapMutex.Lock()
		GuildStashHashMap[change.Id] = hash
		GuildStashHashMapMutex.Unlock()
		realChanges = append(realChanges, change)
		fmt.Printf("Adding stash %s to queue\n", change.Id)
	}
	if len(realChanges) == 0 {
		return
	}

	message, err := json.Marshal(repository.StashChangeMessage{
		ChangeId:     "",
		NextChangeId: "",
		Stashes: utils.Map(realChanges, func(change *client.PublicStashChange) client.PublicStashChange {
			return *change
		}),
		Timestamp: time.Now(),
	})
	if err != nil {
		fmt.Printf("Failed to marshal stash change message: %v\n", err)
		return
	}
	err = kafkaWriter.WriteMessages(
		context.Background(),
		kafka.Message{Value: message},
	)
	if err != nil {
		fmt.Printf("Failed to write stash change message to kafka: %v\n", err)
	}
}

func GuildStashFetchLoop(ctx context.Context, event *repository.Event, poeClient *client.PoEClient) {
	fetchingService := NewFetchingService(ctx, event, poeClient)
	go func() {
		err := fetchingService.FetchStashChanges()
		if err != nil {
			fmt.Printf("Failed to fetch stash changes: %v\n", err)
		}
	}()
	go fetchingService.AccessDeterminationLoop()
}

func ItemFetchLoop(ctx context.Context, event *repository.Event, poeClient *client.PoEClient) {
	fmt.Println("Starting item fetch loop")
	fetchingService := NewFetchingService(ctx, event, poeClient)
	go func() {
		err := fetchingService.FetchStashChanges()
		if err != nil {
			fmt.Printf("Failed to fetch stash changes: %v\n", err)
		}
	}()
	go func() {
		err := fetchingService.FilterStashChanges()
		if err != nil {
			fmt.Printf("Failed to filter stash changes: %v\n", err)
		}
	}()

}
