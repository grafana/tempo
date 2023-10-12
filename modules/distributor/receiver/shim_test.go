package receiver

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestWrapRetryableError confirms that errors are wrapped as expected
func TestWrapRetryableError(t *testing.T) {
	// no wrapping b/c not a grpc error
	err := errors.New("test error")
	wrapped := wrapErrorIfRetryable(err)
	require.Equal(t, err, wrapped)
	require.False(t, isRetryable(wrapped))

	// no wrapping b/c not a resource exhausted grpc error
	err = status.Error(codes.FailedPrecondition, "failed precondition")
	wrapped = wrapErrorIfRetryable(err)
	require.Equal(t, err, wrapped)
	require.False(t, isRetryable(wrapped))

	// wrapping b/c this is a resource exhausted grpc error
	err = status.Error(codes.ResourceExhausted, "res exhausted")
	wrapped = wrapErrorIfRetryable(err)
	require.NotEqual(t, err, wrapped)
	require.True(t, isRetryable(wrapped))
}

func isRetryable(err error) bool {
	st, ok := status.FromError(err)

	if !ok {
		return false
	}

	for _, detail := range st.Details() {
		if _, ok := detail.(*errdetails.RetryInfo); ok {
			return true
		}
	}
	return false
}
