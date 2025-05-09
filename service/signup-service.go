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

func (r *SignupService) CreateSignup(signup *repository.Signup) (*repository.Signup, error) {
	return r.signupRepository.CreateSignup(signup)
}

func (r *SignupService) RemoveSignup(userId int, eventId int) error {
	return r.signupRepository.RemoveSignup(userId, eventId)
}
func (r *SignupService) GetSignupForUser(userId int, eventId int) (*repository.Signup, error) {
	return r.signupRepository.GetSignupForUser(userId, eventId)
}

type SignupWithUser struct {
	Signup   repository.Signup
	TeamUser *repository.TeamUser
}

func (r *SignupService) GetSignupsForEvent(event *repository.Event) ([]*repository.Signup, error) {
	return r.signupRepository.GetSignupsForEvent(event.Id)
}
