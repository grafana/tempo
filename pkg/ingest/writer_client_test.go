package ingest

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kgo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestStatusFromProduceErr(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want codes.Code
	}{
		{"nil", nil, codes.OK},
		{"record timeout", kgo.ErrRecordTimeout, codes.Unavailable},
		{"context deadline", context.DeadlineExceeded, codes.Unavailable},
		{"buffer full", kgo.ErrMaxBuffered, codes.Unavailable},
		{"generic", errors.New("boom"), codes.Unavailable},
		{"message too large", kerr.MessageTooLarge, codes.InvalidArgument},
		{"wrapped message too large", fmt.Errorf("failed to produce: %w", kerr.MessageTooLarge), codes.InvalidArgument},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StatusFromProduceErr(tt.err)
			require.Equal(t, tt.want, status.Code(got))
			if tt.err == nil {
				require.NoError(t, got)
				return
			}
			require.Contains(t, got.Error(), tt.err.Error())
		})
	}
}
