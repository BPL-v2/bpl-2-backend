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
	stashChannel         chan config.StashChangeMessage
	oauthService         *service.OauthService
	userRepository       *repository.UserRepository
	guildStashRepository *repository.GuildStashRepository
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
			stashChannel:         make(chan config.StashChangeMessage),
			userRepository:       repository.NewUserRepository(),
			guildStashRepository: repository.NewGuildStashRepository(),
		}
	})
	return fetchingService
}

func (f *FetchingService) FetchStashChanges() error {
	token, err := f.oauthService.GetApplicationToken(repository.ProviderPoE)
	if err != nil {
		log.Printf("Failed to get PoE token: %v", err)
		return fmt.Errorf("failed to get PoE token: %w", err)
	}
	initialStashChange, err := f.stashChangeService.GetInitialChangeId(f.event)
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
			f.stashChannel <- config.StashChangeMessage{ChangeId: changeId, NextChangeId: response.NextChangeId, Stashes: response.Stashes}
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
	Fetchers map[string][]*GuildStashFetcher // key is team ID
	mu       sync.Mutex
}

func (f *GuildStashFetchers) GetToken(stashId string) (*string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	fetchersForStashTab, exists := f.Fetchers[stashId]
	if exists && len(fetchersForStashTab) > 0 {
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
	fetchers := make(map[string][]*GuildStashFetcher)
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
	for _, stash := range stashes {
		for _, userId := range stash.UserIds {
			fetchers[stash.Id] = append(fetchers[stash.Id], fetcherMap[int(userId)])
		}
	}
	return &GuildStashFetchers{
		Fetchers: fetchers,
		mu:       sync.Mutex{},
	}
}

func (f *FetchingService) FilterStashChanges() {
	err := config.CreateTopic(f.event.Id)
	if err != nil {
		log.Print(err)
		return
	}

	writer, err := config.GetWriter(f.event.Id)
	if err != nil {
		log.Print(err)
		return
	}
	defer writer.Close()

	for stashChange := range f.stashChannel {
		select {
		case <-f.ctx.Done():
			return
		default:
			stashes := make([]client.PublicStashChange, 0)
			for _, stash := range stashChange.Stashes {
				stashCounterTotal.Inc()
				if stash.League != nil && *stash.League == f.event.Name {
					stashes = append(stashes, stash)
					stashCounterFiltered.Inc()
				}
			}
			message := config.StashChangeMessage{
				ChangeId:     stashChange.ChangeId,
				NextChangeId: stashChange.NextChangeId,
				Stashes:      stashes,
				Timestamp:    time.Now(),
			}
			// make sure that stash changes are only saved if the messages are successfully written to kafka
			f.stashChangeService.SaveStashChangesConditionally(message, f.event.Id,
				func(data []byte) error {
					return writer.WriteMessages(context.Background(),
						kafka.Message{
							Value: data,
						},
					)
				})
		}
	}
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
	fmt.Printf("Processed guild stash %s for team %d\n", tab.Id, tab.TeamId)
	return nil
}

func (f *FetchingService) FetchGuildStashes() {
	kafkaWriter, fetchers, userNameMap, err := f.InitGuildStashFetching()
	if err != nil {
		fmt.Printf("Failed to initialize guild stash fetching: %v\n", err)
		return
	}
	guildStashes, err := f.guildStashRepository.GetByEvent(f.event.Id)
	if err != nil {
		fmt.Printf("failed to get guild stashes for event %d: %v\n", f.event.Id, err)
		return
	}
	defer kafkaWriter.Close()

	for {
		stashChanges := []*client.PublicStashChange{}
		persistedStashes := make([]*repository.GuildStashTab, 0)
		mu := sync.Mutex{}
		wg := sync.WaitGroup{}
		for _, stash := range guildStashes {
			// child stashes are handled by their parent
			if !stash.FetchEnabled || stash.ParentId != nil {
				continue
			}
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
		err := f.guildStashRepository.SaveAll(persistedStashes)
		if err != nil {
			fmt.Printf("Failed to save guild stashes: %v\n", err)
		}
		fmt.Printf("Processed %d guild stashes\n", len(stashChanges))
		select {
		case <-f.ctx.Done():
			fmt.Println("Stopping guild stash fetch loop")
			return
		case <-time.After(1 * time.Minute):
		}
	}
}

func (f *FetchingService) fetchStash(stash repository.GuildStashTab, fetchers *GuildStashFetchers, userNameMap map[int]*string) ([]*client.PublicStashChange, []*repository.GuildStashTab, error) {
	fmt.Printf("Fetching guild stash %s for team %d\n", stash.Id, stash.TeamId)
	updatedStashes := make([]*repository.GuildStashTab, 0)
	stashChanges := make([]*client.PublicStashChange, 0)
	token, err := fetchers.GetToken(stash.Id)
	if err != nil {
		fmt.Printf("No token found for team %d: %v\n", stash.TeamId, err)
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

func addGuildStashesToQueue(kafkaWriter *kafka.Writer, changes []*client.PublicStashChange) {
	message, err := json.Marshal(config.StashChangeMessage{
		ChangeId:     "",
		NextChangeId: "",
		Stashes: utils.Map(changes, func(change *client.PublicStashChange) client.PublicStashChange {
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
	go fetchingService.FetchGuildStashes()
}

func ItemFetchLoop(ctx context.Context, event *repository.Event, poeClient *client.PoEClient) {
	fetchingService := NewFetchingService(ctx, event, poeClient)
	go fetchingService.FetchStashChanges()
	go fetchingService.FilterStashChanges()
}
