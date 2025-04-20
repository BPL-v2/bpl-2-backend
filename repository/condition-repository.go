package repository

import (
	"bpl/config"

	"gorm.io/gorm"
)

type Operator string
type ItemField string

const (
	BASE_TYPE       ItemField = "BASE_TYPE"
	NAME            ItemField = "NAME"
	TYPE_LINE       ItemField = "TYPE_LINE"
	QUALITY         ItemField = "QUALITY"
	LEVEL           ItemField = "LEVEL"
	RARITY          ItemField = "RARITY"
	ILVL            ItemField = "ILVL"
	FRAME_TYPE      ItemField = "FRAME_TYPE"
	TALISMAN_TIER   ItemField = "TALISMAN_TIER"
	ENCHANTS        ItemField = "ENCHANT_MODS"
	EXPLICITS       ItemField = "EXPLICIT_MODS"
	IMPLICITS       ItemField = "IMPLICIT_MODS"
	CRAFTED_MODS    ItemField = "CRAFTED_MODS"
	FRACTURED_MODS  ItemField = "FRACTURED_MODS"
	MAX_LINKS       ItemField = "MAX_LINKS"
	SOCKETS         ItemField = "SOCKETS" // as string like "RGGB"
	INCUBATOR_KILLS ItemField = "INCUBATOR_KILLS"
	IS_CORRUPTED    ItemField = "IS_CORRUPTED"
	IS_VAAL         ItemField = "IS_VAAL"
	SANCTUM_MODS    ItemField = "SANCTUM_AFFLICTIONS"
	TEMPLE_ROOMS    ItemField = "TEMPLE_ROOMS"
	RITUAL_BOSSES   ItemField = "RITUAL_VESSEL_BOSSES"
	RITUAL_MAP      ItemField = "RITUAL_VESSEL_MAP"
)

type FieldType string

const (
	String      FieldType = "string"
	Int         FieldType = "int"
	Bool        FieldType = "bool"
	StringArray FieldType = "string[]"
)

var FieldToType = map[ItemField]FieldType{
	BASE_TYPE:       String,
	NAME:            String,
	TYPE_LINE:       String,
	QUALITY:         Int,
	LEVEL:           Int,
	RARITY:          String,
	ILVL:            Int,
	FRAME_TYPE:      String,
	TALISMAN_TIER:   Int,
	ENCHANTS:        StringArray,
	EXPLICITS:       StringArray,
	IMPLICITS:       StringArray,
	CRAFTED_MODS:    StringArray,
	FRACTURED_MODS:  StringArray,
	MAX_LINKS:       Int,
	SOCKETS:         String,
	INCUBATOR_KILLS: Int,
	IS_CORRUPTED:    Bool,
	IS_VAAL:         Bool,
	SANCTUM_MODS:    StringArray,
	TEMPLE_ROOMS:    StringArray,
	RITUAL_BOSSES:   StringArray,
	RITUAL_MAP:      StringArray,
}

var OperatorsForTypes = map[FieldType][]Operator{
	String:      {EQ, NEQ, IN, NOT_IN, MATCHES, CONTAINS, LENGTH_EQ, LENGTH_GT, LENGTH_LT},
	Int:         {EQ, NEQ, GT, LT, IN, NOT_IN},
	Bool:        {EQ, NEQ},
	StringArray: {CONTAINS, CONTAINS_MATCH, LENGTH_EQ, LENGTH_GT, LENGTH_LT},
}

const (
	EQ             Operator = "EQ"
	NEQ            Operator = "NEQ"
	GT             Operator = "GT"
	LT             Operator = "LT"
	IN             Operator = "IN"
	NOT_IN         Operator = "NOT_IN"
	MATCHES        Operator = "MATCHES"
	CONTAINS       Operator = "CONTAINS"
	CONTAINS_MATCH Operator = "CONTAINS_MATCH"
	LENGTH_EQ      Operator = "LENGTH_EQ"
	LENGTH_GT      Operator = "LENGTH_GT"
	LENGTH_LT      Operator = "LENGTH_LT"
)

type Condition struct {
	Id          int       `gorm:"primaryKey;autoIncrement"`
	ObjectiveId int       `gorm:"not null"`
	Field       ItemField `gorm:"not null"`
	Operator    Operator  `gorm:"not null"`
	Value       string    `gorm:"not null"`
}

type ConditionRepository struct {
	DB *gorm.DB
}

func NewConditionRepository() *ConditionRepository {
	return &ConditionRepository{DB: config.DatabaseConnection()}
}

func (r *ConditionRepository) SaveCondition(condition *Condition) (*Condition, error) {
	r.DB.Model(&Objective{}).Where(Objective{Id: condition.ObjectiveId}).Update("sync_status", SyncStatusDesynced)
	result := r.DB.Save(condition)
	if result.Error != nil {
		return nil, result.Error
	}
	return condition, nil
}

func (r *ConditionRepository) DeleteCondition(conditionId int) error {
	condition := Condition{}
	r.DB.Where(Condition{Id: conditionId}).First(&condition)
	r.DB.Model(&Objective{}).Where(Objective{Id: condition.ObjectiveId}).Update("sync_status", SyncStatusDesynced)
	result := r.DB.Delete(condition)
	return result.Error
}
