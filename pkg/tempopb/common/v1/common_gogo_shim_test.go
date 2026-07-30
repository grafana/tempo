package v1

import (
	"strings"
	"testing"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Regression test for the bug that motivated dropping the old StableString
// shim: StableString's ArrayValue branch did fmt.Sprintf("%v", x.ArrayValue.Values)
// on a []AnyValue value-slice. Each element's Value field is an interface
// wrapping a pointer (e.g. *AnyValue_StringValue), and fmt only auto-dereferences
// a pointer with "&{...}" at the top level of a Print call — a pointer found
// while already inside a slice/struct falls through to printing its raw
// address instead, so StableString's output changed from run to run.
// The generated String() below recurses via e.String() on addressable
// elements instead of "%v" on the slice, so it never hits that path.
func TestAnyValue_String_ArrayValueIsDeterministic(t *testing.T) {
	newArray := func() *AnyValue {
		return &AnyValue{
			Value: &AnyValue_ArrayValue{
				ArrayValue: ArrayValue{
					Values: []AnyValue{
						{Value: &AnyValue_StringValue{StringValue: "a"}},
						{Value: &AnyValue_IntValue{IntValue: 1}},
					},
				},
			},
		}
	}

	// Two independently-constructed-but-equal instances must render
	// identically. Under the old bug this would fail intermittently since
	// each construction gets a fresh *AnyValue_StringValue allocation.
	a, b := newArray(), newArray()
	got := a.String()
	assert.Equal(t, got, b.String())
	assert.NotContains(t, got, "0x", "String() must not leak pointer addresses")
	assert.Contains(t, got, `string_value: "a"`)
	assert.Contains(t, got, "int_value: 1")
}

func TestAnyValue_String_StringValueStrindex(t *testing.T) {
	// StableString had no case for this variant at all (it fell through to
	// "", which meant every strindex-backed value collided under sort/dedup
	// keying). String() covers all 8 oneof variants including this one.
	v := &AnyValue{Value: &AnyValue_StringValueStrindex{StringValueStrindex: 42}}
	assert.Equal(t, "string_value_strindex: 42", v.String())
}

// Regression test for the other Blocker fixed alongside StableString: gogo's
// jsonpb builds its oneof field-dispatch table from XXX_OneofWrappers(),
// which omitted (*AnyValue_StringValueStrindex)(nil). Without that entry,
// jsonpb has no wrapper type to route the "stringValueStrindex" JSON field
// to, and Unmarshal errors instead of populating the oneof.
func TestAnyValue_XXX_OneofWrappers_StringValueStrindexRoundTrip(t *testing.T) {
	var v AnyValue
	err := jsonpb.Unmarshal(strings.NewReader(`{"stringValueStrindex": 42}`), &v)
	require.NoError(t, err)
	assert.Equal(t, &AnyValue_StringValueStrindex{StringValueStrindex: 42}, v.Value)
}
