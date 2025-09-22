package service

import (
	"bpl/repository"
)

type SignupService struct {
	eventRepository  *repository.EventRepository
	signupRepository *repository.SignupRepository
	teamRepository   *repository.TeamRepository
}

func NewSignupService() *SignupService {
	return &SignupService{
		signupRepository: repository.NewSignupRepository(),
		eventRepository:  repository.NewEventRepository(),
		teamRepository:   repository.NewTeamRepository(),
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
