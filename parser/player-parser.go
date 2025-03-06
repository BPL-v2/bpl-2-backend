package parser

import (
	"bpl/client"
	"bpl/repository"
	"fmt"
	"sync"
	"time"
)

type Player struct {
	UserId          int
	TeamId          int
	Token           string
	CharacterName   *string
	Character       *client.Character
	LeagueAccount   *client.LeagueAccount
	Mu              sync.Mutex
	LastUpdateTimes struct {
		CharacterName time.Time
		Character     time.Time
		LeagueAccount time.Time
	}
	RemovePlayer bool
}

func (p *Player) hasChosenPantheon() bool {
	return p.Character != nil && (p.Character.Passives.PantheonMajor != nil || p.Character.Passives.PantheonMinor != nil)
}

func (p *Player) hasAscended() bool {
	return p.Character != nil && false // TODO: check if character is ascended
}

func (p *Player) atlasTreeNodes() int {
	maxNodes := 0
	if p.LeagueAccount == nil {
		return maxNodes
	}
	for _, tree := range p.LeagueAccount.AtlasPassiveTrees {
		if len(tree.Hashes) > maxNodes {
			maxNodes = len(tree.Hashes)
		}
	}
	return maxNodes
}

func (p *Player) ShouldUpdateCharacterName() bool {
	if p.CharacterName == nil {
		return time.Since(p.LastUpdateTimes.CharacterName) > 1*time.Minute
	}
	return time.Since(p.LastUpdateTimes.CharacterName) > 10*time.Minute
}

func (p *Player) ShouldUpdateCharacter() bool {
	if p.CharacterName == nil {
		return false
	}
	if p.Character == nil {
		return true
	}
	if p.Character.Level > 40 && !p.hasChosenPantheon() {
		return time.Since(p.LastUpdateTimes.Character) > 1*time.Minute
	}
	if p.Character.Level > 68 && !p.hasAscended() {
		return time.Since(p.LastUpdateTimes.Character) > 1*time.Minute
	}
	return time.Since(p.LastUpdateTimes.Character) > 10*time.Minute
}

func (p *Player) ShouldUpdateLeagueAccount() bool {
	if p.Character == nil || p.Character.Level < 55 {
		return false
	}

	if p.atlasTreeNodes() < 100 {
		return time.Since(p.LastUpdateTimes.LeagueAccount) > 1*time.Minute
	}

	return time.Since(p.LastUpdateTimes.LeagueAccount) > 10*time.Minute
}

type PlayerObjectiveChecker func(p *Player) int

func GetChecker(objective *repository.Objective) (PlayerObjectiveChecker, error) {
	if objective.ObjectiveType != repository.PLAYER {
		return nil, fmt.Errorf("not a player objective")
	}
	switch objective.NumberField {
	case repository.PLAYER_LEVEL:
		return func(p *Player) int {
			if p.Character == nil {
				return 0
			}
			return p.Character.Level
		}, nil
	case repository.DELVE_DEPTH:
		return nil, fmt.Errorf("not implemented")
	case repository.PANTHEON:
		return func(p *Player) int {
			if p.hasChosenPantheon() {
				return 1
			}
			return 0
		}, nil
	case repository.ASCENDANCY:
		return func(p *Player) int {
			if p.hasAscended() {
				return 1
			}
			return 0
		}, nil
	case repository.PLAYER_SCORE:
		return func(p *Player) int {
			score := 0
			if p.Character != nil {
				if p.Character.Level > 75 {
					score += 3
				}
				if p.Character.Level > 90 {
					score += 3
				}
			}
			if p.atlasTreeNodes() > 100 {
				score += 3
			}
			if score > 9 {
				return 9
			}
			return score
		}, nil

	default:
		return nil, fmt.Errorf("unsupported number field")
	}
}

type PlayerChecker map[int]PlayerObjectiveChecker

func NewPlayerChecker(objectives []*repository.Objective) (*PlayerChecker, error) {
	checkers := make(map[int]PlayerObjectiveChecker)
	for _, objective := range objectives {
		if objective.ObjectiveType != repository.PLAYER {
			continue
		}
		checker, err := GetChecker(objective)
		if err != nil {
			return nil, err
		}
		checkers[objective.Id] = checker
	}
	return (*PlayerChecker)(&checkers), nil
}

func (pc *PlayerChecker) CheckForCompletions(player *Player) []*CheckResult {
	results := make([]*CheckResult, 0)
	for id, checker := range *pc {
		number := checker(player)
		if number > 0 {
			results = append(results, &CheckResult{
				ObjectiveId: id,
				Number:      number,
			})
		}
	}
	return results
}
