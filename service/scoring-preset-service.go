package service

import (
	"bpl/repository"

	"gorm.io/gorm"
)

type ScoringPresetsService struct {
	scoring_preset_repository *repository.ScoringPresetRepository
}

func NewScoringPresetsService(db *gorm.DB) *ScoringPresetsService {
	return &ScoringPresetsService{
		scoring_preset_repository: repository.NewScoringPresetRepository(db),
	}
}

func (s *ScoringPresetsService) SavePreset(preset *repository.ScoringPreset) (*repository.ScoringPreset, error) {
	return s.scoring_preset_repository.SavePreset(preset)
}

func (s *ScoringPresetsService) GetPresetById(presetId int) (*repository.ScoringPreset, error) {
	return s.scoring_preset_repository.GetPresetById(presetId)
}

func (s *ScoringPresetsService) GetPresetsForEvent(eventId int) ([]*repository.ScoringPreset, error) {
	return s.scoring_preset_repository.GetPresetsForEvent(eventId)
}
