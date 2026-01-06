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

func (e *ObjectiveService) DuplicateObjectives(oldEventId int, newEventId int, presetIdMap map[int]int) error {
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

		presetIds := utils.Filter(utils.Map(newObjective.ScoringPresets, func(preset *repository.ScoringPreset) int {
			if newId, ok := presetIdMap[preset.Id]; ok {
				return newId
			}
			return 0
		}), func(id int) bool { return id != 0 })
		obj, err := e.objectiveRepository.SaveObjective(&newObjective)
		if err != nil {
			return err
		}
		e.objectiveRepository.AssociateScoringPresets(objective.Id, presetIds)
		newObjectiveMap[oldId] = obj
	}
	for _, objective := range objectives {
		if objective.ParentId != nil {
			if parent, ok := newObjectiveMap[*objective.ParentId]; ok {
				if child, ok := newObjectiveMap[objective.Id]; ok {
					child.ParentId = &parent.Id
					_, err := e.objectiveRepository.SaveObjective(child)
					if err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}
