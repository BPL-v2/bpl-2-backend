package service

import (
	"bpl/parser"
	"bpl/repository"
	"bpl/utils"
)

type ObjectiveService struct {
	objectiveRepository     *repository.ObjectiveRepository
	scoringPresetRepository *repository.ScoringPresetRepository
}

func NewObjectiveService() *ObjectiveService {
	return &ObjectiveService{
		objectiveRepository:     repository.NewObjectiveRepository(),
		scoringPresetRepository: repository.NewScoringPresetRepository(),
	}
}

func (e *ObjectiveService) CreateObjective(objective *repository.Objective, presetIds []int) (*repository.Objective, error) {
	var err error
	objective, err = e.objectiveRepository.SaveObjective(objective)
	if err != nil {
		return nil, err
	}
	err = e.objectiveRepository.AssociateScoringPresets(objective.Id, presetIds)
	if err != nil {
		return nil, err
	}
	return objective, nil
}

func (e *ObjectiveService) DeleteObjective(objectiveId int) error {
	return e.objectiveRepository.DeleteObjective(objectiveId)
}

func (e *ObjectiveService) GetObjectiveById(objectiveId int, preloads ...string) (*repository.Objective, error) {
	return e.objectiveRepository.GetObjectiveById(objectiveId, preloads...)
}

func (e *ObjectiveService) GetParser(eventId int) (*parser.ItemChecker, error) {
	objectives, err := e.GetObjectivesForEvent(eventId, "ScoringPresets")
	if err != nil {
		return nil, err
	}
	return parser.NewItemChecker(objectives, false)
}

func (e *ObjectiveService) StartSync(objectiveIds []int) error {
	return e.objectiveRepository.StartSync(objectiveIds)
}

func (e *ObjectiveService) SetSynced(objectiveIds []int) error {
	return e.objectiveRepository.FinishSync(objectiveIds)
}

func (e *ObjectiveService) GetObjectiveTreeForEvent(eventId int, preloads ...string) (*repository.Objective, error) {
	return e.objectiveRepository.GetObjectivesByEventId(eventId, preloads...)
}

func (e *ObjectiveService) GetObjectivesForEvent(eventId int, preloads ...string) ([]*repository.Objective, error) {
	return e.objectiveRepository.GetObjectivesByEventIdFlat(eventId, preloads...)
}

func (e *ObjectiveService) GetAllObjectives(preloads ...string) ([]*repository.Objective, error) {
	return e.objectiveRepository.GetAllObjectives(preloads...)
}

func (e *ObjectiveService) DuplicateObjectives(oldEventId int, newEventId int, presetMap map[int]*repository.ScoringPreset) error {
	objectives, err := e.objectiveRepository.GetObjectivesByEventIdFlat(oldEventId, "ScoringPresets")
	if err != nil {
		return err
	}
	newObjectiveMap := make(map[int]*repository.Objective)
	for _, objective := range objectives {
		newObjective := *objective
		oldId := newObjective.Id
		newObjective.Id = 0
		newObjective.EventId = newEventId

		newPresets := utils.Filter(utils.Map(objective.ScoringPresets, func(preset *repository.ScoringPreset) *repository.ScoringPreset {
			if newPreset, ok := presetMap[preset.Id]; ok {
				return newPreset
			}
			return nil
		}), func(preset *repository.ScoringPreset) bool { return preset != nil })
		newObjective.ScoringPresets = newPresets

		newObjectiveMap[oldId] = &newObjective
	}
	_, err = e.objectiveRepository.SaveObjectives(utils.Values(newObjectiveMap))
	if err != nil {
		return err
	}
	for _, objective := range objectives {
		newObjective := newObjectiveMap[objective.Id]
		if newObjective == nil || objective.ParentId == nil || newObjectiveMap[*objective.ParentId] == nil {
			continue
		}
		newObjective.ParentId = &newObjectiveMap[*objective.ParentId].Id
	}
	_, err = e.objectiveRepository.SaveObjectives(utils.Values(newObjectiveMap))
	return err

}
