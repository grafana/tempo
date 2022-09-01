package traceql

import "fmt"

type AttributeScope int

const (
	attributeScopeNone AttributeScope = iota
	attributeScopeResource
	attributeScopeSpan
)

func (s AttributeScope) String() string {
	switch s {
	case attributeScopeNone:
		return "none"
	case attributeScopeSpan:
		return "span"
	case attributeScopeResource:
		return "resource"
	}

	return fmt.Sprintf("att(%d).", s)
}

type Intrinsic int

const (
	IntrinsicNone Intrinsic = iota
	IntrinsicDuration
	IntrinsicChildCount
	IntrinsicName
	IntrinsicStatus
	IntrinsicParent
)

func (i Intrinsic) String() string {
	switch i {
	case IntrinsicNone:
		return "none"
	case IntrinsicDuration:
		return "duration"
	case IntrinsicName:
		return "name"
	case IntrinsicStatus:
		return "status"
	case IntrinsicChildCount:
		return "childCount"
	case IntrinsicParent:
		return "parent"
	}

	return fmt.Sprintf("intrinsic(%d)", i)
}

// intrinsicFromString returns the matching intrinsic for the given string or -1 if there is none
func intrinsicFromString(s string) Intrinsic {
	switch s {
	case "duration":
		return IntrinsicDuration
	case "name":
		return IntrinsicName
	case "status":
		return IntrinsicStatus
	case "childCount":
		return IntrinsicChildCount
	case "parent":
		return IntrinsicParent
	}

	return IntrinsicNone
}
