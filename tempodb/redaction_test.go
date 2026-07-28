package tempodb

import (
	"context"
	"testing"
	"time"

	"github.com/grafana/tempo/pkg/tempopb"
	v1_common "github.com/grafana/tempo/pkg/tempopb/common/v1"
	v1_trace "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/stretchr/testify/require"
)

// traceWithResourceAttr builds a single-batch trace carrying a specific resource
// attribute, so a TraceQL query can select it and leave others untouched.
func traceWithResourceAttr(id []byte, key, val string) *tempopb.Trace {
	attrs := []*v1_common.KeyValue{{
		Key:   key,
		Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_StringValue{StringValue: val}},
	}}
	return &tempopb.Trace{
		ResourceSpans: []*v1_trace.ResourceSpans{test.MakeBatchWithAttributes(2, id, attrs)},
	}
}

// TestRedactBlockQuerySelector verifies the TraceQL query path of RedactBlock: exactly the
// traces matching the query are dropped (no over-deletion), and dry-run counts without
// rewriting.
func TestRedactBlockQuerySelector(t *testing.T) {
	_, w, c, _ := testConfig(t, 0)
	ctx := context.Background()

	idMatch := test.ValidTraceID(nil)
	idKeep := test.ValidTraceID(nil)
	now := uint32(time.Now().Unix())

	// Two traces distinguished by resource.namespace; the query targets only one.
	data := []testData{
		{idMatch, traceWithResourceAttr(idMatch, "namespace", "checkout"), now, now},
		{idKeep, traceWithResourceAttr(idKeep, "namespace", "keep"), now, now},
	}
	query := `{resource.namespace = "checkout"}`

	t.Run("apply drops only the matching trace", func(t *testing.T) {
		blk := cutTestBlockWithTraces(t, w, data)
		meta := blk.BlockMeta()
		require.Equal(t, int64(2), meta.TotalObjects)

		rewrote, found, newMeta, err := c.RedactBlock(ctx, meta, testTenantID, nil, query, tempopb.RedactionMode_REDACTION_MODE_APPLY)
		require.NoError(t, err)
		require.True(t, rewrote)
		require.Equal(t, 1, found, "exactly one trace matches the query")
		require.NotNil(t, newMeta)
		require.Equal(t, int64(1), newMeta.TotalObjects, "the non-matching trace must survive")
	})

	t.Run("dry-run counts without rewriting", func(t *testing.T) {
		blk := cutTestBlockWithTraces(t, w, data)
		meta := blk.BlockMeta()

		rewrote, found, newMeta, err := c.RedactBlock(ctx, meta, testTenantID, nil, query, tempopb.RedactionMode_REDACTION_MODE_DRY_RUN)
		require.NoError(t, err)
		require.False(t, rewrote, "dry-run must not rewrite")
		require.Equal(t, 1, found, "dry-run still reports the match count")
		require.Nil(t, newMeta)
	})
}
