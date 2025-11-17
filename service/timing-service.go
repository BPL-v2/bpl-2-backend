package service

import (
	"bpl/repository"
	"time"
)

type TimingService struct {
	timingRepository *repository.TimingRepository
}

func NewTimingService() *TimingService {
	return &TimingService{
		timingRepository: repository.NewTimingRepository(),
	}
}

func (s *TimingService) GetTimings() (map[repository.TimingKey]time.Duration, error) {
	return s.timingRepository.GetTimings()
}

func (s *TimingService) SaveTimings(timings []*repository.Timing) error {
	return s.timingRepository.SaveTimings(timings)
}
