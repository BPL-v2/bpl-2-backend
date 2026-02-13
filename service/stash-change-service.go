package service

import (
	"bpl/repository"
	"bpl/utils"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type NinjaResponse struct {
	Id                      int    `json:"id"`
	NextChangeId            string `json:"next_change_id"`
	APIBytesDownloaded      int    `json:"api_bytes_downloaded"`
	StashTabsProcessed      int    `json:"stash_tabs_processed"`
	APICalls                int    `json:"api_calls"`
	CharacterBytesDl        int    `json:"character_bytes_downloaded"`
	CharacterAPICalls       int    `json:"character_api_calls"`
	LadderBytesDl           int    `json:"ladder_bytes_downloaded"`
	LadderAPICalls          int    `json:"ladder_api_calls"`
	PoBCharactersCalculated int    `json:"pob_characters_calculated"`
	OAuthFlows              int    `json:"oauth_flows"`
}

type StashChangeService struct {
	stashChangeRepository *repository.StashChangeRepository
}

func NewStashChangeService() *StashChangeService {
	return &StashChangeService{
		stashChangeRepository: repository.NewStashChangeRepository(),
	}
}

func (s *StashChangeService) GetLatestTimestamp(eventId int) (time.Time, error) {
	return s.stashChangeRepository.GetLatestTimestamp(eventId)
}

func (s *StashChangeService) SaveStashChangesConditionally(message repository.StashChangeMessage, eventId int, sendFunc func([]byte) error) error {
	return s.stashChangeRepository.SaveStashChangesConditionally(message, eventId, sendFunc)
}

func (s *StashChangeService) GetNextChangeIdForEvent(event *repository.Event) (string, error) {
	changeId, err := s.stashChangeRepository.GetChangeIdForEvent(event)
	if err != nil {
		return "", fmt.Errorf("failed to get next change id for event %d: %s", event.Id, err)
	}
	return changeId.NextChangeId, nil
}
func (s *StashChangeService) GetCurrentChangeIdForEvent(event *repository.Event) (*repository.ChangeId, error) {
	changeId, err := s.stashChangeRepository.GetChangeIdForEvent(event)
	if err != nil {
		return nil, fmt.Errorf("failed to get current change id for event %d: %s", event.Id, err)
	}
	return &changeId, nil
}

func GetNinjaChangeId() (string, error) {
	response, err := http.Get("https://poe.ninja/poe1/api/data/stats")
	if err != nil {
		return "", fmt.Errorf("failed to fetch ninja change id: %s", err)
	}
	defer utils.Closer(response.Body)()
	var ninjaResponse NinjaResponse
	err = json.NewDecoder(response.Body).Decode(&ninjaResponse)
	if err != nil {
		return "", fmt.Errorf("failed to decode ninja change id response: %s", err)
	}
	return ninjaResponse.NextChangeId, nil
}

func (s *StashChangeService) GetInitialChangeId(event *repository.Event) (string, error) {
	stashChange, err := s.GetNextChangeIdForEvent(event)
	if err == nil {
		return stashChange, nil
	}
	log.Print("Initial change id not found, fetching from poe.ninja")
	return GetNinjaChangeId()
}

func ChangeIdToInt(changeId string) int {
	sum := 0
	for _, str := range strings.Split(changeId, "-") {
		i, err := strconv.Atoi(str)
		if err != nil {
			return 0
		}
		sum += i
	}
	return sum
}

func (s *StashChangeService) GetNinjaDifference(changeId string) int {
	ninjaId, err := GetNinjaChangeId()
	if err != nil {
		return 0
	}
	return ChangeIdToInt(ninjaId) - ChangeIdToInt(changeId)
}
