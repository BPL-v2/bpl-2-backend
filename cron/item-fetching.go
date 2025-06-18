package cron

import (
	"bpl/client"
	"bpl/config"
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

func NewFetchingService(ctx context.Context, event *repository.Event, poeClient *client.PoEClient) *FetchingService {
	stashChangeService := service.NewStashChangeService()

	return &FetchingService{
		ctx:                  ctx,
		event:                event,
		poeClient:            poeClient,
		stashChangeService:   stashChangeService,
		oauthService:         service.NewOauthService(),
		stashChannel:         make(chan config.StashChangeMessage),
		userRepository:       repository.NewUserRepository(),
		guildStashRepository: repository.NewGuildStashRepository(),
	}
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
	Fetchers map[int][]*GuildStashFetcher // key is team ID
	mu       sync.Mutex
}

func (f *GuildStashFetchers) GetToken(teamId int) (*string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	fetchersForTeam, exists := f.Fetchers[teamId]
	if exists && len(fetchersForTeam) > 0 {
		// Return the token of the first fetcher for the team
		sort.Slice(fetchersForTeam, func(i, j int) bool {
			return fetchersForTeam[i].LastUse.Before(fetchersForTeam[j].LastUse)
		})
		fetcher, found := utils.FindFirst(fetchersForTeam, func(f *GuildStashFetcher) bool {
			return f.TokenExpiry.After(time.Now())
		})
		if found {
			fetcher.LastUse = time.Now()
			return &fetcher.Token, nil
		}
	}
	return nil, fmt.Errorf("no fetcher found for team %d", teamId)
}

func InitFetchers(users []*repository.TeamUserWithPoEToken) *GuildStashFetchers {
	fetchers := make(map[int][]*GuildStashFetcher)
	for _, user := range users {
		fetcher := &GuildStashFetcher{
			UserId:      user.UserId,
			Token:       user.Token,
			TokenExpiry: user.TokenExpiry,
			LastUse:     time.Now(),
		}
		fetchers[user.TeamId] = append(fetchers[user.TeamId], fetcher)
	}
	return &GuildStashFetchers{
		Fetchers: fetchers,
		mu:       sync.Mutex{},
	}
}

func (f *FetchingService) FetchGuildStashes() error {
	users, err := f.userRepository.GetUsersForEvent(f.event.Id)
	if err != nil {
		return err
	}
	guildStashes, err := f.guildStashRepository.GetByEvent(f.event.Id)
	if err != nil {
		return fmt.Errorf("failed to get guild stashes for event %d: %w", f.event.Id, err)
	}
	fetchers := InitFetchers(users)
	for _, stash := range guildStashes {
		if !stash.FetchEnabled {
			continue
		}
		token, err := fetchers.GetToken(stash.TeamId)
		if err != nil {
			fmt.Printf("No token found for team %d: %v\n", stash.TeamId, err)
			continue
		}
		fmt.Printf("Fetching guild stash %s for team %d in event %s with token %s\n", stash.Id, stash.TeamId, f.event.Name, *token)
		response, httpError := f.poeClient.GetGuildStash(*token, f.event.Name, stash.Id, nil)
		if httpError != nil {
			fmt.Printf("Failed to fetch guild stash %s for team %d: %d - %s\n", stash.Id, stash.TeamId, httpError.StatusCode, httpError.Description)
			return fmt.Errorf("failed to fetch guild stash %s for team %d: %d - %s", stash.Id, stash.TeamId, httpError.StatusCode, httpError.Description)
		}
		stash.LastFetch = time.Now()
		stash.Index = response.Stash.Index
		stash.Name = response.Stash.Name
		stash.Type = response.Stash.Type
		stash.Color = response.Stash.Metadata.Colour
		if response.Stash.Items != nil {
			items, err := json.Marshal(response.Stash.Items)
			if err != nil {
				fmt.Printf("Failed to marshal items for stash %s: %v\n", stash.Id, err)
				return fmt.Errorf("failed to marshal items for stash %s: %w", stash.Id, err)
			}
			stash.Raw = string(items)
		}
		f.guildStashRepository.Save(stash)

	}
	return f.guildStashRepository.SaveAll(guildStashes)
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

func ItemFetchLoop(ctx context.Context, event *repository.Event, poeClient *client.PoEClient) {
	fetchingService := NewFetchingService(ctx, event, poeClient)
	go fetchingService.FetchStashChanges()
	go fetchingService.FilterStashChanges()
}

func GuildStashFetchLoop(ctx context.Context, event *repository.Event, poeClient *client.PoEClient) {
	fetchingService := NewFetchingService(ctx, event, poeClient)
	go fetchingService.FetchGuildStashes()
}
