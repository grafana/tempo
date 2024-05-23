package traceql

import "fmt"

type AttributeScope int

const (
	AttributeScopeNone AttributeScope = iota
	AttributeScopeResource
	AttributeScopeSpan
	AttributeScopeUnknown

	none     = "none"
	duration = "duration"
)

func AllAttributeScopes() []AttributeScope {
	return []AttributeScope{AttributeScopeResource, AttributeScopeSpan}
}

func (s AttributeScope) String() string {
	switch s {
	case AttributeScopeNone:
		return none
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
	case none:
		return AttributeScopeNone
	}

	return AttributeScopeUnknown
}

type Intrinsic int

const (
	IntrinsicNone Intrinsic = iota
	IntrinsicDuration
	IntrinsicName
	IntrinsicStatus
	IntrinsicStatusMessage
	IntrinsicKind
	IntrinsicChildCount
	IntrinsicTraceRootService
	IntrinsicTraceRootSpan
	IntrinsicTraceDuration
	IntrinsicNestedSetLeft
	IntrinsicNestedSetRight
	IntrinsicNestedSetParent

	// not yet implemented in traceql but will be
	IntrinsicParent

	// These intrinsics do not map to specific data points, but are used to
	// indicate that Spans must be able to answer the structural methods
	// DescdendantOf, SiblingOf and ChildOf.  The details of those methods
	// and how these intrinsics are handled is left to the implementation.
	IntrinsicStructuralDescendant
	IntrinsicStructuralSibling
	IntrinsicStructuralChild

	IntrinsicTraceID
	IntrinsicSpanID

	// not yet implemented in traceql and may never be. these exist so that we can retrieve
	// these fields from the fetch layer

	IntrinsicTraceStartTime
	IntrinsicSpanStartTime

	IntrinsicServiceStats
)

var (
	IntrinsicDurationAttribute         = NewIntrinsic(IntrinsicDuration)
	IntrinsicNameAttribute             = NewIntrinsic(IntrinsicName)
	IntrinsicStatusAttribute           = NewIntrinsic(IntrinsicStatus)
	IntrinsicStatusMessageAttribute    = NewIntrinsic(IntrinsicStatusMessage)
	IntrinsicKindAttribute             = NewIntrinsic(IntrinsicKind)
	IntrinsicSpanIDAttribute           = NewIntrinsic(IntrinsicSpanID)
	IntrinsicChildCountAttribute       = NewIntrinsic(IntrinsicChildCount)
	IntrinsicTraceIDAttribute          = NewIntrinsic(IntrinsicTraceID)
	IntrinsicTraceRootServiceAttribute = NewIntrinsic(IntrinsicTraceRootService)
	IntrinsicTraceRootSpanAttribute    = NewIntrinsic(IntrinsicTraceRootSpan)
	IntrinsicTraceDurationAttribute    = NewIntrinsic(IntrinsicTraceDuration)
	IntrinsicSpanStartTimeAttribute    = NewIntrinsic(IntrinsicSpanStartTime)
	IntrinsicNestedSetLeftAttribute    = NewIntrinsic(IntrinsicNestedSetLeft)
	IntrinsicNestedSetRightAttribute   = NewIntrinsic(IntrinsicNestedSetRight)
	IntrinsicNestedSetParentAttribute  = NewIntrinsic(IntrinsicNestedSetParent)
)

func (i Intrinsic) String() string {
	switch i {
	case IntrinsicNone:
		return none
	case IntrinsicDuration:
		return duration
	case IntrinsicName:
		return "name"
	case IntrinsicStatus:
		return "status"
	case IntrinsicStatusMessage:
		return "statusMessage"
	case IntrinsicKind:
		return "kind"
	case IntrinsicChildCount:
		return "childCount"
	case IntrinsicParent:
		return "parent"
	case IntrinsicTraceRootService:
		return "rootServiceName"
	case IntrinsicTraceRootSpan:
		return "rootName"
	case IntrinsicTraceDuration:
		return "traceDuration"
	case IntrinsicTraceID:
		return "trace:id"
	case IntrinsicTraceStartTime:
		return "traceStartTime"
	case IntrinsicSpanID:
		return "span:id"
	// below is unimplemented
	case IntrinsicSpanStartTime:
		return "spanStartTime"
	case IntrinsicNestedSetLeft:
		return "nestedSetLeft"
	case IntrinsicNestedSetRight:
		return "nestedSetRight"
	case IntrinsicNestedSetParent:
		return "nestedSetParent"
	}

	return fmt.Sprintf("intrinsic(%d)", i)
}

// intrinsicFromString returns the matching intrinsic for the given string or -1 if there is none
func intrinsicFromString(s string) Intrinsic {
	switch s {
	case duration:
		return IntrinsicDuration
	case "name":
		return IntrinsicName
	case "status":
		return IntrinsicStatus
	case "statusMessage":
		return IntrinsicStatusMessage
	case "kind":
		return IntrinsicKind
	case "childCount":
		return IntrinsicChildCount
	case "parent":
		return IntrinsicParent
	case "rootServiceName":
		return IntrinsicTraceRootService
	case "rootName":
		return IntrinsicTraceRootSpan
	case "traceDuration":
		return IntrinsicTraceDuration
	case "trace:id":
		return IntrinsicTraceID
	case "traceStartTime":
		return IntrinsicTraceStartTime
	case "span:id":
		return IntrinsicSpanID
	// unimplemented
	case "spanStartTime":
		return IntrinsicSpanStartTime
	case "nestedSetLeft":
		return IntrinsicNestedSetLeft
	case "nestedSetRight":
		return IntrinsicNestedSetRight
	case "nestedSetParent":
		return IntrinsicNestedSetParent
	}

	return IntrinsicNone
}
