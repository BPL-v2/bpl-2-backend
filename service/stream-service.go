package service

import (
	"bpl/client"
	"bpl/repository"
	"bpl/utils"
	"log"
)

type StreamService struct {
	teamRepository   *repository.TeamRepository
	userRepository   *repository.UserRepository
	ladderRepository *repository.LadderRepository
	eventRepository  *repository.EventRepository
	twitchClient     *client.TwitchClient
	oauthService     *OauthService
}

func NewStreamService() *StreamService {
	oauthService := NewOauthService()
	s := &StreamService{
		userRepository:   repository.NewUserRepository(),
		teamRepository:   repository.NewTeamRepository(),
		ladderRepository: repository.NewLadderRepository(),
		eventRepository:  repository.NewEventRepository(),
		oauthService:     oauthService,
	}
	token, err := oauthService.GetApplicationToken(repository.ProviderTwitch)
	if err != nil {
		log.Printf("Failed to get twitch token: %v", err)
		return s
	}
	s.twitchClient = client.NewTwitchClient(token)
	return s

}

func (e *StreamService) GetStreamsForCurrentEvent() ([]*client.TwitchStream, error) {
	streamers, err := e.userRepository.GetStreamersForCurrentEvent()
	if err != nil {
		return nil, err
	}
	token, err := e.oauthService.GetApplicationToken(repository.ProviderTwitch)
	if err != nil {
		return nil, err
	}
	e.twitchClient.Token = token

	userMap := make(map[string]int)
	for _, streamer := range streamers {
		userMap[streamer.TwitchId] = streamer.UserId
	}
	event, err := e.eventRepository.GetCurrentEvent()
	if err != nil {
		return nil, err
	}
	ladderEntries, err := e.ladderRepository.GetLadderForEvent(event.Id)
	if err != nil {
		return nil, err
	}
	for _, entry := range ladderEntries {
		if entry.TwitchAccount != nil && entry.UserId != nil {
			userMap[*entry.TwitchAccount] = *entry.UserId
		}
	}
	streams, err := e.twitchClient.GetAllStreams(utils.Keys(userMap))
	if err != nil {
		return nil, err
	}
	for _, stream := range streams {
		if userId, ok := userMap[stream.UserId]; ok {
			stream.BackendUserId = userId
		}
	}
	return streams, nil
}
