package vparquet

import (
	"context"
	"fmt"
	"testing"

	"github.com/grafana/tempo/pkg/traceql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBackendBlock_FetchSeries(t *testing.T) {
	wantTr := fullyPopulatedTestTrace(nil)
	b := makeBackendBlockWithTraces(t, []*Trace{wantTr})
	ctx := context.Background()

	res, err := b.FetchSeries(ctx, traceql.FetchSpansRequest{})
	require.NoError(t, err)
	assert.NotNil(t, res)
	assert.Greater(t, res.Bytes(), uint64(0))
	for {
		res, err := res.Results.Next(ctx)
		if err != nil || res == nil {
			break
		}
		// Print SpansetSeries
		fmt.Println("SpansetSeries")
		fmt.Printf("  %#v\n", *res)
	}
}
