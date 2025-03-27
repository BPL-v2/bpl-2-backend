package service

import "bpl/repository"

type TeamSuggestionService struct {
	teamSuggestionRepository *repository.TeamSuggestionRepository
}

func NewTeamSuggestionService() *TeamSuggestionService {
	return &TeamSuggestionService{
		teamSuggestionRepository: repository.NewTeamSuggestionRepository(),
	}
}

func (t *TeamSuggestionService) GetSuggestionsForTeam(teamId int) ([]*repository.TeamSuggestion, error) {
	return t.teamSuggestionRepository.GetSuggestionsForTeam(teamId)
}

func (t *TeamSuggestionService) SaveSuggestion(id int, teamId int, isObjective bool) error {
	suggestion := &repository.TeamSuggestion{
		Id:          id,
		TeamId:      teamId,
		IsObjective: isObjective,
	}
	return t.teamSuggestionRepository.SaveSuggestion(suggestion)
}

func (t *TeamSuggestionService) DeleteSuggestion(id int, teamId int, isObjective bool) error {
	suggestion := &repository.TeamSuggestion{
		Id:          id,
		TeamId:      teamId,
		IsObjective: isObjective,
	}
	return t.teamSuggestionRepository.DeleteSuggestion(suggestion)
}
