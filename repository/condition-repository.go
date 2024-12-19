package repository

import (
	"gorm.io/gorm"
)

type Operator string
type ItemField string

const (
	BASE_TYPE      ItemField = "BASE_TYPE"
	NAME           ItemField = "NAME"
	TYPE_LINE      ItemField = "TYPE_LINE"
	RARITY         ItemField = "RARITY"
	ILVL           ItemField = "ILVL"
	FRAME_TYPE     ItemField = "FRAME_TYPE"
	TALISMAN_TIER  ItemField = "TALISMAN_TIER"
	ENCHANT_MODS   ItemField = "ENCHANT_MODS"
	EXPLICIT_MODS  ItemField = "EXPLICIT_MODS"
	IMPLICIT_MODS  ItemField = "IMPLICIT_MODS"
	CRAFTED_MODS   ItemField = "CRAFTED_MODS"
	FRACTURED_MODS ItemField = "FRACTURED_MODS"
	SIX_LINK       ItemField = "SIX_LINK"
)

const (
	EQ                   Operator = "EQ"
	NEQ                  Operator = "NEQ"
	GT                   Operator = "GT"
	GTE                  Operator = "GTE"
	LT                   Operator = "LT"
	LTE                  Operator = "LTE"
	IN                   Operator = "IN"
	NOT_IN               Operator = "NOT_IN"
	MATCHES              Operator = "MATCHES"
	CONTAINS             Operator = "CONTAINS"
	CONTAINS_ALL         Operator = "CONTAINS_ALL"
	CONTAINS_MATCH       Operator = "CONTAINS_MATCH"
	CONTAINS_ALL_MATCHES Operator = "CONTAINS_ALL_MATCHES"
)

type Condition struct {
	ID          int       `gorm:"primaryKey;autoIncrement"`
	ObjectiveID int       `gorm:"not null"`
	Field       ItemField `gorm:"type:bpl2.item_field"`
	Operator    Operator  `gorm:"type:bpl2.operator"`
	Value       string    `gorm:"not null"`
}

type ConditionRepository struct {
	DB *gorm.DB
}

func NewConditionRepository(db *gorm.DB) *ConditionRepository {
	return &ConditionRepository{DB: db}
}

func (r *ConditionRepository) GetConditionByPK(objectiveID int, field ItemField, operator Operator) (*Condition, error) {
	var condition Condition
	result := r.DB.First(&condition, "objective_id = ? AND field = ? AND operator = ?", objectiveID, field, operator)
	if result.Error != nil {
		return nil, result.Error
	}
	return &condition, nil
}

func (r *ConditionRepository) SaveCondition(condition *Condition) (*Condition, error) {
	result := r.DB.Save(condition)
	if result.Error != nil {
		return nil, result.Error
	}
	return condition, nil
}

func (r *ConditionRepository) DeleteCondition(conditionId int) error {
	result := r.DB.Delete(&Condition{}, "id = ?", conditionId)
	return result.Error
}
