package service

import (
	"bpl/client"
	"bpl/repository"
	"bpl/utils"

	"gorm.io/gorm"
)

type StreamService struct {
	team_repository *repository.TeamRepository
	user_repository *repository.UserRepository
	twitchClient    *client.TwitchClient
	oauthService    *OauthService
}

func NewStreamService(db *gorm.DB) *StreamService {
	oauthService := NewOauthService(db)
	token, _ := oauthService.GetToken("twitch")
	return &StreamService{
		user_repository: repository.NewUserRepository(db),
		team_repository: repository.NewTeamRepository(db),
		twitchClient:    client.NewTwitchClient(*token),
		oauthService:    oauthService,
	}
}

func (e *StreamService) GetStreamsForCurrentEvent() ([]*client.Stream, error) {
	users, err := e.user_repository.GetStreamersForCurrentEvent()
	if err != nil {
		return nil, err
	}
	token, err := e.oauthService.GetToken("twitch")
	if err != nil {
		return nil, err
	}
	e.twitchClient.Token = *token
	return e.twitchClient.GetAllStreams(utils.Map(users, func(user *repository.User) string {
		return user.TwitchID
	}))
}
