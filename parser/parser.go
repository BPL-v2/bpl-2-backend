package parser

import (
	model "bpl/model/gggmodel"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type checkerFun func(item *model.Item) bool

type ItemField int

const (
	BASE_TYPE ItemField = iota
	NAME
	TYPE_LINE
	RARITY

	ILVL
	FRAME_TYPE
	TALISMAN_TIER

	ENCHANT_MODS
	EXPLICIT_MODS
	IMPLICIT_MODS
	CRAFTED_MODS
	FRACTURED_MODS
)

func (f *ItemField) ToString() string {
	switch *f {
	case BASE_TYPE:
		return "BASE_TYPE"
	case NAME:
		return "NAME"
	case TYPE_LINE:
		return "TYPE_LINE"
	case RARITY:
		return "RARITY"
	case ILVL:
		return "ILVL"
	case FRAME_TYPE:
		return "FRAME_TYPE"
	case TALISMAN_TIER:
		return "TALISMAN_TIER"
	case ENCHANT_MODS:
		return "ENCHANT_MODS"
	case EXPLICIT_MODS:
		return "EXPLICIT_MODS"
	case IMPLICIT_MODS:
		return "IMPLICIT_MODS"
	case CRAFTED_MODS:
		return "CRAFTED_MODS"
	case FRACTURED_MODS:
		return "FRACTURED_MODS"
	default:
		return "UNKNOWN"
	}
}

type Operator int

const (
	EQ Operator = iota
	NEQ
	GT
	GTE
	LT
	LTE
	IN
	NOT_IN
	MATCHES

	CONTAINS
	CONTAINS_ALL
	CONTAINS_MATCH
	CONTAINS_ALL_MATCHES
)

func (o *Operator) ToString() string {
	switch *o {
	case EQ:
		return "EQ"
	case NEQ:
		return "NEQ"
	case GT:
		return "GT"
	case GTE:
		return "GTE"
	case LT:
		return "LT"
	case LTE:
		return "LTE"
	case IN:
		return "IN"
	case NOT_IN:
		return "NOT_IN"
	case MATCHES:
		return "MATCHES"
	case CONTAINS:
		return "CONTAINS"
	case CONTAINS_ALL:
		return "CONTAINS_ALL"
	case CONTAINS_MATCH:
		return "CONTAINS_MATCH"
	case CONTAINS_ALL_MATCHES:
		return "CONTAINS_ALL_MATCHES"
	default:
		return "UNKNOWN"
	}
}

type Condition struct {
	Field    ItemField
	Operator Operator
	Value    string
}

func StringFieldGetter(x ItemField) (func(item *model.Item) string, error) {
	switch x {
	case BASE_TYPE:
		return func(item *model.Item) string {
			return item.BaseType
		}, nil
	case NAME:
		return func(item *model.Item) string {
			return item.Name
		}, nil
	case TYPE_LINE:
		return func(item *model.Item) string {
			return item.TypeLine
		}, nil
	case RARITY:
		return func(item *model.Item) string {
			if item.Rarity != nil {
				return *item.Rarity
			}
			return ""
		}, nil
	default:
		return nil, fmt.Errorf("FieldGetter: invalid field")
	}
}

func StringArrayFieldGetter(x ItemField) (func(item *model.Item) []string, error) {
	switch x {
	case ENCHANT_MODS:
		return func(item *model.Item) []string {
			if item.EnchantMods != nil {
				return *item.EnchantMods
			}
			return []string{}
		}, nil
	case EXPLICIT_MODS:
		return func(item *model.Item) []string {
			if item.ExplicitMods != nil {
				return *item.ExplicitMods
			}
			return []string{}
		}, nil
	case IMPLICIT_MODS:
		return func(item *model.Item) []string {
			if item.ImplicitMods != nil {
				return *item.ImplicitMods
			}
			return []string{}
		}, nil
	case CRAFTED_MODS:
		return func(item *model.Item) []string {
			if item.CraftedMods != nil {
				return *item.CraftedMods
			}
			return []string{}
		}, nil
	case FRACTURED_MODS:
		return func(item *model.Item) []string {
			if item.FracturedMods != nil {
				return *item.FracturedMods
			}
			return []string{}
		}, nil
	default:
		return nil, fmt.Errorf("FieldGetter: invalid field")
	}
}

func IntFieldGetter(x ItemField) (func(item *model.Item) int, error) {
	switch x {
	case ILVL:
		return func(item *model.Item) int {
			return item.Ilvl
		}, nil
	case FRAME_TYPE:
		return func(item *model.Item) int {
			if item.FrameType != nil {
				return *item.FrameType
			}
			return 0
		}, nil
	case TALISMAN_TIER:
		return func(item *model.Item) int {
			if item.TalismanTier != nil {
				return *item.TalismanTier
			}
			return 0
		}, nil
	default:
		return nil, fmt.Errorf("FieldGetter: invalid field")
	}
}

var fieldToComparator = map[ItemField]func(*Condition) (checkerFun, error){
	BASE_TYPE: StringComparator,
	NAME:      StringComparator,
	TYPE_LINE: StringComparator,
	RARITY:    StringComparator,

	ILVL:          IntComparator,
	FRAME_TYPE:    IntComparator,
	TALISMAN_TIER: IntComparator,

	ENCHANT_MODS:   StringArrayComparator,
	EXPLICIT_MODS:  StringArrayComparator,
	IMPLICIT_MODS:  StringArrayComparator,
	CRAFTED_MODS:   StringArrayComparator,
	FRACTURED_MODS: StringArrayComparator,
}

func IntComparator(comparison *Condition) (checkerFun, error) {
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
	case EQ:
		return func(item *model.Item) bool {
			return getter(item) == intValue
		}, nil
	case NEQ:
		return func(item *model.Item) bool {
			return getter(item) != intValue
		}, nil
	case GT:
		return func(item *model.Item) bool {
			return getter(item) > intValue
		}, nil
	case GTE:
		return func(item *model.Item) bool {
			return getter(item) >= intValue
		}, nil
	case LT:
		return func(item *model.Item) bool {
			return getter(item) < intValue
		}, nil
	case LTE:
		return func(item *model.Item) bool {
			return getter(item) <= intValue
		}, nil
	case IN:
		return func(item *model.Item) bool {
			fiedValue := getter(item)
			for _, v := range intValues {
				if fiedValue == v {
					return true
				}
			}
			return false
		}, nil
	case NOT_IN:
		return func(item *model.Item) bool {
			fiedValue := getter(item)
			for _, v := range intValues {
				if fiedValue == v {
					return false
				}
			}
			return true
		}, nil
	default:
		return nil, fmt.Errorf("IntComparator: invalid operator %d", comparison.Operator)
	}
}

