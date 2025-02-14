package service

import (
	"bpl/repository"
	"bpl/utils"
	"time"

	"gorm.io/gorm"
)

type ObjectiveMatchService struct {
	objective_match_repository *repository.ObjectiveMatchRepository
}

func NewObjectiveMatchService(db *gorm.DB) *ObjectiveMatchService {
	return &ObjectiveMatchService{
		objective_match_repository: repository.NewObjectiveMatchRepository(db),
	}
}

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

func (e *ObjectiveMatchService) CreateMatches(matches map[int]int, userId int, changeId int64, stashId string, eventId int, timestamp time.Time) []*repository.ObjectiveMatch {
	objectiveMatches := make([]*repository.ObjectiveMatch, 0)
	for objectiveId, number := range matches {
		objectiveMatch := &repository.ObjectiveMatch{
			ObjectiveID: objectiveId,
			Timestamp:   timestamp,
			Number:      number,
			UserID:      userId,
			EventId:     eventId,
			StashId:     &stashId,
			ChangeId:    &changeId,
		}
		objectiveMatches = append(objectiveMatches, objectiveMatch)
	}
	return objectiveMatches
}

func (e *ObjectiveMatchService) SaveMatches(matches []*repository.ObjectiveMatch, deleteOld bool) error {
	if len(matches) == 0 {
		return nil
	}
	if deleteOld {
		changeIds := make(map[int64]bool)
		objectiveIds := make(map[int]bool)
		for _, match := range matches {
			changeIds[*match.ChangeId] = true
			objectiveIds[match.ObjectiveID] = true
		}
		err := e.objective_match_repository.DeleteMatches(utils.Keys(changeIds), utils.Keys(objectiveIds))
		if err != nil {
			return err
		}
	}
	return e.objective_match_repository.SaveMatches(matches)
}

func (e *ObjectiveMatchService) GetKafkaConsumer(eventId int) (*repository.KafkaConsumer, error) {
	return e.objective_match_repository.GetKafkaConsumer(eventId)
}

func (e *ObjectiveMatchService) SaveKafkaConsumerId(consumer *repository.KafkaConsumer) error {
	return e.objective_match_repository.SaveKafkaConsumer(consumer)
}
