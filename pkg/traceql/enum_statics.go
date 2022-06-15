package traceql

import "fmt"

type StaticType int

const (
	typeSpanset   StaticType = iota // type used by spanset pipelines
	typeAttribute                   // a special constant that indicates the type is determined at query time by the attribute
	typeInt
	typeFloat
	typeString
	typeBoolean
	typeNil
	typeDuration
	typeStatus
)

// isMatchingOperand returns whether two types can be combined with a binary operator. the kind of operator is
// immaterial. see Operator.typesValid() for code that determines if the passed types are valid for the given
// operator.
func (t StaticType) isMatchingOperand(otherT StaticType) bool {
	if t == typeAttribute || otherT == typeAttribute {
		return true
	}

	if t == otherT {
		return true
	}

	if t.isNumeric() && otherT.isNumeric() {
		return true
	}

	return false
}

func (t StaticType) isNumeric() bool {
	return t == typeInt || t == typeFloat || t == typeDuration
}

// Status represents valid static values of typeStatus
type Status int

const (
	statusError Status = iota
	statusOk
	statusUnset
)

func (s Status) String() string {
	switch s {
	case statusError:
		return "error"
	case statusOk:
		return "ok"
	case statusUnset:
		return "unset"
	}

	return fmt.Sprintf("status(%d)", s)
}
