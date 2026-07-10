package bloomgateway

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewMetrics_NoDuplicateRegistrationPanic guards against copy-paste
// metric-name collisions across the ~28 series in metrics.go, before any
// other WP starts depending on these names. promauto.With(reg).NewX panics
// on a duplicate registration within the same registry, so a single
// successful call already proves every Name+ConstLabels pair in newMetrics
// is unique.
func TestNewMetrics_NoDuplicateRegistrationPanic(t *testing.T) {
	require.NotPanics(t, func() {
		newMetrics(prometheus.NewRegistry())
	})
}

// TestNewMetrics_MultipleInstancesDoNotCollide is the reason newMetrics
// takes an explicit Registerer instead of using package-level promauto
// vars (see metrics.go's doc comment): multiple *BloomGateway instances
// can exist in one test process (every WP6/WP20 multi-instance ring test
// does this), each with its own registry, and construction must not panic
// for any of them.
func TestNewMetrics_MultipleInstancesDoNotCollide(t *testing.T) {
	require.NotPanics(t, func() {
		_ = newMetrics(prometheus.NewRegistry())
		_ = newMetrics(prometheus.NewRegistry())
		_ = newMetrics(prometheus.NewRegistry())
	})
}

// TestNewMetrics_RegisteredUnderTempoBloomGatewayPrefix spot-checks a
// scalar gauge and a scalar counter to lock in DESIGN.md's stated
// "tempo_bloom_gateway_*" prefix (§ Metrics). Vec metrics (e.g.
// ownedLeaves) are deliberately not checked here: a *Vec with no observed
// label combination yet emits zero samples on Gather, which would make a
// count-based assertion meaningless rather than confirm the prefix.
func TestNewMetrics_RegisteredUnderTempoBloomGatewayPrefix(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := newMetrics(reg)

	count, err := testutil.GatherAndCount(reg, "tempo_bloom_gateway_blocks_live")
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	count, err = testutil.GatherAndCount(reg, "tempo_bloom_gateway_add_chunks_total")
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	// Belt-and-suspenders: the fields are usable (not nil, correctly
	// typed) — a compile error here would already catch a nil field, but
	// actually incrementing/observing catches an accidental mismatch
	// between a field's declared type and what was registered above.
	m.blocksLive.Set(1)
	m.addChunksTotal.Inc()
	m.ownedLeaves.WithLabelValues("complete").Set(1)
	m.queriesTotal.WithLabelValues("reject_all").Inc()
	m.queryDurationSeconds.Observe(0.001)
}
