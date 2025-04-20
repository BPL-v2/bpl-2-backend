package repository

import (
	"bpl/config"

	"gorm.io/gorm"
)

type TeamSuggestion struct {
	Id          int  `gorm:"not null;primaryKey"`
	TeamId      int  `gorm:"not null;primaryKey"`
	IsObjective bool `gorm:"not null;primaryKey"`
}

type TeamSuggestionRepository struct {
	DB *gorm.DB
}

func NewTeamSuggestionRepository() *TeamSuggestionRepository {
	return &TeamSuggestionRepository{DB: config.DatabaseConnection()}
}

func (r *TeamSuggestionRepository) SaveSuggestion(suggestion *TeamSuggestion) error {
	return r.DB.Save(&suggestion).Error
}

func (r *TeamSuggestionRepository) GetSuggestionsForTeam(teamId int) ([]*TeamSuggestion, error) {
	var suggestions []*TeamSuggestion
	result := r.DB.Find(&suggestions, TeamSuggestion{TeamId: teamId})
	if result.Error != nil {
		return nil, result.Error
	}
	return suggestions, nil
}

func (r *TeamSuggestionRepository) DeleteSuggestion(suggestion *TeamSuggestion) error {
	return r.DB.Delete(suggestion).Error
}
