package bloomgatewayevents

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewMetrics_NoDuplicateRegistrationPanic guards against copy-paste
// metric-name collisions, mirroring modules/bloomgateway/metrics_test.go's
// same-named test: promauto.With(reg).NewX panics on a duplicate
// registration within the same registry, so one successful call already
// proves every Name+ConstLabels pair in newMetrics is unique.
func TestNewMetrics_NoDuplicateRegistrationPanic(t *testing.T) {
	require.NotPanics(t, func() {
		newMetrics(prometheus.NewRegistry())
	})
}

// TestNewMetrics_MultipleInstancesDoNotCollide: every producer
// (block-builder, backend-worker) constructs its own metrics against its
// own Registerer in the same process; package-level promauto vars would
// panic on the second instance's construction.
func TestNewMetrics_MultipleInstancesDoNotCollide(t *testing.T) {
	require.NotPanics(t, func() {
		_ = newMetrics(prometheus.NewRegistry())
		_ = newMetrics(prometheus.NewRegistry())
	})
}

// TestNewMetrics_RegisteredUnderTempoBloomGatewayPrefix locks in DESIGN.md's
// stated producer-side names (§ Metrics: "bloom_gateway_publishes_total",
// "bloom_gateway_publish_duration_seconds") under the tempo_ namespace.
func TestNewMetrics_RegisteredUnderTempoBloomGatewayPrefix(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := newMetrics(reg)

	// Belt-and-suspenders: also proves the fields are usable (not nil,
	// correctly typed) -- a compile error would already catch a nil field,
	// but incrementing/observing catches a mismatch between a field's
	// declared type and what was registered above.
	m.publishesTotal.WithLabelValues("ok").Inc()
	m.publishDurationSeconds.Observe(0.001)

	count, err := testutil.GatherAndCount(reg, "tempo_bloom_gateway_publishes_total")
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	count, err = testutil.GatherAndCount(reg, "tempo_bloom_gateway_publish_duration_seconds")
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}
