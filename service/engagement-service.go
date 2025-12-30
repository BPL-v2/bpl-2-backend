package service

import (
	"bpl/repository"
)

type EngagementService struct {
	engagementRepository *repository.EngagementRepository
}

func NewEngagementService() *EngagementService {
	return &EngagementService{
		engagementRepository: repository.NewEngagementRepository(),
	}
}

func (s *EngagementService) AddEngagement(name string) error {
	existingEngagement, err := s.engagementRepository.GetEngagementByName(name)
	if err != nil {
		return s.engagementRepository.SaveEngagement(&repository.Engagement{
			Name:   name,
			Number: 1,
		})
	}
	existingEngagement.Number++
	return s.engagementRepository.DB.Save(existingEngagement).Error
}
