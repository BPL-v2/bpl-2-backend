package service

import (
	"bpl/repository"
	"slices"
	"time"
)

type ActivityService struct {
	activityRepository *repository.ActivityRepository
}

func NewActivityService() *ActivityService {
	return &ActivityService{
		activityRepository: repository.NewActivityRepository(),
	}
}

type ActivitySession struct {
	Start time.Time
	End   time.Time
}

func (s *ActivityService) CalculateActiveTime(userId int, event *repository.Event, threshold time.Duration) (time.Duration, error) {
	activities, err := s.activityRepository.GetActivity(userId, event.Id)
	if err != nil || len(activities) == 0 {
		return 0, nil
	}
	return determineActiveTime(activities, threshold), nil
}

func determineActiveTime(activities []*repository.Activity, threshold time.Duration) time.Duration {
	slices.SortFunc(activities, func(a, b *repository.Activity) int {
		return a.Time.Compare(b.Time)
	})
	var totalDuration time.Duration
	var sessions []ActivitySession
	sessionStart := activities[0].Time
	sessionEnd := activities[0].Time
	for _, activity := range activities[1:] {
		if activity.Time.Sub(sessionEnd) > threshold {
			sessions = append(sessions, ActivitySession{Start: sessionStart, End: sessionEnd})
			sessionStart = activity.Time
		}
		sessionEnd = activity.Time

	}
	sessions = append(sessions, ActivitySession{Start: sessionStart, End: sessionEnd})
	for _, session := range sessions {
		totalDuration += session.End.Sub(session.Start)
	}
	return totalDuration
}

func (s *ActivityService) CalculateActiveTimesForEvent(event *repository.Event, threshold time.Duration) (map[int]int, error) {
	activities, err := s.activityRepository.GetAllActivitiesForEvent(event.Id)
	if err != nil {
		return nil, err
	}
	userActivities := make(map[int][]*repository.Activity)
	for _, activity := range activities {
		userActivities[activity.UserId] = append(userActivities[activity.UserId], activity)
	}
	activeTimes := make(map[int]int)
	for userId, activities := range userActivities {
		activeTimes[userId] = int(determineActiveTime(activities, threshold).Milliseconds())
	}
	return activeTimes, nil
}

func (s *ActivityService) RecordActivity(userId int, eventId int, timestamp time.Time) error {
	activity := &repository.Activity{
		Time:    timestamp,
		UserId:  userId,
		EventId: eventId,
	}
	return s.activityRepository.SaveActivity(activity)
}

func (s *ActivityService) GetLatestActiveTimestampsForEvent(eventId int) (map[int]time.Time, error) {
	return s.activityRepository.GetLatestActiveTimestampsForEvent(eventId)
}

func (s *ActivityService) CalculateActiveTimesForUsers(userIds []int) (map[int]map[int]time.Duration, error) {
	activities, err := s.activityRepository.GetActivityHistoryForUsers(userIds)
	if err != nil {
		return nil, err
	}
	activeTimes := make(map[int]map[int]time.Duration)
	for userId, eventActivity := range activities {
		activeTimes[userId] = make(map[int]time.Duration)
		for eventId, activities := range eventActivity {
			activeTimes[userId][eventId] = determineActiveTime(activities, 5*time.Minute)
		}
	}
	return activeTimes, nil
}
