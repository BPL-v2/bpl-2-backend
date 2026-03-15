package service

import (
	"bpl/repository"
	"bpl/utils"
)

type ScoringPresetService interface {
	SavePreset(preset *repository.ScoringPreset) (*repository.ScoringPreset, error)
	SavePresets(presets []*repository.ScoringPreset) ([]*repository.ScoringPreset, error)
	GetPresetsForEvent(eventId int) ([]*repository.ScoringPreset, error)
	DeletePreset(presetId int) error
	DuplicatePresets(oldEventId int, newEventId int) (map[int]*repository.ScoringPreset, error)
}

type ScoringPresetServiceImpl struct {
	scoringPresetRepository repository.ScoringPresetRepository
	objectiveRepository     repository.ObjectiveRepository
}

func NewScoringPresetsService() ScoringPresetService {
	return &ScoringPresetServiceImpl{
		scoringPresetRepository: repository.NewScoringPresetRepository(),
		objectiveRepository:     repository.NewObjectiveRepository(),
	}
}

func (s *ScoringPresetServiceImpl) SavePreset(preset *repository.ScoringPreset) (*repository.ScoringPreset, error) {
	return s.scoringPresetRepository.SavePreset(preset)
}

func (s *ScoringPresetServiceImpl) SavePresets(presets []*repository.ScoringPreset) ([]*repository.ScoringPreset, error) {
	return s.scoringPresetRepository.SavePresets(presets)
}

func (s *ScoringPresetServiceImpl) GetPresetsForEvent(eventId int) ([]*repository.ScoringPreset, error) {
	return s.scoringPresetRepository.GetPresetsForEvent(eventId)
}

func (s *ScoringPresetServiceImpl) DeletePreset(presetId int) error {
	return s.scoringPresetRepository.DeletePreset(presetId)
}

func (s *ScoringPresetServiceImpl) DuplicatePresets(oldEventId int, newEventId int) (map[int]*repository.ScoringPreset, error) {
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
