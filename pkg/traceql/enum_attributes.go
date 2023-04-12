package traceql

import "fmt"

type AttributeScope int

const (
	AttributeScopeNone AttributeScope = iota
	AttributeScopeResource
	AttributeScopeSpan
	AttributeScopeUnknown
)

func AllAttributeScopes() []AttributeScope {
	return []AttributeScope{AttributeScopeResource, AttributeScopeSpan}
}

func (s AttributeScope) String() string {
	switch s {
	case AttributeScopeNone:
		return "none"
	case AttributeScopeSpan:
		return "span"
	case AttributeScopeResource:
		return "resource"
	}

	return fmt.Sprintf("att(%d).", s)
}

func AttributeScopeFromString(s string) AttributeScope {
	switch s {
	case "span":
		return AttributeScopeSpan
	case "resource":
		return AttributeScopeResource
	case "":
		fallthrough
	case "none":
		return AttributeScopeNone
	}

	return AttributeScopeUnknown
}

type Intrinsic int

const (
	IntrinsicNone Intrinsic = iota
	IntrinsicDuration
	IntrinsicChildCount
	IntrinsicName
	IntrinsicStatus
	IntrinsicKind
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
	case IntrinsicKind:
		return "kind"
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
	case "kind":
		return IntrinsicKind
	case "childCount":
		return IntrinsicChildCount
	case "parent":
		return IntrinsicParent
	}

	return IntrinsicNone
}
