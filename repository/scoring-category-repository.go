package repository

type ScoringCategory struct {
	Id            int                `gorm:"primaryKey foreignKey:CategoryId references:Id on:objectives"`
	Name          string             `gorm:"not null"`
	EventId       int                `gorm:"not null;references:events(id)"`
	ParentId      *int               `gorm:"null;references:scoring_category(id)"`
	ScoringId     *int               `gorm:"null;references:scoring_presets(id)"`
	SubCategories []*ScoringCategory `gorm:"foreignKey:ParentId;constraint:OnDelete:CASCADE"`
	Objectives    []*Objective       `gorm:"foreignKey:ParentId;constraint:OnDelete:CASCADE"`
	ScoringPreset *ScoringPreset     `gorm:"foreignKey:ScoringId;references:Id"`
}
