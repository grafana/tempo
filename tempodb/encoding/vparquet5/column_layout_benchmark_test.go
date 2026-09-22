package vparquet5

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/pkg/tempopb"
	"github.com/grafana/tempo/v3/pkg/util/test"
	"github.com/grafana/tempo/v3/tempodb/encoding/common"
)

func BenchmarkSearchColumnLayouts(b *testing.B) {
	for _, options := range []bool{false, true} {
		name := "plain"
		if options {
			name = "options"
		}
		b.Run(name, func(b *testing.B) {
			traces, _, _, _ := makeTraces()
			columns := test.MakeDedicatedColumns()
			if !options {
				for i := range columns {
					columns[i].Options = nil
				}
			}
			block := makeBackendBlockWithTracesWithDedicatedColumns(b, traces, columns)
			req := &tempopb.SearchRequest{Tags: map[string]string{"dedicated.span.1": "dedicated-span-attr-value-1"}, Limit: 20}
			opts := common.DefaultSearchOptions()
			resp, err := block.Search(context.Background(), req, opts)
			require.NoError(b, err)
			require.NotEmpty(b, resp.Traces)
			b.ReportAllocs()
			for b.Loop() {
				resp, err := block.Search(context.Background(), req, opts)
				if err != nil || len(resp.Traces) == 0 {
					b.Fatalf("search: %v", err)
				}
			}
		})
	}
}
