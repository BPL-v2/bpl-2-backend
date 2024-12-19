package parser

import (
	clientModel "bpl/client"
	dbModel "bpl/repository"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type checkerFun func(item *clientModel.Item) bool

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
	default:
		return nil, fmt.Errorf("%s is not a valid string field", field)
	}
}

func StringArrayFieldGetter(field dbModel.ItemField) (func(item *clientModel.Item) []string, error) {
	switch field {
	case dbModel.ENCHANT_MODS:
		return func(item *clientModel.Item) []string {
			if item.EnchantMods != nil {
				return *item.EnchantMods
			}
			return []string{}
		}, nil
	case dbModel.EXPLICIT_MODS:
		return func(item *clientModel.Item) []string {
			if item.ExplicitMods != nil {
				return *item.ExplicitMods
			}
			return []string{}
		}, nil
	case dbModel.IMPLICIT_MODS:
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
	default:
		return nil, fmt.Errorf("%s is not a valid integer field", field)
	}
}

var fieldToComparator = map[dbModel.ItemField]func(*dbModel.Condition) (checkerFun, error){
	dbModel.BASE_TYPE: StringComparator,
	dbModel.NAME:      StringComparator,
	dbModel.TYPE_LINE: StringComparator,
	dbModel.RARITY:    StringComparator,

	dbModel.ILVL:          IntComparator,
	dbModel.FRAME_TYPE:    IntComparator,
	dbModel.TALISMAN_TIER: IntComparator,

	dbModel.ENCHANT_MODS:   StringArrayComparator,
	dbModel.EXPLICIT_MODS:  StringArrayComparator,
	dbModel.IMPLICIT_MODS:  StringArrayComparator,
	dbModel.CRAFTED_MODS:   StringArrayComparator,
	dbModel.FRACTURED_MODS: StringArrayComparator,
}

func IntComparator(condition *dbModel.Condition) (checkerFun, error) {
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
	case dbModel.GTE:
		return func(item *clientModel.Item) bool {
			return getter(item) >= intValue
		}, nil
	case dbModel.LT:
		return func(item *clientModel.Item) bool {
			return getter(item) < intValue
		}, nil
	case dbModel.LTE:
		return func(item *clientModel.Item) bool {
			return getter(item) <= intValue
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

func StringComparator(condition *dbModel.Condition) (checkerFun, error) {
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
	default:
		return nil, fmt.Errorf("%s is an invalid operator for string field %s", condition.Operator, condition.Field)
	}
}

func StringArrayComparator(condition *dbModel.Condition) (checkerFun, error) {
	getter, err := StringArrayFieldGetter(condition.Field)
	if err != nil {
		return nil, err
	}
	values := strings.Split(condition.Value, ",")
	switch condition.Operator {
	case dbModel.CONTAINS:
		return func(item *clientModel.Item) bool {
			for _, fv := range getter(item) {
				if fv == condition.Value {
					return true
				}
			}
			return false
		}, nil
	case dbModel.CONTAINS_ALL:
		return func(item *clientModel.Item) bool {
			fieldValues := getter(item)
			for _, v := range values {
				found := false
				for _, fv := range fieldValues {
					if fv == v {
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
			for _, fv := range getter(item) {
				if expression.MatchString(fv) {
					return true
				}
			}
			return false
		}, nil
	case dbModel.CONTAINS_ALL_MATCHES:
		expressions := make([]*regexp.Regexp, len(values))
		for i, v := range values {
			expressions[i] = regexp.MustCompile(v)
		}
		return func(item *clientModel.Item) bool {
			fieldValues := getter(item)
			for _, expression := range expressions {
				found := false
				for _, fv := range fieldValues {
					if expression.MatchString(fv) {
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
	default:
		return nil, fmt.Errorf("%s is an invalid operator for string array field %s", condition.Operator, condition.Field)
	}
}

func Comparator(condition *dbModel.Condition) (checkerFun, error) {
	if f, ok := fieldToComparator[condition.Field]; ok {
		return f(condition)
	}
	return nil, fmt.Errorf("Comparator: invalid field %s", condition.Field)
}

func ComperatorFromConditions(conditions []*dbModel.Condition) (checkerFun, error) {
	if len(conditions) == 0 {
		return func(item *clientModel.Item) bool {
			return true
		}, nil
	}
	if len(conditions) == 1 {
		return Comparator(conditions[0])
	}
	checkers := make([]checkerFun, len(conditions))
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
		if condition.Field == dbModel.BASE_TYPE || condition.Field == dbModel.NAME {
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

type ObjectiveChecker struct {
	ObjectiveId int
	Function    checkerFun
}

type CheckResult struct {
	ObjectiveId int
	Number      int
}

type ItemChecker struct {
	Funcmap map[dbModel.ItemField]map[string][]*ObjectiveChecker
}

func NewItemChecker(objectives []*dbModel.Objective) (*ItemChecker, error) {
	funcMap := map[dbModel.ItemField]map[string][]*ObjectiveChecker{
		dbModel.BASE_TYPE: make(map[string][]*ObjectiveChecker),
		dbModel.NAME:      make(map[string][]*ObjectiveChecker),
	}
	for _, objective := range objectives {
		if objective.ObjectiveType != dbModel.ITEM {
			continue
		}
		discriminators, remainingConditions, err := GetDiscriminators(objective.Conditions)
		if err != nil {
			return nil, err
		}
		fn, err := ComperatorFromConditions(remainingConditions)
		if err != nil {
			return nil, err
		}
		for _, discriminator := range discriminators {
			if valueToChecker, ok := funcMap[discriminator.field]; ok {
				valueToChecker[discriminator.value] = append(
					valueToChecker[discriminator.value],
					&ObjectiveChecker{
						ObjectiveId: objective.ID,
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
	if checkers, ok := ic.Funcmap[dbModel.BASE_TYPE][item.BaseType]; ok {
		return applyCheckers(checkers, item)
	}
	if checkers, ok := ic.Funcmap[dbModel.NAME][item.Name]; ok {
		return applyCheckers(checkers, item)
	}
	return make([]*CheckResult, 0)
}

func applyCheckers(checkers []*ObjectiveChecker, item *clientModel.Item) []*CheckResult {
	results := make([]*CheckResult, 0)
	for _, checker := range checkers {
		if checker.Function(item) {
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
