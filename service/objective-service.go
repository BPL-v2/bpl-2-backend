package service

import (
	"bpl/parser"
	"bpl/repository"
)

type ObjectiveService struct {
	objectiveRepository *repository.ObjectiveRepository
	conditionRepository *repository.ConditionRepository
}

func NewObjectiveService() *ObjectiveService {
	return &ObjectiveService{
		objectiveRepository: repository.NewObjectiveRepository(),
		conditionRepository: repository.NewConditionRepository(),
	}
}

func (e *ObjectiveService) CreateObjective(objective *repository.Objective) (*repository.Objective, error) {
	var err error
	// saving conditions separately is necessary for some weird reason, otherwise condition updates will not be saved
	if objective.Id != 0 {
		for _, condition := range objective.Conditions {
			condition.ObjectiveId = objective.Id
			res := e.objectiveRepository.DB.Save(condition)
			if res.Error != nil {
				return nil, res.Error
			}
		}
	}
	objective, err = e.objectiveRepository.SaveObjective(objective)
	if err != nil {
		return nil, err
	}
	return objective, nil
}

func (e *ObjectiveService) DeleteObjective(objectiveId int) error {
	return e.objectiveRepository.DeleteObjective(objectiveId)
}

func (e *ObjectiveService) GetObjectiveById(objectiveId int) (*repository.Objective, error) {
	return e.objectiveRepository.GetObjectiveById(objectiveId, "Conditions")
}

func (e *ObjectiveService) GetParser(eventId int) (*parser.ItemChecker, error) {
	objectives, err := e.GetObjectivesForEvent(eventId, "ScoringPreset", "Conditions")
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

func (e *ObjectiveService) DuplicateObjectives(oldEventId, newEventId int, presetIdMap map[int]int) error {
	objectives, err := e.objectiveRepository.GetObjectivesByEventIdFlat(oldEventId, "Conditions")
	if err != nil {
		return err
	}
	newObjectiveMap := make(map[int]*repository.Objective)
	for _, objective := range objectives {
		newObjective := *objective
		oldId := newObjective.Id
		for _, condition := range newObjective.Conditions {
			condition.Id = 0
		}
		newObjective.Id = 0
		newObjective.EventId = newEventId
		if newObjective.ScoringId != nil {
			if newId, ok := presetIdMap[*newObjective.ScoringId]; ok {
				newObjective.ScoringId = &newId
			}
		}
		obj, err := e.objectiveRepository.SaveObjective(&newObjective)
		if err != nil {
			return err
		}
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
