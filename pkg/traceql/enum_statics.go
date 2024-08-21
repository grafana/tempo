package traceql

import "fmt"

type StaticType int

const (
	TypeNil       StaticType = iota
	TypeSpanset              // type used by spanset pipelines
	TypeAttribute            // a special constant that indicates the type is determined at query time by the attribute
	TypeInt
	TypeFloat
	TypeString
	TypeBoolean
	TypeIntArray
	TypeFloatArray
	TypeStringArray
	TypeBooleanArray
	TypeDuration
	TypeStatus
	TypeKind
)

// isMatchingOperand returns whether two types can be combined with a binary operator. the kind of operator is
// immaterial. see Operator.typesValid() for code that determines if the passed types are valid for the given
// operator.
func (t StaticType) isMatchingOperand(otherT StaticType) bool {
	if t == TypeAttribute || otherT == TypeAttribute {
		return true
	}

	if t == otherT {
		return true
	}

	if t.isNumeric() && otherT.isNumeric() {
		return true
	}

	if otherT == TypeNil {
		return true
	}

	return t.isMatchingArrayElement(otherT)
}

func (t StaticType) isNumeric() bool {
	return t == TypeInt || t == TypeFloat || t == TypeDuration
}

// isMatchingArrayElement is like isMatchingOperand but for arrays
func (t StaticType) isMatchingArrayElement(otherT StaticType) bool {
	switch t {
	case TypeIntArray:
		return TypeInt.isMatchingOperand(otherT)
	case TypeFloatArray:
		return TypeFloat.isMatchingOperand(otherT)
	case TypeStringArray:
		return TypeString.isMatchingOperand(otherT)
	case TypeBooleanArray:
		return TypeBoolean.isMatchingOperand(otherT)
	}

	// make it symmetric
	switch otherT {
	case TypeIntArray:
		return TypeInt.isMatchingOperand(t)
	case TypeFloatArray:
		return TypeFloat.isMatchingOperand(t)
	case TypeStringArray:
		return TypeString.isMatchingOperand(t)
	case TypeBooleanArray:
		return TypeBoolean.isMatchingOperand(t)
	}

	return false
}

// isArrayType used to test if a type is ArrayType
func (t StaticType) isArrayType() bool {
	if t == TypeIntArray || t == TypeFloatArray || t == TypeStringArray || t == TypeBooleanArray {
		return true
	}
	return false
}

// Status represents valid static values of typeStatus
type Status int

const (
	StatusError Status = iota
	StatusOk
	StatusUnset
)

func (s Status) String() string {
	switch s {
	case StatusError:
		return "error"
	case StatusOk:
		return "ok"
	case StatusUnset:
		return "unset"
	}

	return fmt.Sprintf("status(%d)", s)
}

type Kind int

const (
	KindUnspecified Kind = iota
	KindInternal
	KindClient
	KindServer
	KindProducer
	KindConsumer
)

func (k Kind) String() string {
	switch k {
	case KindUnspecified:
		return "unspecified"
	case KindInternal:
		return "internal"
	case KindClient:
		return "client"
	case KindServer:
		return "server"
	case KindProducer:
		return "producer"
	case KindConsumer:
		return "consumer"
	}

	return fmt.Sprintf("kind(%d)", k)
}

var (
	StaticTrue  = NewStaticBool(true)
	StaticFalse = NewStaticBool(false)
)
