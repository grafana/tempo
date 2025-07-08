package bufferer

import (
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/stretchr/testify/assert"
)

func TestPushBytes(t *testing.T) {
	// This is a basic test to ensure the pushBytes method doesn't panic
	// More comprehensive tests would require setting up WAL and other dependencies

	b := &Buffer{
		logger: log.NewNopLogger(),
	}

	// Create a simple PushBytesRequest
	req := &tempopb.PushBytesRequest{
		Traces: []tempopb.PreallocBytes{},
		Ids:    [][]byte{},
	}

	// This should not panic
	assert.NotPanics(t, func() {
		b.pushBytes(time.Now(), req, "test-tenant")
	})
}
