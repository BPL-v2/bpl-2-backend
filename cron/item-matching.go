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
	"log"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/segmentio/kafka-go"
)

type MatchingService struct {
	ctx                   context.Context
	objectiveMatchService *service.ObjectiveMatchService
	objectiveService      *service.ObjectiveService
	userService           *service.UserService
	lastChangeId          *string
	event                 *repository.Event
}

var teamMatchesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "team_matches_total",
	Help: "The number of matches for each team",
}, []string{"team"})

func NewMatchingService(ctx context.Context, poeClient *client.PoEClient, event *repository.Event) (*MatchingService, error) {
	objectiveMatchService := service.NewObjectiveMatchService()
	objectiveService := service.NewObjectiveService()
	userService := service.NewUserService()
	matchingService := &MatchingService{
		objectiveMatchService: objectiveMatchService,
		objectiveService:      objectiveService,
		userService:           userService,
		event:                 event,
		ctx:                   ctx,
	}
	changeId, err := service.NewStashChangeService().GetCurrentChangeIdForEvent(event)
	if err == nil {
		log.Printf("Last change id: %s", changeId)
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

func (m *MatchingService) getItemMatches(stashChange config.StashChangeMessage, userMap map[string]int, teamMap map[string]string, itemChecker *parser.ItemChecker, desyncedObjectiveIds []int) []*repository.ObjectiveMatch {
	matches := make([]*repository.ObjectiveMatch, 0)
	syncFinished := len(desyncedObjectiveIds) == 0
	for _, stash := range stashChange.Stashes {

		if stash.League != nil && *stash.League == m.event.Name && stash.AccountName != nil && userMap[*stash.AccountName] != 0 {
			userId := userMap[*stash.AccountName]
			completions := make(map[int]int)
			for _, item := range stash.Items {

				for _, result := range itemChecker.CheckForCompletions(&item) {
					// while syncing we only update the completions for objectives that are desynced
					if syncFinished || utils.Contains(desyncedObjectiveIds, result.ObjectiveId) {
						completions[result.ObjectiveId] += result.Number
					}
				}
			}
			teamMatchesTotal.WithLabelValues(teamMap[*stash.AccountName]).Add(float64(len(completions)))
			matches = append(matches, m.objectiveMatchService.CreateItemMatches(completions, userId, stash.StashChangeId, m.event.Id, stashChange.Timestamp)...)
		}
	}
	return matches
}

func (m *MatchingService) GetReader(desyncedObjectiveIds []int) (*kafka.Reader, error) {
	err := m.objectiveService.StartSync(desyncedObjectiveIds)
	if err != nil {
		return nil, err
	}

	consumer, err := m.objectiveMatchService.GetKafkaConsumer(m.event.Id)
	if err != nil {
		return nil, err
	}

	if len(desyncedObjectiveIds) > 0 {
		consumer.GroupId += 1
		err = m.objectiveMatchService.SaveKafkaConsumerId(consumer)
		if err != nil {
			log.Print(err)
		}
	}

	return config.GetReader(m.event.Id, consumer.GroupId)

}

func (m *MatchingService) ProcessStashChanges(itemChecker *parser.ItemChecker, objectives []*repository.Objective) {

	users, err := m.userService.GetUsersForEvent(m.event.Id)
	if err != nil {
		log.Fatal(err)
		return
	}
	userMap := make(map[string]int)
	teamMap := make(map[string]string)
	teamNames := make(map[int]string)
	for _, team := range m.event.Teams {
		teamNames[team.Id] = team.Name
	}

	for _, user := range users {
		userMap[user.AccountName] = user.UserId
		teamMap[user.AccountName] = teamNames[user.TeamId]
	}
	desyncedObjectiveIds := make([]int, 0)
	for _, objective := range objectives {
		if (objective.SyncStatus == repository.SyncStatusDesynced || objective.SyncStatus == repository.SyncStatusSyncing) && objective.ObjectiveType == repository.ITEM {
			desyncedObjectiveIds = append(desyncedObjectiveIds, objective.Id)
		}
	}
	reader, err := m.GetReader(desyncedObjectiveIds)
	if err != nil {
		log.Fatal(err)
		return
	}
	defer reader.Close()

	syncing := len(desyncedObjectiveIds) > 0
	matches := make([]*repository.ObjectiveMatch, 0)
	if syncing {
		m.objectiveService.StartSync(desyncedObjectiveIds)
		log.Printf("Catching up on desynced objectives: %v", desyncedObjectiveIds)
	}
	if m.lastChangeId == nil {
		log.Println("No last change id found")
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
				log.Fatal(err)
				return
			}
			if m.lastChangeId != nil && stashChange.ChangeId == *m.lastChangeId {
				log.Println("Sync finished")
				// once we reach the starting change id the sync is finished
				m.objectiveService.SetSynced(desyncedObjectiveIds)
				syncing = false
			}

			// this is used for testing, remove this once we have actual users
			// for _, stash := range stashChange.Stashes {
			// 	if stash.League != nil && *stash.League == m.event.Name {
			// 		if stash.AccountName != nil && userMap[*stash.AccountName] == 0 {
			// 			user, err := m.userService.AddUserFromStashchange(*stash.AccountName, m.event)
			// 			if err != nil {
			// 				fmt.Println(err)
			// 				continue
			// 			}
			// 			userMap[*stash.AccountName] = user.Id
			// 		}
			// 	}
			// }

			matches = append(matches, m.getItemMatches(stashChange, userMap, teamMap, itemChecker, desyncedObjectiveIds)...)
			if !syncing {
				err = m.objectiveMatchService.SaveMatches(matches, desyncedObjectiveIds)
				if err != nil {
					log.Fatal(err)
				}
				desyncedObjectiveIds = make([]int, 0)
				matches = make([]*repository.ObjectiveMatch, 0)
			}

		}
	}
}

func StashEvaluationLoop(ctx context.Context, poeClient *client.PoEClient, event *repository.Event) error {
	m, err := NewMatchingService(ctx, poeClient, event)
	if err != nil {
		log.Fatal("Failed to create matching service:", err)
		return err
	}

	objectives, err := m.objectiveService.GetObjectivesByEventId(event.Id)
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
