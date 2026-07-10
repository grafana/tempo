package bloomgateway

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestBlockState_ZeroValueIsPending guards against an uninitialized Block{}
// (WP11) accidentally reading as live: the registry relies on BlockPending
// being the zero value so a freshly zero-valued struct is never mistaken
// for a live, rejectable block.
func TestBlockState_ZeroValueIsPending(t *testing.T) {
	var s BlockState
	assert.Equal(t, BlockPending, s)
}

// TestInvalidHandle_IsZeroValue asserts InvalidHandle is exactly the zero
// value of Handle, so a zero-initialized Handle field (e.g. in a struct
// that hasn't been assigned a real handle yet) reads as invalid by
// construction rather than by convention alone. WP11's own registry tests
// cross-check that its real allocator never hands out InvalidHandle.
func TestInvalidHandle_IsZeroValue(t *testing.T) {
	var h Handle
	assert.Equal(t, InvalidHandle, h)
	assert.EqualValues(t, 0, InvalidHandle)
}

func TestBlockState_String(t *testing.T) {
	tests := []struct {
		state BlockState
		want  string
	}{
		{BlockPending, "pending"},
		{BlockLive, "live"},
		{BlockLiveUnsupportedEncoding, "live_unsupported_encoding"},
		{BlockDeleted, "deleted"},
		{BlockState(99), "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.state.String())
		})
	}
}
