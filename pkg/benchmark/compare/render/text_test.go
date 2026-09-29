package render

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/pkg/benchmark"
	"github.com/grafana/tempo/v3/pkg/benchmark/compare"
	"github.com/grafana/tempo/v3/pkg/benchmark/metrics"
)

func TestWriteReport(t *testing.T) {
	result := func(size int, p50 float64) *benchmark.Result {
		r := newResult("mac", caseResult("search/nopredicate", 5, metrics.Set{"harness.wallNs": measurement(p50)}))
		r.Cases[0].Query = "{}"
		r.Options.ReadBufferSize = size
		return r
	}
	c, err := compare.New([]compare.Run{
		{File: "base.json", Result: result(0, 10e6)},
		{File: "big.json", Result: result(4<<20, 8.7e6)},
	})
	require.NoError(t, err)

	var b strings.Builder
	require.NoError(t, WriteReport(&b, c, []string{"harness.wallNs"}, compare.P50, c.Baseline, 80))
	report := b.String()

	require.Contains(t, report, "runs named by readBufferSize, in its order; baseline default\n")
	require.Contains(t, report, "  default  baseline · readBufferSize default · gitSHA abc")
	require.Contains(t, report, "  4MiB     readBufferSize default → 4MiB\n")
	require.Contains(t, report, "harness.wallNs · p50 per execution · change from default\n"+
		"  case               │ default │     4MiB\n"+
		" search\n"+
		"  search/nopredicate │    10ms │ 8.7ms  -13.0%\n")
	require.Contains(t, report, "search/nopredicate · harness.wallNs · per execution\nquery: {}\n")
	require.Contains(t, report, "4MiB     10  8.7ms")
	require.Contains(t, report, "-13.0%")
}