func StringComparator(comparison *Condition) (checkerFun, error) {
	getter, err := StringFieldGetter(comparison.Field)
	if err != nil {
		return nil, err
	}

	switch comparison.Operator {
	case EQ:
		return func(item *model.Item) bool {
			return getter(item) == comparison.Value
		}, nil
	case NEQ:
		return func(item *model.Item) bool {
			return getter(item) != comparison.Value
		}, nil
	case IN:
		var values = strings.Split(comparison.Value, ",")
		return func(item *model.Item) bool {
			fiedValue := getter(item)
			for _, v := range values {
				if fiedValue == v {
					return true
				}
			}
			return false
		}, nil
	case NOT_IN:
		var values = strings.Split(comparison.Value, ",")
		return func(item *model.Item) bool {
			fiedValue := getter(item)
			for _, v := range values {
				if fiedValue == v {
					return false
				}
			}
			return true
		}, nil
	case MATCHES:
		var expression = regexp.MustCompile(comparison.Value)
		return func(item *model.Item) bool {
			return expression.MatchString(getter(item))
		}, nil
	default:
		return nil, fmt.Errorf("StringComparator: invalid operator %d", comparison.Operator)
	}
}

func StringArrayComparator(comparison *Condition) (checkerFun, error) {
	getter, err := StringArrayFieldGetter(comparison.Field)
	if err != nil {
		return nil, err
	}
	values := strings.Split(comparison.Value, ",")
	switch comparison.Operator {
	case CONTAINS:
		return func(item *model.Item) bool {
			for _, fv := range getter(item) {
				if fv == comparison.Value {
					return true
				}
			}
			return false
		}, nil
	case CONTAINS_ALL:
		return func(item *model.Item) bool {
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
	case CONTAINS_MATCH:
		expression := regexp.MustCompile(comparison.Value)
		return func(item *model.Item) bool {
			for _, fv := range getter(item) {
				if expression.MatchString(fv) {
					return true
				}
			}
			return false
		}, nil
	case CONTAINS_ALL_MATCHES:
		expressions := make([]*regexp.Regexp, len(values))
		for i, v := range values {
			expressions[i] = regexp.MustCompile(v)
		}
		return func(item *model.Item) bool {
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
		return nil, fmt.Errorf("StringArrayComparator: invalid operator %d", comparison.Operator)
	}
}

func Comparator(comparison *Condition) (checkerFun, error) {
	if f, ok := fieldToComparator[comparison.Field]; ok {
		return f(comparison)
	}
	return nil, fmt.Errorf("Comparator: invalid field %d", comparison.Field)
}

func AndComparator(comparisons []*Condition) (checkerFun, error) {
	if len(comparisons) == 0 {
		return func(item *model.Item) bool {
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
	return func(item *model.Item) bool {
		for _, checker := range checkers {
			if !checker(item) {
				return false
			}
		}
		return true
	}, nil
}
