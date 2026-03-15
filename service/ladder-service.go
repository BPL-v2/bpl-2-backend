package service

import (
	"bpl/client"
	"bpl/repository"
)

type LadderService interface {
	UpsertLadder(ladder []*client.LadderEntry, eventId int, playerMap map[string]int) error
	GetLadderForEvent(eventId int) ([]*repository.LadderEntry, error)
}

type LadderServiceImpl struct {
	ladderRepository repository.LadderRepository
}

func NewLadderService() LadderService {
	return &LadderServiceImpl{
		ladderRepository: repository.NewLadderRepository(),
	}
}

func (s *LadderServiceImpl) UpsertLadder(ladder []*client.LadderEntry, eventId int, playerMap map[string]int) error {
	return s.ladderRepository.UpsertLadder(ladder, eventId, playerMap)
}

func (s *LadderServiceImpl) GetLadderForEvent(eventId int) ([]*repository.LadderEntry, error) {
	return s.ladderRepository.GetLadderForEvent(eventId)
}
