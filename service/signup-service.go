package service

import (
	"bpl/repository"
	"time"
)

type SignupService struct {
	eventRepository     *repository.EventRepository
	signupRepository    *repository.SignupRepository
	teamRepository      *repository.TeamRepository
	activityService     *ActivityService
	characterRepository *repository.CharacterRepository
}

func NewSignupService() *SignupService {
	return &SignupService{
		signupRepository:    repository.NewSignupRepository(),
		eventRepository:     repository.NewEventRepository(),
		teamRepository:      repository.NewTeamRepository(),
		activityService:     NewActivityService(),
		characterRepository: repository.NewCharacterRepository(),
	}
}

func (r *SignupService) SaveSignup(signup *repository.Signup) (*repository.Signup, error) {
	return r.signupRepository.SaveSignup(signup)
}

func (r *SignupService) RemoveSignupForUser(userId int, eventId int) error {
	err := r.teamRepository.RemoveUserForEvent(userId, eventId)
	if err != nil {
		return err
	}
	return r.signupRepository.RemoveSignupForUser(userId, eventId)
}

func (r *SignupService) GetSignupForUser(userId int, eventId int) (*repository.Signup, error) {
	return r.signupRepository.GetSignupForUser(userId, eventId)
}

func (r *SignupService) ReportPlaytime(userId int, eventId int, actualPlaytime int) (*repository.Signup, error) {
	signup, err := r.signupRepository.GetSignupForUser(userId, eventId)
	if err != nil {
		return nil, err
	}
	signup.ActualPlayTime = actualPlaytime
	return r.signupRepository.SaveSignup(signup)
}

type SignupWithUser struct {
	Signup   repository.Signup
	TeamUser *repository.TeamUser
}

func (r *SignupService) GetSignupsForEvent(event *repository.Event) ([]*repository.Signup, error) {
	return r.signupRepository.GetSignupsForEvent(event.Id)
}

func (r *SignupService) GetExtendedSignupsForEvent(event *repository.Event) (
	[]*repository.Signup, map[int]map[int]time.Duration, map[int]map[int]int, error) {
	signups, err := r.GetSignupsForEvent(event)
	if err != nil {
		return nil, nil, nil, err
	}
	userIds := make([]int, 0)
	for _, signup := range signups {
		userIds = append(userIds, signup.UserId)
	}
	userEventActivityCount, err := r.activityService.CalculateActiveTimesForUsers(userIds)
	if err != nil {
		return nil, nil, nil, err
	}
	highestCharacterLevels, err := r.characterRepository.GetHighestCharacterLevelForEventsForUsers(userIds)
	if err != nil {
		return nil, nil, nil, err
	}
	return signups, userEventActivityCount, highestCharacterLevels, nil
}
