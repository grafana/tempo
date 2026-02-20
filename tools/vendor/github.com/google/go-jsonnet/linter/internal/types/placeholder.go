package types

import "sort"

type placeholderID int

// 0 value for placeholderID acting as "nil" for placeholders
const (
	noType placeholderID = iota
	anyType
	boolType
	numberType
	stringType
	nullType
	anyArrayType
	numberArrayType
	boolArrayType
	anyObjectType
	anyFunctionType
	stdlibType
)

type placeholderIDs []placeholderID

func (p placeholderIDs) Len() int           { return len(p) }
func (p placeholderIDs) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p placeholderIDs) Less(i, j int) bool { return p[i] < p[j] }

// We need to be very careful, because slices are mutable. It is easy
// to forget to copy a slice, especially when using append and have
// weird bugs when they're accidentally mutated in other place.
func copyPlaceholders(ps []placeholderID) []placeholderID {
	return append(ps[:0:0], ps...)
}

func normalizePlaceholders(placeholders []placeholderID) []placeholderID {
	if len(placeholders) == 0 {
		return placeholders
	}
	sort.Sort(placeholderIDs(placeholders))
	// Unique
	count := 1
	for i := 1; i < len(placeholders); i++ {
		if placeholders[i] == anyType {
			placeholders[0] = anyType
			return placeholders[:1]
		}
		if placeholders[i] != placeholders[count-1] {
			placeholders[count] = placeholders[i]
			count++
		}
	}
	// We return a slice pointing to the same underlying array - reallocation to reduce it is not what we want probably
	return placeholders[:count]
}

type typePlaceholder struct {
	// Derived from AST
	concrete TypeDesc

	contains []placeholderID

	index *indexSpec

	builtinOp *builtinOpDesc
}

func concreteTP(t TypeDesc) typePlaceholder {
	return typePlaceholder{
		concrete: t,
		contains: nil,
	}
}

func tpSum(p1, p2 placeholderID) typePlaceholder {
	return typePlaceholder{
		contains: []placeholderID{p1, p2},
	}
}

func tpIndex(index *indexSpec) typePlaceholder {
	return typePlaceholder{
		concrete: voidTypeDesc(),
		contains: nil,
		index:    index,
	}
}

func tpRef(p placeholderID) typePlaceholder {
	return typePlaceholder{
		contains: []placeholderID{p},
	}
}
