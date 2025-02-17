package service

import (
	"bpl/client"
	"bpl/config"
	"bpl/parser"
	"bpl/repository"
	"bpl/utils"
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"

	"github.com/segmentio/kafka-go"
)

type MatchingService struct {
	ctx                   context.Context
	objectiveMatchService *ObjectiveMatchService
	objectiveService      *ObjectiveService
	lastChangeId          *string
	event                 *repository.Event
}

func NewMatchingService(ctx context.Context, poeClient *client.PoEClient, event *repository.Event) (*MatchingService, error) {
	objectiveMatchService := NewObjectiveMatchService()
	objectiveService := NewObjectiveService()
	matchingService := &MatchingService{
		objectiveMatchService: objectiveMatchService,
		objectiveService:      objectiveService,
		event:                 event,
		ctx:                   ctx,
	}
	changeId, err := NewStashChangeService().GetCurrentChangeIdForEvent(event)
	if err == nil {
		fmt.Println("Last change id:", changeId)
		matchingService.lastChangeId = &changeId
	}
	return matchingService, nil
}

func (m *MatchingService) GetStashChange(reader *kafka.Reader) (stashChange config.StashChangeMessage, err error) {
	msg, err := reader.ReadMessage(context.Background())
	if err != nil {
		return stashChange, err
	}
	if err := json.Unmarshal(msg.Value, &stashChange); err != nil {
		return stashChange, err
	}
	return stashChange, nil
}

func (m *MatchingService) getUserMap() map[string]int {
	userMap := make(map[string]int)
	for _, team := range m.event.Teams {
		for _, user := range team.Users {
			for account := range user.OauthAccounts {
				if user.OauthAccounts[account].Provider == repository.ProviderPoE {
					userMap[user.OauthAccounts[account].AccessToken] = user.ID
				}
			}

		}
	}
	return userMap
}

func (m *MatchingService) getMatches(stashChange config.StashChangeMessage, userMap map[string]int, itemChecker *parser.ItemChecker, desyncedObjectiveIds []int) []*repository.ObjectiveMatch {
	matches := make([]*repository.ObjectiveMatch, 0)
	syncFinished := len(desyncedObjectiveIds) == 0
	for _, stash := range stashChange.Stashes {
		userId := rand.IntN(4) + 1
		// if stash.League != nil && *stash.League == m.event.Name && stash.AccountName != nil && userMap[*stash.AccountName] != 0 {
		// 	userId := userMap[*stash.AccountName]
		completions := make(map[int]int)
		for _, item := range stash.Items {

			for _, result := range itemChecker.CheckForCompletions(&item) {
				// while syncing we only update the completions for objectives that are desynced
				if syncFinished || utils.Contains(desyncedObjectiveIds, result.ObjectiveId) {
					completions[result.ObjectiveId] += result.Number
				}
			}
		}

		matches = append(matches, m.objectiveMatchService.CreateMatches(completions, userId, stash.StashChangeID, m.event.ID, stashChange.Timestamp)...)
		// }
	}
	return matches
}

func (m *MatchingService) GetReader(desyncedObjectiveIds []int) (*kafka.Reader, error) {
	err := m.objectiveService.StartSync(desyncedObjectiveIds)
	if err != nil {
		return nil, err
	}

	consumer, err := m.objectiveMatchService.GetKafkaConsumer(m.event.ID)
	if err != nil {
		return nil, err
	}

	if len(desyncedObjectiveIds) > 0 {
		consumer.GroupID += 1
		err = m.objectiveMatchService.SaveKafkaConsumerId(consumer)
		if err != nil {
			fmt.Println(err)
		}
	}

	return config.GetReader(m.event.ID, consumer.GroupID)

}

func (m *MatchingService) ProcessStashChanges(itemChecker *parser.ItemChecker, objectives []*repository.Objective) {

	userMap := m.getUserMap()
	desyncedObjectiveIds := make([]int, 0)
	for _, objective := range objectives {
		if (objective.SyncStatus == repository.SyncStatusDesynced || objective.SyncStatus == repository.SyncStatusSyncing) && objective.ObjectiveType == repository.ITEM {
			desyncedObjectiveIds = append(desyncedObjectiveIds, objective.ID)
		}
	}
	reader, err := m.GetReader(desyncedObjectiveIds)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer reader.Close()

	syncing := len(desyncedObjectiveIds) > 0
	matches := make([]*repository.ObjectiveMatch, 0)
	if syncing {
		m.objectiveService.StartSync(desyncedObjectiveIds)
	}
	if m.lastChangeId == nil {
		fmt.Println("No last change id found")
		// this means we dont have any earlier changes, so we assume there are no desynced objectives
		m.objectiveService.SetSynced(desyncedObjectiveIds)
		desyncedObjectiveIds = make([]int, 0)
	}
	fmt.Println("desyncedObjectiveIds", desyncedObjectiveIds)
	count := 0
	for {
		select {
		case <-m.ctx.Done():
			return
		default:
			stashChange, err := m.GetStashChange(reader)
			if err != nil {
				fmt.Println(err)
				return
			}
			// fmt.
			count++
			if count%100 == 0 {
				fmt.Printf("Processed %d changes\n", count)
			}
			fmt.Println("Processing change", stashChange.ChangeID)
			if m.lastChangeId != nil && stashChange.ChangeID == *m.lastChangeId {
				fmt.Println("Reached last change id")
				// once we reach the starting change id the sync is finished
				m.objectiveService.SetSynced(desyncedObjectiveIds)
				syncing = false
			}

			matches = append(matches, m.getMatches(stashChange, userMap, itemChecker, desyncedObjectiveIds)...)
			if !syncing {
				err = m.objectiveMatchService.SaveMatches(matches, desyncedObjectiveIds)
				if err != nil {
					fmt.Println(err)
				}
				desyncedObjectiveIds = make([]int, 0)
				matches = make([]*repository.ObjectiveMatch, 0)
			}

		}
	}
}

func StashLoop(ctx context.Context, poeClient *client.PoEClient, event *repository.Event) error {
	m, err := NewMatchingService(ctx, poeClient, event)
	if err != nil {
		fmt.Println("Failed to create matching service:", err)
		return err
	}

	objectives, err := m.objectiveService.GetObjectivesByEventId(event.ID)
	if err != nil {
		return err
	}
	itemChecker, err := parser.NewItemChecker(objectives)
	if err != nil {
		return err
	}
	go m.ProcessStashChanges(itemChecker, objectives)
	return nil
}
