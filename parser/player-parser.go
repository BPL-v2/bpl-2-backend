package parser

import (
	"bpl/client"
	"bpl/repository"
	"bpl/utils"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Player struct {
	AtlasPassiveTrees []client.AtlasPassiveTree
	DelveDepth        int
	Character         *client.Character
	PoB               *repository.CharacterPob
}

type PlayerUpdate struct {
	UserId           int
	AccountName      string
	TeamId           int
	Token            string
	TokenExpiry      time.Time
	Mu               sync.Mutex
	SuccessiveErrors int
	LastActive       time.Time

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

func (p *PlayerUpdate) ShouldUpdateCharacterName(timings map[repository.TimingKey]time.Duration) bool {
	if !p.CanMakeRequests() {
		return false
	}
	return time.Since(p.LastUpdateTimes.CharacterName) > timings[repository.CharacterNameRefetchDelay]
}

func (p *PlayerUpdate) ShouldUpdateCharacter(timings map[repository.TimingKey]time.Duration) bool {
	if !p.CanMakeRequests() {
		return false
	}
	if p.New.Character.Name == "" {
		return false
	}
	if p.LastActive.Before(time.Now().Add(-timings[repository.InactivityDuration])) {
		return time.Since(p.LastUpdateTimes.Character) > timings[repository.CharacterRefetchDelayInactive]
	}
	if p.New.Character.Level > 40 && !p.New.Character.HasPantheon() {
		return time.Since(p.LastUpdateTimes.Character) > timings[repository.CharacterRefetchDelayImportant]
	}
	if p.New.Character.Level > 68 && p.New.Character.GetAscendancyPoints() < 8 {
		return time.Since(p.LastUpdateTimes.Character) > timings[repository.CharacterRefetchDelayImportant]
	}
	return time.Since(p.LastUpdateTimes.Character) > timings[repository.CharacterRefetchDelay]
}

func (p *PlayerUpdate) ShouldUpdateLeagueAccount(timings map[repository.TimingKey]time.Duration) bool {
	if !p.CanMakeRequests() {
		return false
	}
	if p.New.Character.Level < 55 {
		return false
	}
	if p.LastActive.Before(time.Now().Add(-timings[repository.InactivityDuration])) {
		return time.Since(p.LastUpdateTimes.LeagueAccount) > timings[repository.LeagueAccountRefetchDelayInactive]
	}

	if p.New.MaxAtlasTreeNodes() < 100 {
		return time.Since(p.LastUpdateTimes.LeagueAccount) > timings[repository.LeagueAccountRefetchDelayImportant]
	}

	return time.Since(p.LastUpdateTimes.LeagueAccount) > timings[repository.LeagueAccountRefetchDelay]
}

type TeamObjectiveChecker func(p []*Player) int

type PlayerObjectiveChecker func(p *Player) int

var rareAscendancies = []string{
	"Assassin",
	"Juggernaut",
	"Gladiator",
	"Trickster",
	"Guardian",
	"Champion",
	"Occultist",
	"Warden",
	"Inquisitor",
	"Saboteur",
	"Ascendant",
}

func GetPlayerChecker(objective *repository.Objective) (PlayerObjectiveChecker, error) {
	if (objective.ObjectiveType != repository.ObjectiveTypePlayer) && (objective.ObjectiveType != repository.ObjectiveTypeTeam) {
		return nil, fmt.Errorf("not a player objective")
	}
	switch objective.NumberField {
	case repository.NumberFieldPlayerLevel:
		return func(p *Player) int {
			if p.Character == nil {
				return 0
			}
			return p.Character.Level
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
			count := 0
			if p.Character == nil {
				return count
			}
			if p.Character.Passives.PantheonMajor != nil {
				count++
			}
			if p.Character.Passives.PantheonMinor != nil {
				count++
			}
			return count
		}, nil
	case repository.NumberFieldAscendancy:
		return func(p *Player) int {
			if p.Character == nil {
				return 0
			}
			return p.Character.GetAscendancyPoints()
		}, nil
	case repository.NumberFieldFullyAscended:
		return func(p *Player) int {
			if p.Character == nil || p.Character.GetAscendancyPoints() < 8 {
				return 0
			}
			return 1
		}, nil
	case repository.NumberFieldPlayerScore:
		return func(p *Player) int {
			score := 0
			if p.Character == nil {
				return 0
			}
			ascendancyPoints := p.Character.GetAscendancyPoints()
			if p.Character.Level >= 40 {
				score += 1
			}
			if p.Character.Level >= 60 {
				score += 1
			}
			if p.Character.Level >= 80 {
				score += 1
			}
			if p.Character.Level >= 90 {
				score += 3
			}
			if ascendancyPoints >= 4 {
				score += 1
			}
			if ascendancyPoints >= 6 {
				score += 1
			}
			if ascendancyPoints >= 8 {
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
	case repository.NumberFieldInfluenceEquipped:
		return func(p *Player) int {
			return itemCount(p.Character, func(item client.Item) bool {
				return item.Influences != nil && len(*item.Influences) > 0
			})
		}, nil
	case repository.NumberFieldFoulbornEquipped:
		return func(p *Player) int {
			return itemCount(p.Character, func(item client.Item) bool {
				return item.Mutated != nil
			})
		}, nil
	case repository.NumberFieldGemsEquipped:
		return func(p *Player) int {
			if p.Character == nil || p.Character.Equipment == nil {
				return 0
			}
			count := 0
			for _, item := range *p.Character.Equipment {
				if item.SocketedItems != nil {
					for _, socketed := range *item.SocketedItems {
						if socketed.AbyssJewel == nil {
							count++
						}
					}
				}
			}
			return count
		}, nil
	case repository.NumberFieldCorruptedItemsEquipped:
		return func(p *Player) int {
			return itemCount(p.Character, func(item client.Item) bool {
				return item.Corrupted != nil
			})
		}, nil
	case repository.NumberFieldJewelsWithImplicitsEquipped:
		return func(p *Player) int {
			return itemCount(p.Character, func(item client.Item) bool {
				return strings.HasSuffix(item.BaseType, "Jewel") && item.ImplicitMods != nil && len(*item.ImplicitMods) > 0
			})
		}, nil
	case repository.NumberFieldAtlasPoints:
		return func(p *Player) int {
			total := 0
			for _, tree := range p.AtlasPassiveTrees {
				points := len(tree.Hashes)
				if slices.Contains(tree.Hashes, 65225) {
					points -= 20
				}
				total = utils.Max(total, points)
			}
			return total
		}, nil
	case repository.NumberFieldArmourQuality:
		return func(p *Player) int {
			return quality(p.Character, "Armour")
		}, nil
	case repository.NumberFieldWeaponQuality:
		return func(p *Player) int {
			return quality(p.Character, "Weapon")
		}, nil
	case repository.NumberFieldFlaskQuality:
		return func(p *Player) int {
			return quality(p.Character, "Flask")
		}, nil
	case repository.NumberFieldEvasion:
		return func(p *Player) int {
			if p.PoB == nil {
				return 0
			}
			return int(p.PoB.Evasion)
		}, nil
	case repository.NumberFieldArmour:
		return func(p *Player) int {
			if p.PoB == nil {
				return 0
			}
			return int(p.PoB.Armour)
		}, nil
	case repository.NumberFieldEnergyShield:
		return func(p *Player) int {
			if p.PoB == nil {
				return 0
			}
			return int(p.PoB.ES)
		}, nil
	case repository.NumberFieldMana:
		return func(p *Player) int {
			if p.PoB == nil {
				return 0
			}
			return int(p.PoB.Mana)
		}, nil
	case repository.NumberFieldHP:
		return func(p *Player) int {
			if p.PoB == nil {
				return 0
			}
			return int(p.PoB.HP)
		}, nil

	case repository.NumberFieldEHP:
		return func(p *Player) int {
			if p.PoB == nil {
				return 0
			}
			return int(p.PoB.EHP)
		}, nil
	case repository.NumberFieldPhysMaxHit:
		return func(p *Player) int {
			if p.PoB == nil {
				return 0
			}
			return int(p.PoB.PhysMaxHit)
		}, nil
	case repository.NumberFieldEleMaxHit:
		return func(p *Player) int {
			if p.PoB == nil {
				return 0
			}
			return int(p.PoB.EleMaxHit)
		}, nil
	case repository.NumberFieldIncMovementSpeed:
		return func(p *Player) int {
			if p.PoB == nil {
				return 0
			}

			return int(p.PoB.MovementSpeed) - 100
		}, nil
	case repository.NumberFieldFullDPS:
		return func(p *Player) int {
			if p.PoB == nil {
				return 0
			}
			return int(p.PoB.DPS)
		}, nil
	case repository.NumberFieldHasRareAscendancyPast90:
		return func(p *Player) int {
			if p.Character == nil || p.Character.Level < 90 {
				return 0
			}
			for _, rare := range rareAscendancies {
				if p.Character.Class == rare {
					return 1
				}
			}
			return 0
		}, nil
	case repository.NumberFieldBloodlineAscendancy:
		return func(p *Player) int {
			if p.Character == nil || p.Character.GetBloodlinePoints() == 0 {
				return 0
			}
			return 1
		}, nil
	default:
		return nil, fmt.Errorf("unsupported number field")
	}
}

func quality(character *client.Character, superclass string) int {
	if character == nil || character.Equipment == nil {
		return 0
	}
	totalQuality := 0
	for _, item := range *character.Equipment {
		if SuperClasses[ItemClasses[item.BaseType]] != superclass || item.Properties == nil {
			continue
		}
		for _, property := range *item.Properties {
			if strings.Contains(property.Name, "Quality") {
				quality, err := strconv.Atoi(strings.ReplaceAll(strings.ReplaceAll(property.Values[0].Name(), "%", ""), "+", ""))
				if err != nil {
					continue
				}
				totalQuality += quality
			}
		}
	}
	return totalQuality

}

func itemCount(character *client.Character, predicate func(item client.Item) bool) int {
	if character == nil {
		return 0
	}
	count := 0
	if character.Equipment != nil {
		for _, item := range *character.Equipment {
			if predicate(item) {
				count++
			}
		}
	}
	if character.Jewels != nil {
		for _, item := range *character.Jewels {
			if predicate(item) {
				count++
			}
		}
	}
	return count
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
			c := checker(player)
			sum += c
			fmt.Println("Checking player", player.Character.Name, "for team objective", objective.Name, "got", c)
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
		fmt.Println("Checking team objective", id, "old value", checker(oldTeam), "new value", new)
		if new != checker(oldTeam) {
			results = append(results, &CheckResult{
				ObjectiveId: id,
				Number:      new,
			})
		}
	}
	return results
}
