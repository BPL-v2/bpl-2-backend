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

func IntComparator(comparison *dbModel.Condition) (checkerFun, error) {
	getter, err := IntFieldGetter(comparison.Field)
	if err != nil {
		return nil, err
	}
	var values = strings.Split(comparison.Value, ",")
	intValues := make([]int, len(values))
	for i, v := range values {
		intValue, err := strconv.Atoi(v)
		if err != nil {
			return nil, err
		}
		intValues[i] = intValue
	}
	intValue := intValues[0]

	switch comparison.Operator {
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
		return nil, fmt.Errorf("%s is an invalid operator for integer field %s", comparison.Operator, comparison.Field)
	}
}

func StringComparator(comparison *dbModel.Condition) (checkerFun, error) {
	getter, err := StringFieldGetter(comparison.Field)
	if err != nil {
		return nil, err
	}

	switch comparison.Operator {
	case dbModel.EQ:
		return func(item *clientModel.Item) bool {
			return getter(item) == comparison.Value
		}, nil
	case dbModel.NEQ:
		return func(item *clientModel.Item) bool {
			return getter(item) != comparison.Value
		}, nil
	case dbModel.IN:
		var values = strings.Split(comparison.Value, ",")
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
		var values = strings.Split(comparison.Value, ",")
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
		var expression = regexp.MustCompile(comparison.Value)
		return func(item *clientModel.Item) bool {
			return expression.MatchString(getter(item))
		}, nil
	default:
		return nil, fmt.Errorf("%s is an invalid operator for string field %s", comparison.Operator, comparison.Field)
	}
}

func StringArrayComparator(comparison *dbModel.Condition) (checkerFun, error) {
	getter, err := StringArrayFieldGetter(comparison.Field)
	if err != nil {
		return nil, err
	}
	values := strings.Split(comparison.Value, ",")
	switch comparison.Operator {
	case dbModel.CONTAINS:
		return func(item *clientModel.Item) bool {
			for _, fv := range getter(item) {
				if fv == comparison.Value {
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
		expression := regexp.MustCompile(comparison.Value)
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
		return nil, fmt.Errorf("%s is an invalid operator for string array field %s", comparison.Operator, comparison.Field)
	}
}

func Comparator(comparison *dbModel.Condition) (checkerFun, error) {
	if f, ok := fieldToComparator[comparison.Field]; ok {
		return f(comparison)
	}
	return nil, fmt.Errorf("Comparator: invalid field %s", comparison.Field)
}

func AndComparator(comparisons []*dbModel.Condition) (checkerFun, error) {
	if len(comparisons) == 0 {
		return func(item *clientModel.Item) bool {
			return true
		}, nil
	}
	if len(comparisons) == 1 {
		return Comparator(comparisons[0])
	}
	checkers := make([]checkerFun, len(comparisons))
	for i, comparison := range comparisons {
		checker, err := Comparator(comparison)
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
	for i, comparison := range conditions {
		if comparison.Field == dbModel.BASE_TYPE || comparison.Field == dbModel.NAME {
			if comparison.Operator == dbModel.EQ {

				discriminators := []*Discriminator{
					{field: comparison.Field, value: comparison.Value},
				}
				remainingConditions := append(conditions[:i], conditions[i+1:]...)
				return discriminators, remainingConditions, nil
			}
			if comparison.Operator == dbModel.IN {
				values := strings.Split(comparison.Value, ",")
				discriminators := make([]*Discriminator, 0, len(values))
				for _, value := range values {
					discriminators = append(discriminators, &Discriminator{field: comparison.Field, value: value})
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
