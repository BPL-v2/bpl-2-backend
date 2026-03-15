package repository

import (
	"bpl/config"

	"gorm.io/gorm"
)

type TeamSuggestion struct {
	Id     int    `gorm:"not null;primaryKey"`
	TeamId int    `gorm:"not null;primaryKey"`
	Extra  string `gorm:"not null"`
}

type TeamSuggestionRepository interface {
	SaveSuggestion(suggestion *TeamSuggestion) error
	GetSuggestionsForTeam(teamId int) ([]*TeamSuggestion, error)
	DeleteSuggestion(suggestion *TeamSuggestion) error
}

type TeamSuggestionRepositoryImpl struct {
	DB *gorm.DB
}

func NewTeamSuggestionRepository() TeamSuggestionRepository {
	return &TeamSuggestionRepositoryImpl{DB: config.DatabaseConnection()}
}

func (r *TeamSuggestionRepositoryImpl) SaveSuggestion(suggestion *TeamSuggestion) error {
	return r.DB.Save(&suggestion).Error
}

func (r *TeamSuggestionRepositoryImpl) GetSuggestionsForTeam(teamId int) ([]*TeamSuggestion, error) {
	var suggestions []*TeamSuggestion
	result := r.DB.Find(&suggestions, TeamSuggestion{TeamId: teamId})
	if result.Error != nil {
		return nil, result.Error
	}
	return suggestions, nil
}

func (r *TeamSuggestionRepositoryImpl) DeleteSuggestion(suggestion *TeamSuggestion) error {
	return r.DB.Delete(suggestion).Error
}
