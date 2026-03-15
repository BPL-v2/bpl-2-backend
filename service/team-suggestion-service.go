package service

import "bpl/repository"

type TeamSuggestionService interface {
	GetSuggestionsForTeam(teamId int) ([]*repository.TeamSuggestion, error)
	SaveSuggestion(id int, teamId int, extra string) error
	DeleteSuggestion(id int, teamId int) error
}

type TeamSuggestionServiceImpl struct {
	teamSuggestionRepository repository.TeamSuggestionRepository
}

func NewTeamSuggestionService() TeamSuggestionService {
	return &TeamSuggestionServiceImpl{
		teamSuggestionRepository: repository.NewTeamSuggestionRepository(),
	}
}

func (t *TeamSuggestionServiceImpl) GetSuggestionsForTeam(teamId int) ([]*repository.TeamSuggestion, error) {
	return t.teamSuggestionRepository.GetSuggestionsForTeam(teamId)
}

func (t *TeamSuggestionServiceImpl) SaveSuggestion(id int, teamId int, extra string) error {
	suggestion := &repository.TeamSuggestion{
		Id:     id,
		TeamId: teamId,
		Extra:  extra,
	}
	return t.teamSuggestionRepository.SaveSuggestion(suggestion)
}

func (t *TeamSuggestionServiceImpl) DeleteSuggestion(id int, teamId int) error {
	suggestion := &repository.TeamSuggestion{
		Id:     id,
		TeamId: teamId,
	}
	return t.teamSuggestionRepository.DeleteSuggestion(suggestion)
}
