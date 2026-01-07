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

func (s *ScoringPresetService) GetPresetsForEvent(eventId int) ([]*repository.ScoringPreset, error) {
	return s.scoringPresetRepository.GetPresetsForEvent(eventId)
}

func (s *ScoringPresetService) DeletePreset(presetId int) error {
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
			EventId:       newEventId,
			Name:          preset.Name,
			Description:   preset.Description,
			Points:        preset.Points,
			ScoringMethod: preset.ScoringMethod,
			PointCap:      preset.PointCap,
		}
		newPreset, err := s.SavePreset(newPreset)
		if err != nil {
			return nil, err
		}
		presetMap[preset.Id] = newPreset.Id
	}
	return presetMap, nil
}
