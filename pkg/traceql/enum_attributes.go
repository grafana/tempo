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
	intrinsicNone Intrinsic = iota
	intrinsicDuration
	intrinsicChildCount
	intrinsicName
	intrinsicStatus
	intrinsicParent
)

func (i Intrinsic) String() string {
	switch i {
	case intrinsicNone:
		return "none"
	case intrinsicDuration:
		return "duration"
	case intrinsicName:
		return "name"
	case intrinsicStatus:
		return "status"
	case intrinsicChildCount:
		return "childCount"
	case intrinsicParent:
		return "parent"
	}

	return fmt.Sprintf("intrinsic(%d)", i)
}

// intrinsicFromString returns the matching intrinsic for the given string or -1 if there is none
func intrinsicFromString(s string) Intrinsic {
	switch s {
	case "duration":
		return intrinsicDuration
	case "name":
		return intrinsicName
	case "status":
		return intrinsicStatus
	case "childCount":
		return intrinsicChildCount
	case "parent":
		return intrinsicParent
	}

	return intrinsicNone
}
