package parser

import (
	clientModel "bpl/client"
	dbModel "bpl/repository"
	"bpl/utils"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type itemChecker func(item *clientModel.Item) bool

func BoolFieldGetter(field dbModel.ItemField) (func(item *clientModel.Item) bool, error) {
	switch field {
	case dbModel.IS_CORRUPTED:
		return func(item *clientModel.Item) bool {
			if item.Corrupted != nil {
				return *item.Corrupted
			}
			return false
		}, nil
	case dbModel.IS_VAAL:
		return func(item *clientModel.Item) bool {
			if item.Hybrid != nil && item.Hybrid.IsVaalGem != nil {
				return *item.Hybrid.IsVaalGem
			}
			return false
		}, nil
	case dbModel.IS_SPLIT:
		return func(item *clientModel.Item) bool {
			if item.Split != nil {
				return *item.Split
			}
			return false
		}, nil
	case dbModel.IS_IDENTIFIED:
		return func(item *clientModel.Item) bool {
			return item.Identified
		}, nil
	default:
		return nil, fmt.Errorf("%s is not a valid boolean field", field)
	}
}

func StringFieldGetter(field dbModel.ItemField) (func(item *clientModel.Item) string, error) {
	switch field {
	case dbModel.BASE_TYPE:
		return func(item *clientModel.Item) string {
			return item.BaseType
		}, nil
	case dbModel.NAME:
		return func(item *clientModel.Item) string {
			return item.Name
		}, nil
	case dbModel.ITEM_CLASS:
		return func(item *clientModel.Item) string {
			return ItemClasses[item.BaseType]
		}, nil
	case dbModel.TYPE_LINE:
		return func(item *clientModel.Item) string {
			return item.TypeLine
		}, nil
	case dbModel.RARITY:
		return func(item *clientModel.Item) string {
			if item.Rarity != nil {
				return *item.Rarity
			}
			return ""
		}, nil
	case dbModel.SOCKETS:
		return func(item *clientModel.Item) string {
			if item.Sockets != nil {
				socketString := ""
				for _, socket := range *item.Sockets {
					if socket.SColour != nil {
						socketString += *socket.SColour
					}
				}
				return socketString
			}
			return ""
		}, nil

	case dbModel.RITUAL_MAP:
		return func(item *clientModel.Item) string {
			if item.Properties != nil {
				for _, property := range *item.Properties {
					if property.Name == "From" {
						return property.Values[0].Name()
					}
				}
			}
			return ""
		}, nil
	case dbModel.ICON_NAME:
		return func(item *clientModel.Item) string {
			parts := strings.Split(item.Icon, "/")
			if len(parts) > 0 {
				return strings.Split(parts[len(parts)-1], ".")[0]
			}
			return ""
		}, nil
	case dbModel.HEIST_TARGET:
		return func(item *clientModel.Item) string {
			if item.Properties != nil {
				for _, property := range *item.Properties {
					if property.Name == "Heist Target: {0} ({1})" {
						return property.Values[0].Name() + " (" + property.Values[1].Name() + ")"
					}
				}
			}
			return ""
		}, nil
	case dbModel.HEIST_ROGUE_REQUIREMENT:
		return func(item *clientModel.Item) string {
			if item.Properties != nil {
				for _, property := range *item.Properties {
					if property.Name == "Requires {1} (Level {0})" {
						return property.Values[0].Name() + " (Level " + property.Values[1].Name() + ")"
					}
				}
			}
			return ""
		}, nil
	default:
		return nil, fmt.Errorf("%s is not a valid string field", field)
	}
}

func StringArrayFieldGetter(field dbModel.ItemField) (func(item *clientModel.Item) []string, error) {
	switch field {
	case dbModel.ENCHANTS:
		return func(item *clientModel.Item) []string {
			if item.EnchantMods != nil {
				return *item.EnchantMods
			}
			return []string{}
		}, nil
	case dbModel.EXPLICITS:
		return func(item *clientModel.Item) []string {
			if item.ExplicitMods != nil {
				return *item.ExplicitMods
			}
			return []string{}
		}, nil
	case dbModel.IMPLICITS:
		return func(item *clientModel.Item) []string {
			if item.ImplicitMods != nil {
				return *item.ImplicitMods
			}
			return []string{}
		}, nil
	case dbModel.CRAFTED_MODS:
		return func(item *clientModel.Item) []string {
			if item.CraftedMods != nil {
				return *item.CraftedMods
			}
			return []string{}
		}, nil
	case dbModel.FRACTURED_MODS:
		return func(item *clientModel.Item) []string {
			if item.FracturedMods != nil {
				return *item.FracturedMods
			}
			return []string{}
		}, nil
	case dbModel.SANCTUM_MODS:
		return func(item *clientModel.Item) []string {
			mods := make([]string, 0)
			if item.Properties != nil {
				for _, property := range *item.Properties {
					if utils.Contains([]string{"Minor Afflictions", "Major Afflictions", "Minor Boons", "Major Boons"}, property.Name) {
						for _, value := range property.Values {
							mods = append(mods, value.Name())
						}
					}
				}
			}
			return mods
		}, nil
	case dbModel.TEMPLE_ROOMS:
		return func(item *clientModel.Item) []string {
			rooms := make([]string, 0)
			if item.AdditionalProperties != nil {
				for _, property := range *item.AdditionalProperties {
					if property.Type != nil && *property.Type == 49 {
						// we can also only look for open rooms by requiring value.ID == 0
						for _, value := range property.Values {
							rooms = append(rooms, strings.Split(value.Name(), " (Tier")[0])
						}
					}
				}
			}
			return rooms
		}, nil
	case dbModel.RITUAL_BOSSES:
		return func(item *clientModel.Item) []string {
			if item.Properties != nil {
				for _, property := range *item.Properties {
					if property.Name == "Monsters:\n{0}" {
						return strings.Split(property.Values[0].Name(), "\n")
					}
				}
			}
			return make([]string, 0)
		}, nil
	case dbModel.INFLUENCES:
		return func(item *clientModel.Item) []string {
			influences := make([]string, 0)
			if item.Influences != nil {
				for influence := range *item.Influences {
					influences = append(influences, influence)
				}
			}
			return influences
		}, nil
	default:
		return nil, fmt.Errorf("%s is not a valid string array field", field)
	}
}

func IntFieldGetter(field dbModel.ItemField) (func(item *clientModel.Item) int, error) {
	switch field {
	case dbModel.ILVL:
		return func(item *clientModel.Item) int {
			return item.Ilvl
		}, nil
	case dbModel.FRAME_TYPE:
		return func(item *clientModel.Item) int {
			if item.FrameType != nil {
				return *item.FrameType
			}
			return 0
		}, nil
	case dbModel.TALISMAN_TIER:
		return func(item *clientModel.Item) int {
			if item.TalismanTier != nil {
				return *item.TalismanTier
			}
			return 0
		}, nil
	case dbModel.MAP_TIER:
		return func(item *clientModel.Item) int {
			if item.Properties != nil {
				for _, property := range *item.Properties {
					if property.Name == "Map Tier" {
						tier, err := strconv.Atoi(property.Values[0].Name())
						if err != nil {
							log.Printf("Error parsing map tier %s", property.Values[0].Name())
							return 0
						}
						return tier
					}
				}
			}
			return 0
		}, nil
	case dbModel.MAP_QUANT:
		return func(item *clientModel.Item) int {
			if item.Properties != nil {
				for _, property := range *item.Properties {
					if property.Name == "Item Quantity" {
						quantity, err := strconv.Atoi(strings.ReplaceAll(strings.ReplaceAll(property.Values[0].Name(), "%", ""), "+", ""))
						if err != nil {
							log.Printf("Error parsing map quantity %s", property.Values[0].Name())
							return 0
						}
						return quantity
					}
				}
			}
			return 0
		}, nil
	case dbModel.MAP_RARITY:
		return func(item *clientModel.Item) int {
			if item.Properties != nil {
				for _, property := range *item.Properties {
					if property.Name == "Map Rarity" {
						rarity, err := strconv.Atoi(strings.ReplaceAll(strings.ReplaceAll(property.Values[0].Name(), "%", ""), "+", ""))
						if err != nil {
							log.Printf("Error parsing map rarity %s", property.Values[0].Name())
							return 0
						}
						return rarity
					}
				}
			}
			return 0
		}, nil
	case dbModel.MAP_PACK_SIZE:
		return func(item *clientModel.Item) int {
			if item.Properties != nil {
				for _, property := range *item.Properties {
					if property.Name == "Monster Pack Size" {
						size, err := strconv.Atoi(strings.ReplaceAll(strings.ReplaceAll(property.Values[0].Name(), "%", ""), "+", ""))
						if err != nil {
							log.Printf("Error parsing monster pack size %s", property.Values[0].Name())
							return 0
						}
						return size
					}
				}
			}
			return 0
		}, nil
	case dbModel.INCUBATOR_KILLS:
		return func(item *clientModel.Item) int {
			if item.IncubatedItem != nil {
				return item.IncubatedItem.Progress
			}
			return 0
		}, nil

	case dbModel.FACETOR_LENS_EXP:
		return func(item *clientModel.Item) int {
			if item.Properties != nil {
				for _, property := range *item.Properties {
					if property.Name == "Stored Experience: {0}" {
						exp, err := strconv.Atoi(property.Values[0].Name())
						if err != nil {
							log.Printf("Error parsing facetor lens exp %s", property.Values[0].Name())
							return 0
						}
						return exp
					}
				}
			}
			return 0
		}, nil
	case dbModel.QUALITY:
		return func(item *clientModel.Item) int {
			if item.Properties != nil {
				for _, property := range *item.Properties {
					if strings.Contains(property.Name, "Quality") {
						quality, err := strconv.Atoi(strings.ReplaceAll(strings.ReplaceAll(property.Values[0].Name(), "%", ""), "+", ""))
						if err != nil {
							log.Printf("Error parsing quality %s", property.Values[0].Name())
							return 0
						}
						return quality
					}
				}
			}
			return 0
		}, nil
	case dbModel.LEVEL:
		return func(item *clientModel.Item) int {
			if item.Properties != nil {
				for _, property := range *item.Properties {
					if property.Name == "Level" {
						level, err := strconv.Atoi(strings.ReplaceAll(property.Values[0].Name(), " (Max)", ""))
						if err != nil {
							log.Printf("Error parsing level %s", property.Values[0].Name())
							return 0
						}
						return level
					}
				}
			}
			return 0
		}, nil
	case dbModel.MEMORY_STRANDS:
		return func(item *clientModel.Item) int {
			if item.Properties != nil {
				for _, property := range *item.Properties {
					if property.Name == "Memory Strands" {
						strands, err := strconv.Atoi(property.Values[0].Name())
						if err != nil {
							log.Printf("Error parsing memory strands %s", property.Values[0].Name())
							return 0
						}
						return strands
					}
				}
			}
			return 0
		}, nil
	default:
		return nil, fmt.Errorf("%s is not a valid integer field", field)
	}
}

func StringToBool(value string) bool {
	if value == "true" || value == "True" || value == "1" {
		return true
	}
	return false
}

func BoolComparator(condition *dbModel.Condition) (itemChecker, error) {
	getter, err := BoolFieldGetter(condition.Field)
	if err != nil {
		return nil, err
	}
	value := StringToBool(condition.Value)
	switch condition.Operator {
	case dbModel.EQ:
		return func(item *clientModel.Item) bool {
			return getter(item) == value
		}, nil
	case dbModel.NEQ:
		return func(item *clientModel.Item) bool {
			return getter(item) != value
		}, nil
	default:
		return nil, fmt.Errorf("%s is an invalid operator for boolean field %s", condition.Operator, condition.Field)
	}
}

func IntComparator(condition *dbModel.Condition) (itemChecker, error) {
	getter, err := IntFieldGetter(condition.Field)
	if err != nil {
		return nil, err
	}
	var values = strings.Split(condition.Value, ",")
	intValues := make([]int, len(values))
	for i, v := range values {
		intValue, err := strconv.Atoi(v)
		if err != nil {
			return nil, err
		}
		intValues[i] = intValue
	}
	intValue := intValues[0]

	switch condition.Operator {
	case dbModel.EQ:
		return func(item *clientModel.Item) bool {
			return getter(item) == intValue
		}, nil
	case dbModel.NEQ:
		return func(item *clientModel.Item) bool {
			return getter(item) != intValue
		}, nil
	case dbModel.GT:
		return func(item *clientModel.Item) bool {
			return getter(item) > intValue
		}, nil
	case dbModel.LT:
		return func(item *clientModel.Item) bool {
			return getter(item) < intValue
		}, nil
	case dbModel.IN:
		return func(item *clientModel.Item) bool {
			fiedValue := getter(item)
			for _, v := range intValues {
				if fiedValue == v {
					return true
				}
			}
			return false
		}, nil
	case dbModel.NOT_IN:
		return func(item *clientModel.Item) bool {
			fiedValue := getter(item)
			for _, v := range intValues {
				if fiedValue == v {
					return false
				}
			}
			return true
		}, nil
	default:
		return nil, fmt.Errorf("%s is an invalid operator for integer field %s", condition.Operator, condition.Field)
	}
}

func StringComparator(condition *dbModel.Condition) (itemChecker, error) {
	getter, err := StringFieldGetter(condition.Field)
	if err != nil {
		return nil, err
	}

	switch condition.Operator {
	case dbModel.EQ:
		return func(item *clientModel.Item) bool {
			return getter(item) == condition.Value
		}, nil
	case dbModel.NEQ:
		return func(item *clientModel.Item) bool {
			return getter(item) != condition.Value
		}, nil
	case dbModel.IN:
		var values = strings.Split(condition.Value, ",")
		return func(item *clientModel.Item) bool {
			fiedValue := getter(item)
			for _, v := range values {
				if fiedValue == v {
					return true
				}
			}
			return false
		}, nil
	case dbModel.NOT_IN:
		var values = strings.Split(condition.Value, ",")
		return func(item *clientModel.Item) bool {
			fiedValue := getter(item)
			for _, v := range values {
				if fiedValue == v {
					return false
				}
			}
			return true
		}, nil
	case dbModel.MATCHES:
		var expression = regexp.MustCompile(condition.Value)
		return func(item *clientModel.Item) bool {
			return expression.MatchString(getter(item))
		}, nil
	case dbModel.CONTAINS:
		return func(item *clientModel.Item) bool {
			return strings.Contains(getter(item), condition.Value)
		}, nil
	case dbModel.LENGTH_EQ:
		length, err := strconv.Atoi(condition.Value)
		if err != nil {
			return nil, err
		}
		return func(item *clientModel.Item) bool {
			return len(getter(item)) == length
		}, nil
	case dbModel.LENGTH_GT:
		length, err := strconv.Atoi(condition.Value)
		if err != nil {
			return nil, err
		}
		return func(item *clientModel.Item) bool {
			return len(getter(item)) > length
		}, nil
	case dbModel.LENGTH_LT:
		length, err := strconv.Atoi(condition.Value)
		if err != nil {
			return nil, err
		}
		return func(item *clientModel.Item) bool {
			return len(getter(item)) < length
		}, nil
	case dbModel.DOES_NOT_MATCH:
		var expression = regexp.MustCompile(condition.Value)
		return func(item *clientModel.Item) bool {
			return !expression.MatchString(getter(item))
		}, nil
	default:
		return nil, fmt.Errorf("%s is an invalid operator for string field %s", condition.Operator, condition.Field)
	}
}

func StringArrayComparator(condition *dbModel.Condition) (itemChecker, error) {
	getter, err := StringArrayFieldGetter(condition.Field)
	if err != nil {
		return nil, err
	}
	switch condition.Operator {
	case dbModel.CONTAINS:
		return func(item *clientModel.Item) bool {
			for _, actualValue := range getter(item) {
				if strings.Contains(actualValue, condition.Value) {
					return true
				}
			}
			return false
		}, nil
	case dbModel.CONTAINS_ALL:
		values := utils.Map(strings.Split(condition.Value, ","), func(s string) string {
			return strings.Trim(s, " ")
		})
		return func(item *clientModel.Item) bool {
			for _, expectedValue := range values {
				found := false
				for _, actualValue := range getter(item) {
					if strings.Contains(actualValue, expectedValue) {
						found = true
						break
					}
				}
				if !found {
					return false
				}
			}
			return true
		}, nil
	case dbModel.CONTAINS_MATCH:
		expression := regexp.MustCompile(condition.Value)
		return func(item *clientModel.Item) bool {
			for _, actualValues := range getter(item) {
				if expression.MatchString(actualValues) {
					return true
				}
			}
			return false
		}, nil
	case dbModel.LENGTH_EQ:
		length, err := strconv.Atoi(condition.Value)
		if err != nil {
			return nil, err
		}
		return func(item *clientModel.Item) bool {
			return len(getter(item)) == length
		}, nil
	case dbModel.LENGTH_GT:
		length, err := strconv.Atoi(condition.Value)
		if err != nil {
			return nil, err
		}
		return func(item *clientModel.Item) bool {
			return len(getter(item)) > length
		}, nil
	case dbModel.LENGTH_LT:
		length, err := strconv.Atoi(condition.Value)
		if err != nil {
			return nil, err
		}
		return func(item *clientModel.Item) bool {
			return len(getter(item)) < length
		}, nil
	case dbModel.DOES_NOT_MATCH:
		expression := regexp.MustCompile(condition.Value)
		return func(item *clientModel.Item) bool {
			for _, actualValue := range getter(item) {
				if expression.MatchString(actualValue) {
					return false
				}
			}
			return true
		}, nil
	default:
		return nil, fmt.Errorf("%s is an invalid operator for string array field %s", condition.Operator, condition.Field)
	}
}

func Comparator(condition *dbModel.Condition) (itemChecker, error) {
	switch dbModel.FieldToType[condition.Field] {
	case dbModel.Bool:
		return BoolComparator(condition)
	case dbModel.String:
		return StringComparator(condition)
	case dbModel.StringArray:
		return StringArrayComparator(condition)
	case dbModel.Int:
		return IntComparator(condition)
	default:
		return nil, fmt.Errorf("Comparator: invalid field type %s", condition.Field)
	}
}

func ComperatorFromConditions(conditions []*dbModel.Condition) (itemChecker, error) {
	if len(conditions) == 0 {
		return func(item *clientModel.Item) bool {
			return true
		}, nil
	}
	if len(conditions) == 1 {
		return Comparator(conditions[0])
	}
	checkers := make([]itemChecker, len(conditions))
	for i, condition := range conditions {
		checker, err := Comparator(condition)
		if err != nil {
			return nil, err
		}
		checkers[i] = checker
	}
	return func(item *clientModel.Item) bool {
		for _, checker := range checkers {
			if !checker(item) {
				return false
			}
		}
		return true
	}, nil
}

type Discriminator struct {
	field dbModel.ItemField
	value string
}

func GetDiscriminators(conditions []*dbModel.Condition) ([]*Discriminator, []*dbModel.Condition, error) {
	for i, condition := range conditions {
		if condition.Field == dbModel.BASE_TYPE || condition.Field == dbModel.NAME || condition.Field == dbModel.ITEM_CLASS {
			if condition.Operator == dbModel.EQ {
				discriminators := []*Discriminator{
					{field: condition.Field, value: condition.Value},
				}
				remainingConditions := append(conditions[:i], conditions[i+1:]...)
				return discriminators, remainingConditions, nil
			}
			if condition.Operator == dbModel.IN {
				values := strings.Split(condition.Value, ",")
				discriminators := make([]*Discriminator, 0, len(values))
				for _, value := range values {
					discriminators = append(discriminators, &Discriminator{field: condition.Field, value: value})
				}
				remainingConditions := append(conditions[:i], conditions[i+1:]...)
				return discriminators, remainingConditions, nil
			}
		}
	}
	return nil, conditions, fmt.Errorf("at least one condition must be an equality/in condition on the baseType or name field")
}

func ValidateConditions(conditions []*dbModel.Condition) error {
	if _, _, err := GetDiscriminators(conditions); err != nil {
		return err
	}
	for _, condition := range conditions {
		if _, err := Comparator(condition); err != nil {
			return err
		}
	}
	return nil
}

type ItemObjectiveChecker struct {
	ObjectiveId int
	Function    itemChecker
	ValidFrom   *time.Time
	ValidTo     *time.Time
}

func (oc *ItemObjectiveChecker) Check(item *clientModel.Item) bool {
	now := time.Now()
	if (oc.ValidFrom != nil && oc.ValidFrom.After(now)) || (oc.ValidTo != nil && oc.ValidTo.Before(now)) {
		return false
	}
	return oc.Function(item)
}

type CheckResult struct {
	ObjectiveId int
	Number      int
}

type ItemChecker struct {
	Funcmap map[dbModel.ItemField]map[string][]*ItemObjectiveChecker
}

func NewItemChecker(objectives []*dbModel.Objective) (*ItemChecker, error) {
	funcMap := map[dbModel.ItemField]map[string][]*ItemObjectiveChecker{
		dbModel.BASE_TYPE:  make(map[string][]*ItemObjectiveChecker),
		dbModel.NAME:       make(map[string][]*ItemObjectiveChecker),
		dbModel.ITEM_CLASS: make(map[string][]*ItemObjectiveChecker),
	}
	for _, objective := range objectives {
		if objective.ObjectiveType != dbModel.ObjectiveTypeItem {
			continue
		}
		discriminators, remainingConditions, err := GetDiscriminators(objective.Conditions)
		if err != nil {
			fmt.Printf("Error getting discriminators for objective %d-%s: %s\n", objective.Id, objective.Name, err)
			continue
			// return nil, err
		}
		fn, err := ComperatorFromConditions(remainingConditions)
		if err != nil {
			fmt.Printf("Error getting comperator for objective %d: %s\n", objective.Id, err)
			continue
			// return nil, err
		}
		for _, discriminator := range discriminators {
			if valueToChecker, ok := funcMap[discriminator.field]; ok {
				valueToChecker[discriminator.value] = append(
					valueToChecker[discriminator.value],
					&ItemObjectiveChecker{
						ObjectiveId: objective.Id,
						Function:    fn,
					})
			} else {
				return nil, fmt.Errorf("invalid discriminator field")
			}

		}
	}

	return &ItemChecker{
		Funcmap: funcMap,
	}, nil
}

func (ic *ItemChecker) CheckForCompletions(item *clientModel.Item) []*CheckResult {
	results := make([]*CheckResult, 0)
	if checkers, ok := ic.Funcmap[dbModel.BASE_TYPE][item.BaseType]; ok {
		results = append(results, applyCheckers(checkers, item)...)
	}
	if checkers, ok := ic.Funcmap[dbModel.NAME][item.Name]; ok {
		results = append(results, applyCheckers(checkers, item)...)
	}
	if checkers, ok := ic.Funcmap[dbModel.ITEM_CLASS][ItemClasses[item.BaseType]]; ok {
		results = append(results, applyCheckers(checkers, item)...)
	}
	return results
}

func applyCheckers(checkers []*ItemObjectiveChecker, item *clientModel.Item) []*CheckResult {
	results := make([]*CheckResult, 0)
	// sort out foiled items
	if item.FrameType != nil && *item.FrameType == 10 {
		return results
	}
	for _, checker := range checkers {
		if checker.Check(item) {
			number := 1
			if item.StackSize != nil {
				number = *item.StackSize
			}
			results = append(results, &CheckResult{
				ObjectiveId: checker.ObjectiveId,
				Number:      number,
			})
		}
	}
	return results
}
