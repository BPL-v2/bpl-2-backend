package service

import (
	"bpl/client"
	"bpl/repository"
	"bpl/utils"
	"log"
)

type StreamService struct {
	team_repository *repository.TeamRepository
	user_repository *repository.UserRepository
	twitchClient    *client.TwitchClient
	oauthService    *OauthService
}

func NewStreamService() *StreamService {
	oauthService := NewOauthService()
	s := &StreamService{
		user_repository: repository.NewUserRepository(),
		team_repository: repository.NewTeamRepository(),
		oauthService:    oauthService,
	}
	token, err := oauthService.GetApplicationToken("twitch")
	if err != nil {
		log.Fatalf("Failed to get twitch token: %v", err)
		return s
	}
	s.twitchClient = client.NewTwitchClient(*token)
	return s

}

func (e *StreamService) GetStreamsForCurrentEvent() ([]*client.TwitchStream, error) {
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
