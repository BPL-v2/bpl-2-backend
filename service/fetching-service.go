package service

import (
	"bpl/client"
	"bpl/config"
	"bpl/repository"
	"context"
	"fmt"
	"os"
	"strconv"
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
	ctx                context.Context
	event              *repository.Event
	poeClient          *client.PoEClient
	stashChangeService *StashChangeService
	stashChannel       chan config.StashChangeMessage
}

func NewFetchingService(ctx context.Context, event *repository.Event, poeClient *client.PoEClient) *FetchingService {
	stashChangeService := NewStashChangeService()

	return &FetchingService{
		ctx:                ctx,
		event:              event,
		poeClient:          poeClient,
		stashChangeService: stashChangeService,
		stashChannel:       make(chan config.StashChangeMessage),
	}
}

func (f *FetchingService) FetchStashChanges() error {
	token := os.Getenv("POE_CLIENT_TOKEN")
	if token == "" {
		return fmt.Errorf("POE_CLIENT_TOKEN environment variable not set")
	}
	initialStashChange, err := f.stashChangeService.GetInitialChangeId(f.event)
	if err != nil {
		fmt.Println(err)
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
			fmt.Println("Fetching stashes with change id:", changeId)
			response, err := f.poeClient.GetPublicStashes(token, "pc", changeId)
			if err != nil {
				consecutiveErrors++
				if consecutiveErrors > 5 {
					fmt.Println("Too many consecutive errors, exiting")
					return fmt.Errorf("too many consecutive errors")
				}
				if err.StatusCode == 429 {
					fmt.Println(err.ResponseHeaders)
					retryAfter, err := strconv.Atoi(err.ResponseHeaders.Get("Retry-After"))
					if err != nil {
						retryAfter = 60
					}
					<-time.After((time.Duration(retryAfter) + 1) * time.Second)
				} else {
					fmt.Println(err)
					<-time.After(60 * time.Second)
				}
				continue
			}
			consecutiveErrors = 0
			f.stashChannel <- config.StashChangeMessage{ChangeID: changeId, NextChangeID: response.NextChangeID, Stashes: response.Stashes}
			changeId = response.NextChangeID
			changeIdGauge.Add(float64(ChangeIdToInt(changeId)))
			if count%20 == 0 {
				ninjaId, err := f.stashChangeService.GetNinjaChangeId()
				if err == nil {
					ninjaChangeIdGauge.Add(float64(ChangeIdToInt(ninjaId)))
				}

			}
			count++
		}
	}
}

func (f *FetchingService) FilterStashChanges() {
	err := config.CreateTopic(f.event.ID)
	if err != nil {
		fmt.Println(err)
		return
	}

	writer, err := config.GetWriter(f.event.ID)
	if err != nil {
		fmt.Println(err)
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
				ChangeID:     stashChange.ChangeID,
				NextChangeID: stashChange.NextChangeID,
				Timestamp:    time.Now(),
			}
			fmt.Printf("Found %d stashes\n", len(stashes))
			// make sure that stash changes are only saved if the messages are successfully written to kafka
			f.stashChangeService.SaveStashChangesConditionally(stashes, message, f.event.ID,
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

func FetchLoop(ctx context.Context, event *repository.Event, poeClient *client.PoEClient) {
	fetchingService := NewFetchingService(ctx, event, poeClient)
	go fetchingService.FetchStashChanges()
	go fetchingService.FilterStashChanges()
}
