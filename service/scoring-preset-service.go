package service

import (
	"bpl/repository"
	"bpl/utils"
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

func (s *ScoringPresetService) SavePresets(presets []*repository.ScoringPreset) ([]*repository.ScoringPreset, error) {
	return s.scoringPresetRepository.SavePresets(presets)
}

func (s *ScoringPresetService) GetPresetsForEvent(eventId int) ([]*repository.ScoringPreset, error) {
	return s.scoringPresetRepository.GetPresetsForEvent(eventId)
}

func (s *ScoringPresetService) DeletePreset(presetId int) error {
	return s.scoringPresetRepository.DeletePreset(presetId)
}

func (s *ScoringPresetService) DuplicatePresets(oldEventId int, newEventId int) (map[int]*repository.ScoringPreset, error) {
	presets, err := s.GetPresetsForEvent(oldEventId)
	if err != nil {
		return nil, err
	}
	presetMap := make(map[int]*repository.ScoringPreset)
	for _, preset := range presets {
		newPreset := &repository.ScoringPreset{
			EventId:       newEventId,
			Name:          preset.Name,
			Description:   preset.Description,
			Points:        preset.Points,
			ScoringMethod: preset.ScoringMethod,
			PointCap:      preset.PointCap,
			Extra:         preset.Extra,
		}
		presetMap[preset.Id] = newPreset
	}
	_, err = s.SavePresets(utils.Values(presetMap))
	return presetMap, err
}
