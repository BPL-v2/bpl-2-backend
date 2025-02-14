package service

import (
	"bpl/repository"
)

type SignupService struct {
	event_repository  *repository.EventRepository
	signup_repository *repository.SignupRepository
	team_repository   *repository.TeamRepository
}

func NewSignupService() *SignupService {
	return &SignupService{
		signup_repository: repository.NewSignupRepository(),
		event_repository:  repository.NewEventRepository(),
		team_repository:   repository.NewTeamRepository(),
	}
}

func (r *SignupService) CreateSignup(signup *repository.Signup) (*repository.Signup, error) {
	return r.signup_repository.CreateSignup(signup)
}

func (r *SignupService) RemoveSignup(userID int, eventID int) error {
	return r.signup_repository.RemoveSignup(userID, eventID)
}
func (r *SignupService) GetSignupForUser(userID int, eventID int) (*repository.Signup, error) {
	return r.signup_repository.GetSignupForUser(userID, eventID)
}

type SignupWithUser struct {
	Signup   repository.Signup
	TeamUser *repository.TeamUser
}

func (r *SignupService) GetSignupsForEvent(eventId int) (map[int][]*repository.Signup, error) {

	event, err := r.event_repository.GetEventById(eventId, "Teams")
	if err != nil {
		return nil, err
	}
	teamUsers, err := r.team_repository.GetTeamUsersForEvent(event)
	if err != nil {
		return nil, err
	}
	signups, err := r.signup_repository.GetSignupsForEvent(eventId, event.MaxSize)
	if err != nil {
		return nil, err
	}
	userToTeam := make(map[int]int)
	for _, teamUser := range teamUsers {
		userToTeam[teamUser.UserID] = teamUser.TeamID
	}
	teamSignups := make(map[int][]*repository.Signup)
	for _, signup := range signups {
		teamID, ok := userToTeam[signup.UserID]
		if !ok {
			teamID = 0
		}
		teamSignups[teamID] = append(teamSignups[teamID], signup)
	}

	return teamSignups, nil
}
