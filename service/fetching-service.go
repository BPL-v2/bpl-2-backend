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

	"github.com/segmentio/kafka-go"
)

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
			if count%100 == 0 {
				diff := f.stashChangeService.GetNinjaDifference(changeId)
				fmt.Printf("Difference between ninja and poe change ids: %d\n", diff)
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
				if stash.League != nil && *stash.League == f.event.Name {
					stashes = append(stashes, stash)
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
