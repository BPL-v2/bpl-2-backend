package service

import (
	"bpl/client"
	"bpl/repository"
)

type CachedDataService interface {
	GetLatestScore(eventId int) ([]byte, error)
	GetLatestLadder(eventId int) ([]byte, error)
	GetLatestLadderUnMarshalled(eventId int) (*client.Ladder, error)
	SaveScore(eventId int, data []byte) error
	SaveLadder(eventId int, ladder *client.Ladder) error
}

type CachedDataServiceImpl struct {
	repository repository.CachedDataRepository
}

func NewCachedDataService() CachedDataService {
	return &CachedDataServiceImpl{repository: repository.NewCachedDataRepository()}
}

func (s *CachedDataServiceImpl) GetLatestScore(eventId int) ([]byte, error) {
	return s.repository.GetLatestScore(eventId)
}

func (s *CachedDataServiceImpl) GetLatestLadder(eventId int) ([]byte, error) {
	return s.repository.GetLatestLadder(eventId)
}

func (s *CachedDataServiceImpl) GetLatestLadderUnMarshalled(eventId int) (*client.Ladder, error) {
	return s.repository.GetLatestLadderUnMarshalled(eventId)
}

func (s *CachedDataServiceImpl) SaveScore(eventId int, data []byte) error {
	return s.repository.SaveScore(eventId, data)
}

func (s *CachedDataServiceImpl) SaveLadder(eventId int, ladder *client.Ladder) error {
	return s.repository.SaveLadder(eventId, ladder)
}
