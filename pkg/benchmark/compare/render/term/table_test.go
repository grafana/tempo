package term

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/pkg/benchmark/compare"
	"github.com/grafana/tempo/v3/pkg/benchmark/compare/internal/comparetest"
	"github.com/grafana/tempo/v3/pkg/benchmark/compare/render"
	"github.com/grafana/tempo/v3/pkg/benchmark/metrics"
)

func TestTable(t *testing.T) {
	base := comparetest.Summary(15e6, 26e6, 32.3e6, 37e6, 40.3e6, 47.8e6, 1.12e9)
	faster := comparetest.Summary(14e6, 23e6, 28.1e6, 33e6, 36.9e6, 44e6, 0.98e9)
	s := compare.Series{Unit: compare.Nanoseconds, Summaries: []*metrics.Summary{&base, &faster, nil}}
	names := []string{"base", "rb-4M", "gone"}

	header, rows := Table(render.NewTable(names, s, 0), 80)
	require.Equal(t, "run     n     p50     p90     p99    max    Δp50   Δp99", header.String())
	require.Equal(t, render.Dim, header[0].Style)
	require.Equal(t, []string{
		"base   10  32.3ms  40.3ms  47.8ms  1.12s",
		"rb-4M  10  28.1ms  36.9ms    44ms  980ms  -13.0%  -7.9%",
		"gone    –       –       –       –      –",
	}, texts(rows))
	require.Equal(t, render.BaselineName, rows[0][0].Style)
	require.Equal(t, render.RunName, rows[1][0].Style)

	// Rows are cut to the width.
	_, rows = Table(render.NewTable(names, s, 0), 20)
	require.Equal(t, "rb-4M  10  28.1ms  …", rows[1].String())
}

func texts(lines []Line) []string {
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = l.String()
	}
	return out
}
