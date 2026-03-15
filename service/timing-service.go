package service

import (
	"bpl/repository"
	"time"
)

type TimingService interface {
	GetTimings() (map[repository.TimingKey]time.Duration, error)
	SaveTimings(timings []*repository.Timing) error
}

type TimingServiceImpl struct {
	timingRepository repository.TimingRepository
}

func NewTimingService() TimingService {
	return &TimingServiceImpl{
		timingRepository: repository.NewTimingRepository(),
	}
}

func (s *TimingServiceImpl) GetTimings() (map[repository.TimingKey]time.Duration, error) {
	return s.timingRepository.GetTimings()
}

func (s *TimingServiceImpl) SaveTimings(timings []*repository.Timing) error {
	return s.timingRepository.SaveTimings(timings)
}
