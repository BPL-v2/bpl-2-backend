package service

import (
	"bpl/repository"
)

type EngagementService interface {
	AddEngagement(name string) error
}

type EngagementServiceImpl struct {
	engagementRepository repository.EngagementRepository
}

func NewEngagementService() EngagementService {
	return &EngagementServiceImpl{
		engagementRepository: repository.NewEngagementRepository(),
	}
}

func (s *EngagementServiceImpl) AddEngagement(name string) error {
	existingEngagement, err := s.engagementRepository.GetEngagementByName(name)
	if err != nil {
		return s.engagementRepository.SaveEngagement(&repository.Engagement{
			Name:   name,
			Number: 1,
		})
	}
	existingEngagement.Number++
	return s.engagementRepository.UpdateEngagement(existingEngagement)
}
