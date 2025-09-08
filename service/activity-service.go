package service

import (
	"bpl/repository"
	"fmt"
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

func (s *ActivityService) CalculateActiveTime(userId int, event *repository.Event, threshold time.Duration) (time.Duration, error) {

	// inactiveTime, err := s.CalculateInactiveTime(userId, event, threshold)
	// seconds := event.EventEndTime.Unix() - event.EventStartTime.Unix() - int64(inactiveTime.Seconds())
	// return time.Duration(seconds) * time.Second, err

	activities, err := s.activityRepository.GetActivity(userId, event.Id)
	if err != nil || len(activities) == 0 {
		fmt.Println("Error fetching activities or no activities found:", err)
		return 0, nil
	}
	var totalDuration time.Duration
	sessionStart := activities[0].Time.Add(-threshold)
	if sessionStart.Before(event.EventStartTime) {
		sessionStart = event.EventStartTime
	}
	lastActivityTime := activities[0].Time

	for _, activity := range activities[1:] {
		if activity.Time.Sub(lastActivityTime) > threshold {
			sessionEnd := lastActivityTime.Add(threshold)
			if sessionEnd.After(event.EventEndTime) {
				sessionEnd = event.EventEndTime
			}
			totalDuration += sessionEnd.Sub(sessionStart)
			sessionStart = activity.Time.Add(-threshold)
			if sessionStart.Before(event.EventStartTime) {
				sessionStart = event.EventStartTime
			}
		}
		lastActivityTime = activity.Time
	}

	sessionEnd := lastActivityTime.Add(threshold)
	if sessionEnd.After(event.EventEndTime) {
		sessionEnd = event.EventEndTime
	}
	totalDuration += sessionEnd.Sub(sessionStart)
	return totalDuration, nil
}

func (s *ActivityService) CalculateInactiveTime(userId int, event *repository.Event, minInactivityWindow time.Duration) (time.Duration, error) {
	activities, err := s.activityRepository.GetActivity(userId, event.Id)
	if err != nil || len(activities) == 0 {
		fmt.Println("Error fetching activities or no activities found:", err)
		// If no activities, the entire event duration is inactive time
		if err == nil {
			return event.EventEndTime.Sub(event.EventStartTime), nil
		}
		return 0, err
	}

	var totalInactiveTime time.Duration

	// Check for inactivity before the first activity
	if activities[0].Time.Sub(event.EventStartTime) >= minInactivityWindow {
		totalInactiveTime += activities[0].Time.Sub(event.EventStartTime)
	}

	// Check for inactivity between activities
	for i := 1; i < len(activities); i++ {
		inactivityPeriod := activities[i].Time.Sub(activities[i-1].Time)
		if inactivityPeriod >= minInactivityWindow {
			totalInactiveTime += inactivityPeriod
		}
	}

	// Check for inactivity after the last activity
	lastActivity := activities[len(activities)-1]
	if event.EventEndTime.Sub(lastActivity.Time) >= minInactivityWindow {
		totalInactiveTime += event.EventEndTime.Sub(lastActivity.Time)
	}

	return totalInactiveTime, nil
}

func (s *ActivityService) RecordActivity(userId int, eventId int, timestamp time.Time) error {
	activity := &repository.Activity{
		Time:    timestamp,
		UserId:  userId,
		EventId: eventId,
	}
	return s.activityRepository.SaveActivity(activity)
}
