package scoring

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
	"math/rand/v2"
	"strconv"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
)

type StashChange struct {
	Stashes      []client.PublicStashChange
	ChangeID     string
	NextChangeID string
	Timestamp    time.Time
}

type MatchingService struct {
	ctx                   context.Context
	poeClient             *client.PoEClient
	objectiveMatchService *service.ObjectiveMatchService
	objectiveService      *service.ObjectiveService
	stashChannel          chan StashChange
	lastChangeId          int64
	event                 *repository.Event
}

func NewMatchingService(ctx context.Context, poeClient *client.PoEClient, event *repository.Event) (*MatchingService, error) {
	objectiveMatchService := service.NewObjectiveMatchService()
	objectiveService := service.NewObjectiveService()
	stashService := service.NewStashChangeService()
	stashChange, err := stashService.GetInitialChangeId(event)
	if err != nil {
		return nil, err
	}
	return &MatchingService{
		poeClient:             poeClient,
		objectiveMatchService: objectiveMatchService,
		objectiveService:      objectiveService,
		stashChannel:          make(chan StashChange, 10000),
		lastChangeId:          stashChange.IntChangeID,
		event:                 event,
		ctx:                   ctx,
	}, nil
}

func (m *MatchingService) GetStashChange(reader *kafka.Reader) (stashChange StashChange, err error) {
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

func (m *MatchingService) getMatches(stashChange StashChange, userMap map[string]int, itemChecker *parser.ItemChecker, desyncedObjectiveIds []int) []*repository.ObjectiveMatch {
	matches := make([]*repository.ObjectiveMatch, 0)
	syncFinished := len(desyncedObjectiveIds) == 0
	intChangeId, err := stashChangeToInt(stashChange.ChangeID)
	if err != nil {
		fmt.Println(err)
		return matches
	}
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
		matches = append(matches, m.objectiveMatchService.CreateMatches(completions, userId, intChangeId, stash.ID, m.event.ID, stashChange.Timestamp)...)
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
	if m.lastChangeId == 0 {
		// this means we dont have any earlier changes, so we assume there are no desynced objectives
		m.objectiveService.SetSynced(desyncedObjectiveIds)
		desyncedObjectiveIds = make([]int, 0)
	}

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
			fmt.Printf("Processing change %s\n", stashChange.ChangeID)
			intChangeId, err := stashChangeToInt(stashChange.ChangeID)
			if err != nil {
				fmt.Println(err)
				return
			}

			if intChangeId == m.lastChangeId {
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

func stashChangeToInt(change string) (int64, error) {
	sum := int64(0)
	for _, part := range strings.Split(change, "-") {
		value, err := strconv.ParseInt(part, 10, 64)
		if err != nil {
			return 0, err
		}
		sum += value
	}
	return sum, nil
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
