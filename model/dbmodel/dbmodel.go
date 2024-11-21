package dbmodel

type ObjectiveType int
type Operator int

type ItemField int

const (
	BASE_TYPE ItemField = iota
	NAME
	TYPE_LINE
	RARITY
	ILVL
	FRAME_TYPE
	TALISMAN_TIER
	ENCHANT_MODS
	EXPLICIT_MODS
	IMPLICIT_MODS
	CRAFTED_MODS
	FRACTURED_MODS
	SIX_LINK
)

func (f *ItemField) ToString() string {
	switch *f {
	case BASE_TYPE:
		return "BASE_TYPE"
	case NAME:
		return "NAME"
	case TYPE_LINE:
		return "TYPE_LINE"
	case RARITY:
		return "RARITY"
	case ILVL:
		return "ILVL"
	case FRAME_TYPE:
		return "FRAME_TYPE"
	case TALISMAN_TIER:
		return "TALISMAN_TIER"
	case ENCHANT_MODS:
		return "ENCHANT_MODS"
	case EXPLICIT_MODS:
		return "EXPLICIT_MODS"
	case IMPLICIT_MODS:
		return "IMPLICIT_MODS"
	case CRAFTED_MODS:
		return "CRAFTED_MODS"
	case FRACTURED_MODS:
		return "FRACTURED_MODS"
	case SIX_LINK:
		return "SIX_LINK"
	default:
		return "UNKNOWN"
	}
}

const (
	EQ Operator = iota
	NEQ
	GT
	GTE
	LT
	LTE
	IN
	NOT_IN
	MATCHES
	CONTAINS
	CONTAINS_ALL
	CONTAINS_MATCH
	CONTAINS_ALL_MATCHES
)

const (
	ITEM ObjectiveType = iota
)

type Objective struct {
	ID             int           `gorm:"primaryKey"`
	Name           string        `gorm:"not null"`
	RequiredNumber string        `gorm:"not null"`
	Conditions     []Condition   `gorm:"foreignKey:ObjectiveID"`
	CategoryID     int           `gorm:"not null"`
	ObjectiveType  ObjectiveType `gorm:"not null"`
	ValidFrom      *int64        `gorm:"null"`
	ValidTo        *int64        `gorm:"null"`
}

type Condition struct {
	ObjectiveID int       `gorm:"primaryKey;autoIncrement:false"`
	Field       ItemField `gorm:"primaryKey"`
	Operator    Operator  `gorm:"primaryKey"`
	Value       string    `gorm:"not null"`
}

type ScoringMethodType string

const (
	PRESENCE          ScoringMethodType = "PRESENCE"
	RANKED            ScoringMethodType = "RANKED"
	RELATIVE_PRESENCE ScoringMethodType = "RELATIVE_PRESENCE"
)

type ScoringMethod struct {
	CategoryID int               `gorm:"primaryKey"`
	Type       ScoringMethodType `gorm:"primaryKey"`
	Points     []int             `gorm:"type:integer[]"`
}

type ScoringMethodInheritance string

const (
	OVERWRITE ScoringMethodInheritance = "OVERWRITE"
	INHERIT   ScoringMethodInheritance = "INHERIT"
	EXTEND    ScoringMethodInheritance = "EXTEND"
)

type ScoringCategory struct {
	ID             int                      `gorm:"primaryKey foreignKey:CategoryID references:ID on:objectives"`
	Name           string                   `gorm:"not null"`
	Inheritance    ScoringMethodInheritance `gorm:"not null"`
	ParentID       *int                     `gorm:"null"`
	ScoringMethods []ScoringMethod          `gorm:"foreignKey:CategoryID"`
	Event          Event                    `gorm:"foreignKey:ScoringCategoryID"`
}

type Event struct {
	ID                int    `gorm:"primaryKey"`
	Name              string `gorm:"not null"`
	ScoringCategoryID int    `gorm:"not null"`
}
