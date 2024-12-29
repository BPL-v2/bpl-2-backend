package scoring

import (
	"bpl/client"
	"bpl/parser"
	"bpl/repository"
	"bpl/service"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
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

type StashChange struct {
	Stashes  []client.PublicStashChange
	ChangeID string
}

func getInitialChangeId() (string, error) {
	response, err := http.Get("https://poe.ninja/api/data/GetStats")
	if err != nil {
		return "", fmt.Errorf("failed to fetch initial change id: %s", err)
	}
	defer response.Body.Close()
	var ninjaResponse NinjaResponse
	err = json.NewDecoder(response.Body).Decode(&ninjaResponse)
	if err != nil {
		return "", fmt.Errorf("failed to decode initial change id response: %s", err)
	}
	return ninjaResponse.NextChangeID, nil
}

func FetchStashChanges(poeClient *client.PoEClient, endTime time.Time, stashChannel chan StashChange) error {
	token := os.Getenv("POE_CLIENT_TOKEN")
	if token == "" {
		return fmt.Errorf("POE_CLIENT_TOKEN environment variable not set")
	}
	changeId, err := getInitialChangeId()
	if err != nil {
		fmt.Println(err)
		return err
	}
	fmt.Println("Initial change id:", changeId)
	for time.Now().Before(endTime) {
		response, err := poeClient.GetPublicStashes(token, "pc", changeId)
		if err != nil {
			if err.StatusCode == 429 {
				fmt.Println(err.ResponseHeaders)
				retryAfter, err := strconv.Atoi(err.ResponseHeaders.Get("Retry-After"))
				if err != nil {
					fmt.Println(err)
					return fmt.Errorf("failed to parse Retry-After header: %s", err)
				}
				<-time.After((time.Duration(retryAfter) + 1) * time.Second)
			} else {
				fmt.Println(err)
				return fmt.Errorf("failed to fetch public stashes: %s", err.Description)
			}
		}
		stashChannel <- StashChange{ChangeID: changeId, Stashes: response.Stashes}
		changeId = response.NextChangeID
	}
	return nil
}

func ProcessStashChanges(event *repository.Event, itemChecker *parser.ItemChecker, objectiveMatchService *service.ObjectiveMatchService, stashChannel chan StashChange) {
	userMap := make(map[string]int)
	for _, team := range event.Teams {
		for _, user := range team.Users {
			userMap[user.AccountName] = user.ID
		}
	}

	for stashChange := range stashChannel {
		intStashChange, err := stashChangeToInt(stashChange.ChangeID)
		if err != nil {
			fmt.Println(err)
			continue
		}
		for _, stash := range stashChange.Stashes {
			objectiveMatchService.SaveStashChange(stash.ID, intStashChange)
			if stash.League != nil && *stash.League == event.Name && stash.AccountName != nil && userMap[*stash.AccountName] != 0 {
				userId := userMap[*stash.AccountName]
				completions := make(map[int]int)
				for _, item := range stash.Items {
					for _, result := range itemChecker.CheckForCompletions(&item) {
						completions[result.ObjectiveId] += result.Number
					}
				}
				objectiveMatchService.SaveItemMatches(completions, userId, intStashChange, stash.ID)
				// for objectiveId, number := range completions {
				// 	fmt.Printf("User: %d, Objective: %d, Number: %d\n", userId, objectiveId, number)
				// }
			}
		}
	}
}

func stashChangeToInt(change string) (int64, error) {
	sum := int64(0)
	for _, part := range strings.Split(change, "-") {
		value, err := strconv.ParseInt(part, 10, 64)
		if err != nil {
			return 0, err
		}
		sum += value
	}
	return sum, nil
}

func StashLoop(db *gorm.DB, poeClient *client.PoEClient, endTime time.Time) {
	var stashChannel = make(chan StashChange, 10000)

	event, err := service.NewEventService(db).GetCurrentEvent("Teams", "Teams.Users")
	if err != nil {
		fmt.Println(err)
		return
	}

	objectives, err := service.NewObjectiveService(db).GetObjectivesByEventId(event.ID)
	if err != nil {
		fmt.Println(err)
		return
	}
	itemChecker, err := parser.NewItemChecker(objectives)
	if err != nil {
		fmt.Println(err)
		return
	}
	objectiveMatchService := service.NewObjectiveMatchService(db)
	go FetchStashChanges(poeClient, endTime, stashChannel)
	go ProcessStashChanges(event, itemChecker, objectiveMatchService, stashChannel)
}
