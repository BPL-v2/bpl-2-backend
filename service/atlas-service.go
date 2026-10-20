package service

import (
	"bpl/client"
	"bpl/repository"
	"slices"
	"sync"
	"time"
)

const (
	atlasMilestoneFirstPoint   = 1
	atlasMilestoneEightyPoints = 80
	// atlasMilestoneSessionGap is the longest pause between two activity entries that still counts as
	// playing, the same as the default of the activity endpoint.
	atlasMilestoneSessionGap = 30 * time.Minute
	// atlasMilestoneRecentEvents is how many of the latest events with atlas history are looked at.
	atlasMilestoneRecentEvents = 3
)

// AtlasMilestones is how many hours a player was actively playing (breaks do not count) before they
// first had a given amount of atlas points allocated. A milestone the player did not reach is nil.
type AtlasMilestones struct {
	FirstPointActiveHours   *float64
	EightyPointsActiveHours *float64
}

type AtlasService interface {
	SaveAtlasTrees(userId int, eventId int, treeHashes []client.LeagueAccountAtlasPassiveTree) error
	GetLatestAtlasesForEventAndTeam(eventId int, teamId int) (atlas []*repository.AtlasTree, err error)
	GetAtlasesForEventAndUser(userId int, eventId int) (atlas []*repository.AtlasTree, err error)
	// GetAtlasMilestonesInRecentEvents returns for the given users the active time to atlas milestones
	// in the latest events before currentEvent that have an atlas history (user id -> event id -> milestones).
	GetAtlasMilestonesInRecentEvents(currentEvent *repository.Event, allEvents []*repository.Event, userIds []int) (map[int]map[int]AtlasMilestones, error)
}

type AtlasServiceImpl struct {
	atlasRepository repository.AtlasRepository
	latestAtlas     map[int]map[int]map[int][32]byte
}

func NewAtlasService() AtlasService {
	return &AtlasServiceImpl{
		atlasRepository: repository.NewAtlasRepository(),
		latestAtlas:     make(map[int]map[int]map[int][32]byte),
	}
}

func (a *AtlasServiceImpl) initCache(userId int, eventId int) error {
	if a.latestAtlas[eventId] == nil {
		trees, err := a.atlasRepository.GetLatestTreesForEvent(eventId)
		if err != nil {
			return err
		}
		a.latestAtlas[eventId] = make(map[int]map[int][32]byte)
		for _, tree := range trees {
			if a.latestAtlas[eventId][tree.UserID] == nil {
				a.latestAtlas[eventId][tree.UserID] = map[int][32]byte{0: {}, 1: {}, 2: {}}
			}
			a.latestAtlas[eventId][tree.UserID][tree.Index] = tree.Nodes.GetHash()
		}
	}
	if a.latestAtlas[eventId][userId] == nil {
		a.latestAtlas[eventId][userId] = map[int][32]byte{0: {}, 1: {}, 2: {}}
	}
	return nil
}

func (a *AtlasServiceImpl) SaveAtlasTrees(userId int, eventId int, trees []client.LeagueAccountAtlasPassiveTree) error {
	if err := a.initCache(userId, eventId); err != nil {
		return err
	}
	hashValues := a.latestAtlas[eventId][userId]
	for index, tree := range trees {
		hashes := []int{}
		for _, hash := range tree.Hashes {
			hashes = append(hashes, hash)
		}
		if repository.PassiveNodes(hashes).GetHash() != hashValues[index] {
			err := a.atlasRepository.CreateAtlasTree(userId, eventId, index, hashes)
			if err != nil {
				return err
			}
			hashValues[index] = repository.PassiveNodes(hashes).GetHash()
		}
	}
	return nil
}

func (a *AtlasServiceImpl) GetLatestAtlasesForEventAndTeam(eventId int, teamId int) (atlas []*repository.AtlasTree, err error) {
	return a.atlasRepository.GetLatestAtlasesForEventAndTeam(eventId, teamId)
}

func (a *AtlasServiceImpl) GetAtlasesForEventAndUser(userId int, eventId int) (atlas []*repository.AtlasTree, err error) {
	return a.atlasRepository.GetAtlasesForEventAndUser(eventId, userId)
}

// recentAtlasHistoryEventIds returns the ids of up to n events that started before currentEvent and have an
// atlas history, the most recent first.
func recentAtlasHistoryEventIds(currentEvent *repository.Event, allEvents []*repository.Event, withHistory []int, n int) []int {
	candidates := make([]*repository.Event, 0)
	for _, event := range allEvents {
		if slices.Contains(withHistory, event.Id) && event.EventStartTime.Before(currentEvent.EventStartTime) {
			candidates = append(candidates, event)
		}
	}
	slices.SortFunc(candidates, func(a, b *repository.Event) int {
		return b.EventStartTime.Compare(a.EventStartTime)
	})
	ids := make([]int, 0, n)
	for _, event := range candidates[:min(n, len(candidates))] {
		ids = append(ids, event.Id)
	}
	return ids
}

func (a *AtlasServiceImpl) GetAtlasMilestonesInRecentEvents(
	currentEvent *repository.Event, allEvents []*repository.Event, userIds []int,
) (map[int]map[int]AtlasMilestones, error) {
	withHistory, err := a.atlasRepository.GetEventIdsWithAtlasHistory()
	if err != nil {
		return nil, err
	}
	eventIds := recentAtlasHistoryEventIds(currentEvent, allEvents, withHistory, atlasMilestoneRecentEvents)

	type eventResult struct {
		eventId int
		times   []*repository.AtlasMilestoneActiveTime
		err     error
	}
	results := make([]eventResult, len(eventIds))
	var wg sync.WaitGroup
	for i, eventId := range eventIds {
		wg.Add(1)
		go func() {
			defer wg.Done()
			times, err := a.atlasRepository.GetActiveSecondsAtAtlasMilestones(
				eventId, userIds, []int{atlasMilestoneFirstPoint, atlasMilestoneEightyPoints}, atlasMilestoneSessionGap)
			results[i] = eventResult{eventId: eventId, times: times, err: err}
		}()
	}
	wg.Wait()

	milestones := make(map[int]map[int]AtlasMilestones)
	for _, result := range results {
		if result.err != nil {
			return nil, result.err
		}
		for _, reached := range result.times {
			hours := reached.ActiveSeconds / time.Hour.Seconds()
			if milestones[reached.UserId] == nil {
				milestones[reached.UserId] = make(map[int]AtlasMilestones)
			}
			userMilestones := milestones[reached.UserId][result.eventId]
			switch reached.Points {
			case atlasMilestoneFirstPoint:
				userMilestones.FirstPointActiveHours = &hours
			case atlasMilestoneEightyPoints:
				userMilestones.EightyPointsActiveHours = &hours
			}
			milestones[reached.UserId][result.eventId] = userMilestones
		}
	}
	return milestones, nil
}
