package service

import (
	"bpl/client"
	"bpl/config"
	"bpl/repository"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type NinjaResponse struct {
	ID                      int    `json:"id"`
	NextChangeID            string `json:"next_change_id"`
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

func (s *StashChangeService) SaveStashChangesConditionally(stashChanges []client.PublicStashChange, message config.StashChangeMessage, eventId int, sendFunc func([]byte) error) error {
	return s.stashChangeRepository.SaveStashChangesConditionally(stashChanges, message, eventId, sendFunc)
}

func (s *StashChangeService) GetNextChangeIdForEvent(event *repository.Event) (string, error) {
	return s.stashChangeRepository.GetNextChangeIdForEvent(event)
}
func (s *StashChangeService) GetCurrentChangeIdForEvent(event *repository.Event) (string, error) {
	return s.stashChangeRepository.GetCurrentChangeIdForEvent(event)
}

func (s *StashChangeService) GetNinjaChangeId() (string, error) {
	response, err := http.Get("https://poe.ninja/api/data/GetStats")
	if err != nil {
		return "", fmt.Errorf("failed to fetch ninja change id: %s", err)
	}
	defer response.Body.Close()
	var ninjaResponse NinjaResponse
	err = json.NewDecoder(response.Body).Decode(&ninjaResponse)
	if err != nil {
		return "", fmt.Errorf("failed to decode ninja change id response: %s", err)
	}
	return ninjaResponse.NextChangeID, nil
}

func (s *StashChangeService) GetInitialChangeId(event *repository.Event) (string, error) {
	stashChange, err := s.GetNextChangeIdForEvent(event)
	if err == nil {
		return stashChange, nil
	}
	log.Print("Initial change id not found, fetching from poe.ninja")
	return s.GetNinjaChangeId()
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
	ninjaId, err := s.GetNinjaChangeId()
	if err != nil {
		return 0
	}
	return ChangeIdToInt(ninjaId) - ChangeIdToInt(changeId)
}
