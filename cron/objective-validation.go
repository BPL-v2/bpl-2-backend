package cron

import (
	"bpl/client"
	"bpl/parser"
	"bpl/repository"
	"bpl/service"
	"bpl/utils"
	"context"
	"log/slog"
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

func ValidationLoop(ctx context.Context, poeClient *client.PoEClient) {
	objectiveMatchRepository := repository.NewObjectiveMatchRepository()
	objectiveService := service.NewObjectiveService()
	objectives, err := objectiveService.GetAllObjectives()
	for _, obj := range objectives {
		obj.ValidFrom = nil
		obj.ValidTo = nil
	}

	if err != nil {
		slog.Error("failed to get objectives for event", "error", err)
		return
	}
	objectiveMap := make(map[int]*repository.Objective)
	for _, obj := range objectives {
		objectiveMap[obj.Id] = obj
	}
	itemChecker, err := parser.NewItemChecker(objectives, false)
	if err != nil {
		slog.Error("failed to create item checker", "error", err)
		return
	}

	token, err := service.NewOauthService().GetApplicationToken(repository.ProviderPoE)
	if err != nil {
		slog.Error("failed to get PoE token", "error", err)
		return
	}
	initialStashChange, err := service.GetNinjaChangeId()
	if err != nil {
		slog.Error("failed to get initial stash change id", "error", err)
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
					slog.Error("too many consecutive errors, exiting")
					return
				}
				if clientError.StatusCode == 429 {
					slog.Warn("rate limited", "headers", clientError.ResponseHeaders)
					retryAfter, err := strconv.Atoi(clientError.ResponseHeaders.Get("Retry-After"))
					if err != nil {
						retryAfter = 60
					}
					<-time.After((time.Duration(retryAfter) + 1) * time.Second)
				} else {
					slog.Error("failed to get public stashes", "error", clientError)
					<-time.After(60 * time.Second)
				}
				continue
			}
			validations := make(map[int]*repository.ObjectiveValidation, 0)
			changeId = response.NextChangeId
			for _, stash := range response.Stashes {
				for _, item := range stash.Items {
					for _, match := range itemChecker.CheckForCompletions(&item) {
						slog.Debug("item validated objective", "item", item.Name, "objective", objectiveMap[match.ObjectiveId].Name)
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
					slog.Error("failed to save validations", "error", err)
					return
				}
			} else {
				slog.Debug("no validations found in batch", "change_id", changeId)
			}
		}
	}
}
