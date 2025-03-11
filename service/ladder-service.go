package service

import (
	"bpl/client"
	"bpl/repository"
)

type LadderService struct {
	ladderRepository *repository.LadderRepository
}

func NewLadderService() *LadderService {
	return &LadderService{
		ladderRepository: repository.NewLadderRepository(),
	}
}

func (s *LadderService) UpsertLadder(ladder []*client.LadderEntry, eventId int, playerMap map[string]int) error {
	return s.ladderRepository.UpsertLadder(ladder, eventId, playerMap)
}

func (s *LadderService) GetLadderForEvent(eventId int) ([]*repository.LadderEntry, error) {
	return s.ladderRepository.GetLadderForEvent(eventId)
}
