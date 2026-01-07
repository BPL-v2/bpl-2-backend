package repository

import (
	"bpl/config"
	"database/sql/driver"

	"github.com/lib/pq"
	"gorm.io/gorm"
)

type ExtendingNumberSlice []float64

func (e ExtendingNumberSlice) Get(i int) float64 {
	if len(e) == 0 {
		return 0
	}
	if i >= len(e) {
		return e[len(e)-1]
	}
	return e[i]
}

func (e *ExtendingNumberSlice) Scan(value interface{}) error {
	var floatArray pq.Float64Array
	if err := floatArray.Scan(value); err != nil {
		return err
	}
	*e = ExtendingNumberSlice(floatArray)
	return nil
}

func (e ExtendingNumberSlice) Value() (driver.Value, error) {
	floatArray := pq.Float64Array(e)
	return floatArray.Value()
}

type ScoringMethod string

const (
	// Objective Scoring Methods
	PRESENCE          ScoringMethod = "PRESENCE"
	POINTS_FROM_VALUE ScoringMethod = "POINTS_FROM_VALUE"
	RANKED_TIME       ScoringMethod = "RANKED_TIME"
	RANKED_VALUE      ScoringMethod = "RANKED_VALUE"
	RANKED_REVERSE    ScoringMethod = "RANKED_REVERSE"
	// Category Scoring Methods
	RANKED_COMPLETION    ScoringMethod = "RANKED_COMPLETION_TIME"
	BONUS_PER_COMPLETION ScoringMethod = "BONUS_PER_COMPLETION"
	BINGO_3              ScoringMethod = "BINGO_3"
	BINGO_BOARD          ScoringMethod = "BINGO_BOARD"
)

type ScoringPreset struct {
	Id            int                  `gorm:"primaryKey"`
	EventId       int                  `gorm:"not null;references events(id)"`
	Name          string               `gorm:"not null"`
	Description   string               `gorm:"not null"`
	Points        ExtendingNumberSlice `gorm:"type:numeric[];not null"`
	PointCap      int                  `gorm:"not null"`
	ScoringMethod ScoringMethod        `gorm:"not null"`
	Extra         *string              `gorm:"null"`
}

type ScoringPresetRepository struct {
	DB *gorm.DB
}

func NewScoringPresetRepository() *ScoringPresetRepository {
	return &ScoringPresetRepository{DB: config.DatabaseConnection()}
}

func (r *ScoringPresetRepository) SavePreset(preset *ScoringPreset) (*ScoringPreset, error) {
	result := r.DB.Save(preset)
	if result.Error != nil {
		return nil, result.Error
	}
	return preset, nil
}

func (r *ScoringPresetRepository) GetPresetsForEvent(eventId int) ([]*ScoringPreset, error) {
	var presets []*ScoringPreset
	result := r.DB.Find(&presets, ScoringPreset{EventId: eventId})
	if result.Error != nil {
		return nil, result.Error
	}
	return presets, nil
}

func (r *ScoringPresetRepository) DeletePreset(presetId int) error {
	result := r.DB.Delete(&ScoringPreset{}, &ScoringPreset{Id: presetId})
	return result.Error
}

func (r *ScoringPresetRepository) DeletePresetsForEvent(eventId int) error {
	result := r.DB.Delete(&ScoringPreset{}, &ScoringPreset{EventId: eventId})
	return result.Error
}
