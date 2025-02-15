package service

import (
	"bpl/repository"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type StashChangeService struct {
	stashChangeRepository *repository.StashChangeRepository
}

func NewStashChangeService() *StashChangeService {
	return &StashChangeService{
		stashChangeRepository: repository.NewStashChangeRepository(),
	}
}

func (s *StashChangeService) SaveStashChangesConditionally(stashChanges []*repository.StashChange, condFunc func() error) error {
	return s.stashChangeRepository.SaveStashChangesConditionally(stashChanges, condFunc)
}

func (s *StashChangeService) GetLatestStashChangeForEvent(event *repository.Event) (*repository.StashChange, error) {
	return s.stashChangeRepository.GetLatestStashChangeForEvent(event)
}

func (s *StashChangeService) GetInitialChangeId(event *repository.Event) (*repository.StashChange, error) {
	stashChange, err := s.GetLatestStashChangeForEvent(event)
	if err == nil {
		return stashChange, nil
	}
	fmt.Println("Initial change id not found, fetching from poe.ninja")

	response, err := http.Get("https://poe.ninja/api/data/GetStats")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch initial change id: %s", err)
	}
	defer response.Body.Close()
	var ninjaResponse NinjaResponse
	err = json.NewDecoder(response.Body).Decode(&ninjaResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to decode initial change id response: %s", err)
	}
	return &repository.StashChange{
		NextChangeID: ninjaResponse.NextChangeID,
		IntChangeID:  0,
		EventID:      event.ID,
		Timestamp:    time.Now(),
	}, nil

}
