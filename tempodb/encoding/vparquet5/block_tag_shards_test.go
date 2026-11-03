package vparquet5

import (
	"context"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	v1_common "github.com/grafana/tempo/v3/pkg/tempopb/common/v1"
	"github.com/grafana/tempo/v3/pkg/traceql"
	"github.com/grafana/tempo/v3/pkg/util/test"
	"github.com/grafana/tempo/v3/tempodb/backend"
	"github.com/grafana/tempo/v3/tempodb/encoding/common"
)

// countingReader counts the bytes a block reads, independently of what the
// block reports about itself.
type countingReader struct {
	backend.Reader
	bytes atomic.Uint64
}

func (r *countingReader) ReadRange(ctx context.Context, name string, blockID uuid.UUID, tenantID string, offset uint64, buffer []byte, cacheInfo *backend.CacheInfo) error {
	r.bytes.Add(uint64(len(buffer)))
	return r.Reader.ReadRange(ctx, name, blockID, tenantID, offset, buffer, cacheInfo)
}

// tagShardBlock builds a block of several row groups in which each group of
// traces carries its own tag name and tag value, so a shard can only find the
// ones in its own row groups.
func tagShardBlock(t *testing.T) (*backendBlock, *countingReader, int) {
	traces := make([]*Trace, 0, 400)
	for i := range 400 {
		id := test.ValidTraceID(nil)
		pb := test.MakeTrace(1, id)
		group := strconv.Itoa(i / 50)
		span := pb.ResourceSpans[0].ScopeSpans[0].Spans[0]
		span.Attributes = append(span.Attributes,
			&v1_common.KeyValue{Key: "group-" + group, Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_StringValue{StringValue: "x"}}},
			&v1_common.KeyValue{Key: "group", Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_StringValue{StringValue: group}}},
		)
		tr, _ := traceToParquet(&backend.BlockMeta{}, id, pb, nil)
		traces = append(traces, tr)
	}

	b := makeBackendBlockWithTraces(t, traces)
	reader := &countingReader{Reader: b.r}
	b = newBackendBlock(b.meta, reader)

	pf, _, err := b.openForSearch(context.Background(), common.DefaultSearchOptions())
	require.NoError(t, err)
	rowGroups := len(pf.RowGroups())
	require.Greater(t, rowGroups, 1, "the block needs several row groups to shard")

	return b, reader, rowGroups
}

func TestTagSearchHonoursShard(t *testing.T) {
	ctx := context.Background()
	b, _, rowGroups := tagShardBlock(t)

	tag := traceql.MustParseIdentifier("span.group")
	scoped, err := traceql.ExtractConditionGroups(`{ span.group != "" }`, 10)
	require.NoError(t, err)
	unscoped, err := traceql.ExtractConditionGroups(`{ .group != "" }`, 10)
	require.NoError(t, err)

	groupNames := func(collect func(func(string, traceql.AttributeScope)) error) (out []string, err error) {
		err = collect(func(name string, _ traceql.AttributeScope) {
			if strings.HasPrefix(name, "group-") {
				out = append(out, name)
			}
		})
		return out, err
	}

	for name, search := range map[string]func(common.SearchOptions) ([]string, error){
		"SearchTags": func(opts common.SearchOptions) ([]string, error) {
			return groupNames(func(cb func(string, traceql.AttributeScope)) error {
				return b.SearchTags(ctx, traceql.AttributeScopeSpan, cb, func(uint64) {}, opts)
			})
		},
		"FetchTagNames": func(opts common.SearchOptions) ([]string, error) {
			return groupNames(func(cb func(string, traceql.AttributeScope)) error {
				req := traceql.FetchTagsRequest{ConditionGroups: scoped, Scope: traceql.AttributeScopeSpan}
				return b.FetchTagNames(ctx, req, func(n string, s traceql.AttributeScope) bool { cb(n, s); return false }, func(uint64) {}, opts)
			})
		},
		"SearchTagValuesV2": func(opts common.SearchOptions) (out []string, err error) {
			err = b.SearchTagValuesV2(ctx, tag, func(v traceql.Static) bool { out = append(out, v.EncodeToString(false)); return false }, func(uint64) {}, opts)
			return out, err
		},
		"FetchTagValues unscoped": func(opts common.SearchOptions) (out []string, err error) {
			req := traceql.FetchTagValuesRequest{ConditionGroups: unscoped, TagName: tag}
			err = b.FetchTagValues(ctx, req, func(v traceql.Static) bool { out = append(out, v.EncodeToString(false)); return false }, func(uint64) {}, opts)
			return out, err
		},
	} {
		t.Run(name, func(t *testing.T) {
			whole, err := search(common.DefaultSearchOptions())
			require.NoError(t, err)

			union := map[string]struct{}{}
			for i := range rowGroups {
				opts := common.DefaultSearchOptions()
				opts.StartPage, opts.TotalPages = i, 1
				got, err := search(opts)
				require.NoError(t, err)
				if i == 0 {
					require.NotEmpty(t, got)
					require.Less(t, len(dedupe(got)), len(dedupe(whole)), "a shard must only see its own row groups")
				}
				for _, v := range got {
					union[v] = struct{}{}
				}
			}
			require.Equal(t, dedupe(whole), union, "the shards together must find everything")
		})
	}
}

func TestTagFetchReportsBytesRead(t *testing.T) {
	ctx := context.Background()
	b, reader, _ := tagShardBlock(t)

	tag := traceql.MustParseIdentifier("span.group")
	// Two scoped conditions, so neither call falls back to the no-condition path,
	// which reports its bytes correctly.
	conds, err := traceql.ExtractConditionGroups(`{ span.group != "" && resource.service.name != "" }`, 10)
	require.NoError(t, err)

	for name, fetch := range map[string]func(common.MetricsCallback) error{
		"FetchTagNames": func(mcb common.MetricsCallback) error {
			req := traceql.FetchTagsRequest{ConditionGroups: conds, Scope: traceql.AttributeScopeSpan}
			return b.FetchTagNames(ctx, req, func(string, traceql.AttributeScope) bool { return false }, mcb, common.DefaultSearchOptions())
		},
		"FetchTagValues": func(mcb common.MetricsCallback) error {
			req := traceql.FetchTagValuesRequest{ConditionGroups: conds, TagName: tag}
			return b.FetchTagValues(ctx, req, func(traceql.Static) bool { return false }, mcb, common.DefaultSearchOptions())
		},
	} {
		t.Run(name, func(t *testing.T) {
			before := reader.bytes.Load()
			var reported uint64
			require.NoError(t, fetch(func(n uint64) { reported += n }))
			require.Equal(t, reader.bytes.Load()-before, reported, "reported bytes must match what was read")
		})
	}
}

func dedupe(values []string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, v := range values {
		out[v] = struct{}{}
	}
	return out
}
