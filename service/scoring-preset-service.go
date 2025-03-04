package service

import (
	"bpl/repository"
)

type ScoringPresetService struct {
	scoringPresetRepository *repository.ScoringPresetRepository
	objectiveRepository     *repository.ObjectiveRepository
}

func NewScoringPresetsService() *ScoringPresetService {
	return &ScoringPresetService{
		scoringPresetRepository: repository.NewScoringPresetRepository(),
		objectiveRepository:     repository.NewObjectiveRepository(),
	}
}

func (s *ScoringPresetService) SavePreset(preset *repository.ScoringPreset) (*repository.ScoringPreset, error) {
	return s.scoringPresetRepository.SavePreset(preset)
}

func (s *ScoringPresetService) GetPresetById(presetId int) (*repository.ScoringPreset, error) {
	return s.scoringPresetRepository.GetPresetById(presetId)
}

func (s *ScoringPresetService) GetPresetsForEvent(eventId int) ([]*repository.ScoringPreset, error) {
	return s.scoringPresetRepository.GetPresetsForEvent(eventId)
}

func (s *ScoringPresetService) DeletePreset(presetId int) error {
	err := s.objectiveRepository.RemoveScoringId(presetId)
	if err != nil {
		return err
	}
	return s.scoringPresetRepository.DeletePreset(presetId)
}

func (s *ScoringPresetService) DuplicatePresets(oldEventId int, newEventId int) (map[int]int, error) {
	presets, err := s.GetPresetsForEvent(oldEventId)
	if err != nil {
		return nil, err
	}
	presetMap := make(map[int]int)
	for _, preset := range presets {
		newPreset := &repository.ScoringPreset{
			EventID:       newEventId,
			Name:          preset.Name,
			Description:   preset.Description,
			Points:        preset.Points,
			ScoringMethod: preset.ScoringMethod,
			Type:          preset.Type,
		}
		newPreset, err := s.SavePreset(newPreset)
		if err != nil {
			return nil, err
		}
		presetMap[preset.ID] = newPreset.ID
	}
	return presetMap, nil
}
