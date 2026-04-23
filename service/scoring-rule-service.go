package service

import (
	"bpl/repository"
	"bpl/utils"
)

type ScoringRuleService interface {
	SaveRule(rule *repository.ScoringRule) (*repository.ScoringRule, error)
	SaveRules(rules []*repository.ScoringRule) ([]*repository.ScoringRule, error)
	GetRulesForEvent(eventId int) ([]*repository.ScoringRule, error)
	DeleteRule(ruleId int) error
	DuplicateRules(oldEventId int, newEventId int) (map[int]*repository.ScoringRule, error)
}

type ScoringRuleServiceImpl struct {
	scoringRuleRepository repository.ScoringRuleRepository
	objectiveRepository   repository.ObjectiveRepository
}

func NewScoringRulesService() ScoringRuleService {
	return &ScoringRuleServiceImpl{
		scoringRuleRepository: repository.NewScoringRuleRepository(),
		objectiveRepository:   repository.NewObjectiveRepository(),
	}
}

func (s *ScoringRuleServiceImpl) SaveRule(rule *repository.ScoringRule) (*repository.ScoringRule, error) {
	return s.scoringRuleRepository.SaveRule(rule)
}

func (s *ScoringRuleServiceImpl) SaveRules(rules []*repository.ScoringRule) ([]*repository.ScoringRule, error) {
	return s.scoringRuleRepository.SaveRules(rules)
}

func (s *ScoringRuleServiceImpl) GetRulesForEvent(eventId int) ([]*repository.ScoringRule, error) {
	return s.scoringRuleRepository.GetRulesForEvent(eventId)
}

func (s *ScoringRuleServiceImpl) DeleteRule(ruleId int) error {
	return s.scoringRuleRepository.DeleteRule(ruleId)
}

func (s *ScoringRuleServiceImpl) DuplicateRules(oldEventId int, newEventId int) (map[int]*repository.ScoringRule, error) {
	rules, err := s.GetRulesForEvent(oldEventId)
	if err != nil {
		return nil, err
	}
	ruleMap := make(map[int]*repository.ScoringRule)
	for _, rule := range rules {
		newRule := &repository.ScoringRule{
			EventId:     newEventId,
			Name:        rule.Name,
			Description: rule.Description,
			Points:      rule.Points,
			RuleType:    rule.RuleType,
			PointCap:    rule.PointCap,
			Extra:       rule.Extra,
		}
		ruleMap[rule.Id] = newRule
	}
	_, err = s.SaveRules(utils.Values(ruleMap))
	return ruleMap, err
}
