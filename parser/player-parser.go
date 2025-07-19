package parser

import (
	"bpl/client"
	"bpl/repository"
	"bpl/utils"
	"fmt"
	"sync"
	"time"
)

type Player struct {
	CharacterId       string
	CharacterName     string
	CharacterLevel    int
	CharacterXP       int
	MainSkill         string
	Pantheon          bool
	Ascendancy        string
	AscendancyPoints  int
	AtlasPassiveTrees []client.AtlasPassiveTree
	DelveDepth        int
	EquipmentHash     [32]byte
}

type PlayerUpdate struct {
	UserId           int
	AccountName      string
	TeamId           int
	Token            string
	TokenExpiry      time.Time
	Mu               sync.Mutex
	SuccessiveErrors int

	New Player
	Old Player

	LastUpdateTimes struct {
		CharacterName time.Time
		Character     time.Time
		LeagueAccount time.Time
		PoB           time.Time
	}
}

func (p *Player) MaxAtlasTreeNodes() int {
	return utils.Max(utils.Map(p.AtlasPassiveTrees, func(tree client.AtlasPassiveTree) int {
		return len(tree.Hashes)
	})...)
}

func (p *PlayerUpdate) CanMakeRequests() bool {
	return p.TokenExpiry.After(time.Now()) && p.Token != "" && p.SuccessiveErrors < 5
}

func (p *PlayerUpdate) ShouldUpdateCharacterName() bool {
	if !p.CanMakeRequests() {
		return false
	}
	if p.New.CharacterName == "" {
		return time.Since(p.LastUpdateTimes.CharacterName) > 30*time.Minute
	}
	return time.Since(p.LastUpdateTimes.CharacterName) > 60*time.Minute
}

func (p *PlayerUpdate) ShouldUpdateCharacter() bool {
	if !p.CanMakeRequests() {
		return false
	}
	if p.New.CharacterName == "" {
		return false
	}
	if p.New.CharacterLevel > 40 && !p.New.Pantheon {
		return time.Since(p.LastUpdateTimes.Character) > 2*time.Minute
	}
	if p.New.CharacterLevel > 68 && !(p.New.AscendancyPoints >= 8) {
		return time.Since(p.LastUpdateTimes.Character) > 2*time.Minute
	}
	return time.Since(p.LastUpdateTimes.Character) > 2*time.Minute
}

func (p *PlayerUpdate) ShouldUpdateLeagueAccount() bool {
	if !p.CanMakeRequests() {
		return false
	}
	if p.New.CharacterLevel < 55 {
		return false
	}

	if p.New.MaxAtlasTreeNodes() < 100 {
		return time.Since(p.LastUpdateTimes.LeagueAccount) > 1*time.Minute
	}

	return time.Since(p.LastUpdateTimes.LeagueAccount) > 10*time.Minute
}

type TeamObjectiveChecker func(p []*Player) int

type PlayerObjectiveChecker func(p *Player) int

func GetPlayerChecker(objective *repository.Objective) (PlayerObjectiveChecker, error) {
	if (objective.ObjectiveType != repository.ObjectiveTypePlayer) && (objective.ObjectiveType != repository.ObjectiveTypeTeam) {
		return nil, fmt.Errorf("not a player objective")
	}
	switch objective.NumberField {
	case repository.NumberFieldPlayerLevel:
		return func(p *Player) int {
			return p.CharacterLevel
		}, nil
	case repository.NumberFieldDelveDepth:
		return func(p *Player) int {
			return p.DelveDepth
		}, nil
	case repository.NumberFieldDelveDepthPast100:
		return func(p *Player) int {
			return max(p.DelveDepth-100, 0)
		}, nil
	case repository.NumberFieldPantheon:
		return func(p *Player) int {
			if p.Pantheon {
				return 1
			}
			return 0
		}, nil
	case repository.NumberFieldAscendancy:
		return func(p *Player) int {
			return p.AscendancyPoints
		}, nil
	case repository.NumberFieldPlayerScore:
		return func(p *Player) int {
			score := 0
			if p.CharacterLevel >= 40 {
				score += 1
			}
			if p.CharacterLevel >= 60 {
				score += 1
			}
			if p.CharacterLevel >= 80 {
				score += 1
			}
			if p.CharacterLevel >= 90 {
				score += 3
			}
			if p.AscendancyPoints >= 4 {
				score += 1
			}
			if p.AscendancyPoints >= 6 {
				score += 1
			}
			if p.AscendancyPoints >= 8 {
				score += 1
			}
			if p.MaxAtlasTreeNodes() >= 40 {
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

func GetTeamChecker(objective *repository.Objective) (TeamObjectiveChecker, error) {
	if objective.ObjectiveType != repository.ObjectiveTypeTeam {
		return nil, fmt.Errorf("not a team objective")
	}
	checker, err := GetPlayerChecker(objective)
	if err != nil {
		return nil, err
	}
	return func(p []*Player) int {
		sum := 0
		for _, player := range p {
			sum += checker(player)
		}
		return sum
	}, nil
}

type PlayerChecker map[int]PlayerObjectiveChecker
type TeamChecker map[int]TeamObjectiveChecker

func NewPlayerChecker(objectives []*repository.Objective) (*PlayerChecker, error) {
	checkers := make(map[int]PlayerObjectiveChecker)
	for _, objective := range objectives {
		if objective.ObjectiveType != repository.ObjectiveTypePlayer {
			continue
		}
		checker, err := GetPlayerChecker(objective)
		if err != nil {
			return nil, err
		}
		checkers[objective.Id] = checker
	}
	return (*PlayerChecker)(&checkers), nil
}

func NewTeamChecker(objectives []*repository.Objective) (*TeamChecker, error) {
	checkers := make(map[int]TeamObjectiveChecker)
	for _, objective := range objectives {
		if objective.ObjectiveType != repository.ObjectiveTypeTeam {
			continue
		}
		checker, err := GetTeamChecker(objective)
		if err != nil {
			return nil, err
		}
		checkers[objective.Id] = checker
	}
	return (*TeamChecker)(&checkers), nil
}

func (pc *PlayerChecker) CheckForCompletions(update *PlayerUpdate) []*CheckResult {
	results := make([]*CheckResult, 0)
	for id, checker := range *pc {
		new := checker(&update.New)
		if new != checker(&update.Old) {
			results = append(results, &CheckResult{
				ObjectiveId: id,
				Number:      new,
			})
		}
	}
	return results
}

func (tc *TeamChecker) CheckForCompletions(updates []*PlayerUpdate) []*CheckResult {
	results := make([]*CheckResult, 0)
	oldTeam := make([]*Player, 0)
	newTeam := make([]*Player, 0)
	for _, update := range updates {
		oldTeam = append(oldTeam, &update.Old)
		newTeam = append(newTeam, &update.New)
	}
	for id, checker := range *tc {
		new := checker(newTeam)
		if new != checker(oldTeam) {
			results = append(results, &CheckResult{
				ObjectiveId: id,
				Number:      new,
			})
		}
	}
	return results
}
