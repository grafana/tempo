package frontend

import (
	"testing"
	"time"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/stretchr/testify/require"
)

func TestClampInstantQueryEnd(t *testing.T) {
	// A "last 6 hours" instant query. The end carries a sub-second remainder to
	// mimic a real client timestamp; start is exactly 6h earlier and shares that
	// same remainder.
	end := time.Now().Truncate(time.Second).Add(123456789 * time.Nanosecond)
	start := end.Add(-6 * time.Hour)

	req := &tempopb.QueryRangeRequest{
		Start: uint64(start.UnixNano()),
		End:   uint64(end.UnixNano()),
	}

	cfg := Config{QueryEndCutoff: 30 * time.Second}
	require.NoError(t, clampInstantQueryEnd(cfg, req))

	// The clamped range must be a whole number of seconds - the sub-second jitter
	// is gone - which is what keeps the query-range cache key stable across refreshes.
	rng := req.End - req.Start
	require.Zero(t, rng%uint64(time.Second), "range %v is not a whole number of seconds", time.Duration(rng))
}
