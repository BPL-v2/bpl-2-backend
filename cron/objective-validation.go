package cron

import (
	"bpl/client"
	"bpl/parser"
	"bpl/repository"
	"bpl/service"
	"bpl/utils"
	"context"
	"log"
	"strconv"
	"strings"
	"time"
)

func AddToChangeId(changeId string, value int) string {
	parts := strings.Split(changeId, "-")
	for i, part := range parts {
		num, err := strconv.Atoi(part)
		if err != nil {
			return changeId
		}
		parts[i] = strconv.Itoa(num + value)
	}
	return strings.Join(parts, "-")
}

func ValidationLoop(ctx context.Context, event *repository.Event, poeClient *client.PoEClient) {
	objectiveMatchRepository := repository.NewObjectiveMatchRepository()
	m, err := NewMatchingService(ctx, poeClient, event)
	if err != nil {
		log.Fatal("Failed to create matching service:", err)
	}
	objectives, err := m.objectiveService.GetObjectivesForEvent(event.Id, "Conditions")
	if err != nil {
		log.Print("Failed to get objectives for event:", err)
		return
	}
	objectiveMap := make(map[int]*repository.Objective)
	for _, obj := range objectives {
		objectiveMap[obj.Id] = obj
	}
	itemChecker, err := parser.NewItemChecker(objectives, false)
	if err != nil {
		log.Print("Failed to create item checker:", err)
		return
	}

	token, err := service.NewOauthService().GetApplicationToken(repository.ProviderPoE)
	if err != nil {
		log.Print("Failed to get PoE token:", err)
		return
	}
	initialStashChange, err := service.GetNinjaChangeId()
	if err != nil {
		log.Print("Failed to get initial stash change ID:", err)
		return
	}
	changeId := AddToChangeId(initialStashChange, -100000)
	consecutiveErrors := 0
	for {
		select {
		case <-ctx.Done():
			return
		default:
			response, clientError := poeClient.GetPublicStashes(token, "pc", changeId)
			if clientError != nil {
				consecutiveErrors++
				if consecutiveErrors > 5 {
					log.Print("Too many consecutive errors, exiting")
					return
				}
				if clientError.StatusCode == 429 {
					log.Print(clientError.ResponseHeaders)
					retryAfter, err := strconv.Atoi(clientError.ResponseHeaders.Get("Retry-After"))
					if err != nil {
						retryAfter = 60
					}
					<-time.After((time.Duration(retryAfter) + 1) * time.Second)
				} else {
					log.Print(clientError)
					<-time.After(60 * time.Second)
				}
				continue
			}
			validations := make(map[int]*repository.ObjectiveValidation, 0)
			changeId = response.NextChangeId
			for _, stash := range response.Stashes {
				for _, item := range stash.Items {
					for _, match := range itemChecker.CheckForCompletions(&item) {
						validations[match.ObjectiveId] = &repository.ObjectiveValidation{
							ObjectiveId: match.ObjectiveId,
							Timestamp:   time.Now(),
							Item:        item,
						}

					}
				}
			}
			if len(validations) > 0 {
				err := objectiveMatchRepository.SaveValidations(utils.Values(validations))
				if err != nil {
					log.Print("Failed to save validations:", err)
					return
				}
			}
		}
	}
}
