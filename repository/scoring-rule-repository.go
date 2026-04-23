package repository

import (
	"bpl/config"
	"database/sql/driver"
	"encoding/json"

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

func (e ExtendingNumberSlice) GetScoreFromNumber(number int) float64 {
	if len(e) == 0 {
		return 0
	}
	capped := min(number, len(e))
	score := 0.0
	for i := range capped {
		score += e[i]
	}
	score += float64(number-capped) * e[len(e)-1]
	return score
}

func (e *ExtendingNumberSlice) Scan(value any) error {
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

type ExtraMap map[string]string

func (e *ExtraMap) Scan(value any) error {
	if value == nil {
		*e = make(map[string]string)
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, e)
}

func (e ExtraMap) Value() (driver.Value, error) {
	if e == nil {
		return json.Marshal(map[string]string{})
	}
	return json.Marshal(e)
}

type ScoringRuleType string

const (
	// Objective Scoring Methods
	FIXED_POINTS_ON_COMPLETION ScoringRuleType = "FIXED_POINTS_ON_COMPLETION"
	POINTS_BY_VALUE            ScoringRuleType = "POINTS_BY_VALUE"
	RANK_BY_COMPLETION_TIME    ScoringRuleType = "RANK_BY_COMPLETION_TIME"
	RANK_BY_HIGHEST_VALUE      ScoringRuleType = "RANK_BY_HIGHEST_VALUE"
	RANK_BY_LOWEST_VALUE       ScoringRuleType = "RANK_BY_LOWEST_VALUE"
	// Category Scoring Methods
	RANK_BY_CHILD_COMPLETION_TIME ScoringRuleType = "RANK_BY_CHILD_COMPLETION_TIME"
	BONUS_PER_CHILD_COMPLETION    ScoringRuleType = "BONUS_PER_CHILD_COMPLETION"
	BINGO_BOARD_RANKING           ScoringRuleType = "BINGO_BOARD_RANKING"
	RANK_BY_CHILD_VALUE_SUM       ScoringRuleType = "RANK_BY_CHILD_VALUE_SUM"
)

type ScoringRule struct {
	Id          int                  `gorm:"primaryKey"`
	EventId     int                  `gorm:"not null;references events(id)"`
	Name        string               `gorm:"not null"`
	Description string               `gorm:"not null"`
	Points      ExtendingNumberSlice `gorm:"type:numeric[];not null"`
	PointCap    int                  `gorm:"not null"`
	RuleType    ScoringRuleType      `gorm:"column:scoring_rule;not null"`
	Extra       ExtraMap             `gorm:"type:jsonb;not null;default:'{}'"`
}

type ScoringRuleRepository interface {
	SaveRule(rule *ScoringRule) (*ScoringRule, error)
	SaveRules(rules []*ScoringRule) ([]*ScoringRule, error)
	GetRulesForEvent(eventId int) ([]*ScoringRule, error)
	DeleteRule(ruleId int) error
	DeleteRulesForEvent(eventId int) error
}

type ScoringRuleRepositoryImpl struct {
	DB *gorm.DB
}

func NewScoringRuleRepository() ScoringRuleRepository {
	return &ScoringRuleRepositoryImpl{DB: config.DatabaseConnection()}
}

func (r *ScoringRuleRepositoryImpl) SaveRule(rule *ScoringRule) (*ScoringRule, error) {
	result := r.DB.Save(rule)
	return rule, result.Error
}
func (r *ScoringRuleRepositoryImpl) SaveRules(rules []*ScoringRule) ([]*ScoringRule, error) {
	result := r.DB.Save(rules)
	return rules, result.Error
}

func (r *ScoringRuleRepositoryImpl) GetRulesForEvent(eventId int) ([]*ScoringRule, error) {
	var rules []*ScoringRule
	result := r.DB.Find(&rules, ScoringRule{EventId: eventId})
	if result.Error != nil {
		return nil, result.Error
	}
	return rules, nil
}

func (r *ScoringRuleRepositoryImpl) DeleteRule(ruleId int) error {
	result := r.DB.Delete(&ScoringRule{}, &ScoringRule{Id: ruleId})
	return result.Error
}

func (r *ScoringRuleRepositoryImpl) DeleteRulesForEvent(eventId int) error {
	result := r.DB.Delete(&ScoringRule{}, &ScoringRule{EventId: eventId})
	return result.Error
}
