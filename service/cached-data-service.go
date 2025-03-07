package service

import (
	"bpl/client"
	"bpl/repository"
)

type CachedDataService struct {
	repository *repository.CachedDataRepository
}

func NewCachedDataService() *CachedDataService {
	return &CachedDataService{repository: repository.NewCachedDataRepository()}
}

func (s *CachedDataService) GetLatestScore(eventId int) ([]byte, error) {
	return s.repository.GetLatestScore(eventId)
}

func (s *CachedDataService) GetLatestLadder(eventId int) ([]byte, error) {
	return s.repository.GetLatestLadder(eventId)
}

func (s *CachedDataService) GetLatestLadderUnMarshalled(eventId int) (*client.Ladder, error) {
	return s.repository.GetLatestLadderUnMarshalled(eventId)
}

func (s *CachedDataService) SaveScore(eventId int, data []byte) error {
	return s.repository.SaveScore(eventId, data)
}

func (s *CachedDataService) SaveLadder(eventId int, ladder *client.Ladder) error {
	return s.repository.SaveLadder(eventId, ladder)
}
