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
	token, _ := oauthService.GetApplicationToken("twitch")
	return &StreamService{
		user_repository: repository.NewUserRepository(db),
		team_repository: repository.NewTeamRepository(db),
		twitchClient:    client.NewTwitchClient(*token),
		oauthService:    oauthService,
	}
}

func (e *StreamService) GetStreamsForCurrentEvent() ([]*client.Stream, error) {
	streamers, err := e.user_repository.GetStreamersForCurrentEvent()
	if err != nil {
		return nil, err
	}
	token, err := e.oauthService.GetApplicationToken("twitch")
	if err != nil {
		return nil, err
	}
	e.twitchClient.Token = *token

	userMap := make(map[string]int)
	for _, streamer := range streamers {
		userMap[streamer.TwitchID] = streamer.UserID
	}

	streams, err := e.twitchClient.GetAllStreams(utils.Map(streamers, func(user *repository.Streamer) string {
		return user.TwitchID
	}))
	if err != nil {
		return nil, err
	}
	for _, stream := range streams {
		if userID, ok := userMap[stream.UserID]; ok {
			stream.BackendUserId = userID
		}
	}
	return streams, nil
}
