package repository

import (
	"bpl/utils"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

type ScoringPresetType string
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
	if value == nil {
		*e = ExtendingNumberSlice{}
		return nil
	}

	switch v := value.(type) {
	case string:
		// Handle PostgreSQL array format
		v = strings.Trim(v, "{}")
		if v == "" {
			*e = ExtendingNumberSlice{}
			return nil
		}
		var numbers []float64
		for _, s := range strings.Split(v, ",") {
			num, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
			if err != nil {
				return err
			}
			numbers = append(numbers, num)
		}
		*e = ExtendingNumberSlice(numbers)
		return nil
	case []byte:
		// Handle PostgreSQL array format
		str := strings.Trim(string(v), "{}")
		if str == "" {
			*e = ExtendingNumberSlice{}
			return nil
		}
		var numbers []float64
		for _, s := range strings.Split(str, ",") {
			num, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
			if err != nil {
				return err
			}
			numbers = append(numbers, num)
		}
		*e = ExtendingNumberSlice(numbers)
		return nil
	default:
		return errors.New("unsupported data type for ExtendingNumberSlice")
	}
}

func (e ExtendingNumberSlice) Value() (driver.Value, error) {
	return json.Marshal(e)
}

const (
	OBJECTIVE ScoringPresetType = "OBJECTIVE"
	CATEGORY  ScoringPresetType = "CATEGORY"
)

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
)

type ScoringPreset struct {
	ID            int                  `gorm:"primaryKey"`
	EventID       int                  `gorm:"not null;references events(id)"`
	Name          string               `gorm:"not null"`
	Description   string               `gorm:"not null"`
	Points        ExtendingNumberSlice `gorm:"type:numeric[];not null"`
	ScoringMethod ScoringMethod        `gorm:"not null;type:bpl2.scoring_method"`
	Type          ScoringPresetType    `gorm:"not null;type:bpl2.scoring_preset_type"`
}

func (s *ScoringPresetType) GetValidMethods() []ScoringMethod {
	switch *s {
	case OBJECTIVE:
		return []ScoringMethod{PRESENCE, POINTS_FROM_VALUE, RANKED_TIME, RANKED_VALUE, RANKED_REVERSE}
	case CATEGORY:
		return []ScoringMethod{RANKED_COMPLETION, BONUS_PER_COMPLETION}
	default:
		return []ScoringMethod{}
	}
}

func (s *ScoringPreset) Validate() error {
	if !utils.Contains(s.Type.GetValidMethods(), s.ScoringMethod) {
		return errors.New("invalid scoring method for scoring preset type")
	}
	return nil
}

type ScoringPresetRepository struct {
	DB *gorm.DB
}

func NewScoringPresetRepository(db *gorm.DB) *ScoringPresetRepository {
	return &ScoringPresetRepository{DB: db}
}

func (r *ScoringPresetRepository) SavePreset(preset *ScoringPreset) (*ScoringPreset, error) {
	err := preset.Validate()
	if err != nil {
		return nil, err
	}
	result := r.DB.Save(preset)
	if result.Error != nil {
		return nil, result.Error
	}
	return preset, nil
}

func (r *ScoringPresetRepository) GetPresetById(presetId int) (*ScoringPreset, error) {
	var preset ScoringPreset
	result := r.DB.First(&preset, "id = ?", presetId)
	if result.Error != nil {
		return nil, result.Error
	}
	return &preset, nil
}

func (r *ScoringPresetRepository) GetPresetsForEvent(eventId int) ([]*ScoringPreset, error) {
	var presets []*ScoringPreset
	result := r.DB.Find(&presets, "event_id = ?", eventId)
	if result.Error != nil {
		return nil, result.Error
	}
	return presets, nil
}
