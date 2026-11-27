package vparquet5

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	tempo_io "github.com/grafana/tempo/v3/pkg/io"
	"github.com/grafana/tempo/v3/pkg/tempopb"
	v1 "github.com/grafana/tempo/v3/pkg/tempopb/common/v1"
	v1_resource "github.com/grafana/tempo/v3/pkg/tempopb/resource/v1"
	v1_trace "github.com/grafana/tempo/v3/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/v3/pkg/tracefingerprint"
	"github.com/grafana/tempo/v3/pkg/util/test"
	"github.com/grafana/tempo/v3/tempodb/backend"
	"github.com/grafana/tempo/v3/tempodb/backend/local"
	"github.com/grafana/tempo/v3/tempodb/encoding/common"
)

// TestTraceFingerprintSurvivesRoundTrip checks that a trace has the same
// fingerprint after it is written to a vParquet5 file and read back, including
// resources with several service.name attributes or non-string ones.
func TestTraceFingerprintSurvivesRoundTrip(t *testing.T) {
	meta := backend.BlockMeta{DedicatedColumns: test.MakeDedicatedColumns()}
	traceID := []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F}

	str := func(s string) *v1.AnyValue { return &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: s}} }
	arr := func(s string) *v1.AnyValue {
		return &v1.AnyValue{Value: &v1.AnyValue_ArrayValue{ArrayValue: &v1.ArrayValue{Values: []*v1.AnyValue{str(s)}}}}
	}
	num := &v1.AnyValue{Value: &v1.AnyValue_IntValue{IntValue: 7}}
	resources := [][]*v1.KeyValue{
		{{Key: LabelServiceName, Value: str("a")}, {Key: LabelServiceName, Value: str("b")}},
		{{Key: LabelServiceName, Value: str("checkout")}, {Key: LabelServiceName, Value: num}},
		{{Key: LabelServiceName, Value: num}, {Key: LabelServiceName, Value: str("payment")}},
		{{Key: LabelServiceName, Value: arr("cart")}},
	}
	withResources := func(resources [][]*v1.KeyValue) *tempopb.Trace {
		tr := ParquetTraceToTempopbTrace(&meta, fullyPopulatedTestTrace(traceID))
		parent := tr.ResourceSpans[0].ScopeSpans[0].Spans[0].SpanId
		for i, attrs := range resources {
			tr.ResourceSpans = append(tr.ResourceSpans, &v1_trace.ResourceSpans{
				Resource: &v1_resource.Resource{Attributes: attrs},
				ScopeSpans: []*v1_trace.ScopeSpans{{Spans: []*v1_trace.Span{{
					TraceId:      traceID,
					SpanId:       []byte{0xf0, byte(i), 0, 0, 0, 0, 0, 1},
					ParentSpanId: parent,
					Name:         fmt.Sprintf("duplicate-service-%d", i),
				}}}},
			})
		}
		return tr
	}
	tr := withResources(resources)

	var f tracefingerprint.Fingerprinter
	want := f.Fingerprint(tr)
	require.Equal(t, tracefingerprint.StatusComputed, want.Status)

	// An array is not a service name, so ["cart"] must count as no service.
	noArray := slices.Clone(resources)
	noArray[3] = nil
	require.Equal(t, want, f.Fingerprint(withResources(noArray)))

	row, _ := traceToParquet(&meta, traceID, tr, nil)
	tsc := TraceSchemaConfig{DedicatedColumns: meta.DedicatedColumns}
	read := readTraceTestFile(t, tsc, writeTraceTestFile(t, tsc, []*Trace{row}), 1)
	require.Equal(t, want, f.Fingerprint(ParquetTraceToTempopbTrace(&meta, read[0])))
}

// TestTraceFingerprintDedicatedServiceNameColumn checks the exception in the
// Fingerprint docs. A dedicated string column named service.name reads a
// one-element array back as a plain string, so the stored trace fingerprints
// differently from the trace that was written.
func TestTraceFingerprintDedicatedServiceNameColumn(t *testing.T) {
	meta := backend.BlockMeta{DedicatedColumns: backend.DedicatedColumns{{
		Scope: backend.DedicatedColumnScopeResource,
		Name:  LabelServiceName,
		Type:  backend.DedicatedColumnTypeString,
	}}}
	traceID := []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F}
	withService := func(value *v1.AnyValue) *tempopb.Trace {
		var attrs []*v1.KeyValue
		if value != nil {
			attrs = []*v1.KeyValue{{Key: LabelServiceName, Value: value}}
		}
		return &tempopb.Trace{ResourceSpans: []*v1_trace.ResourceSpans{{
			Resource: &v1_resource.Resource{Attributes: attrs},
			ScopeSpans: []*v1_trace.ScopeSpans{{Spans: []*v1_trace.Span{{
				TraceId: traceID,
				SpanId:  []byte{1, 2, 3, 4, 5, 6, 7, 8},
				Name:    "GET",
			}}}},
		}}}
	}
	cart := &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "cart"}}
	array := &v1.AnyValue{Value: &v1.AnyValue_ArrayValue{ArrayValue: &v1.ArrayValue{Values: []*v1.AnyValue{cart}}}}
	tr := withService(array)

	var f tracefingerprint.Fingerprinter
	written := f.Fingerprint(tr)
	require.Equal(t, f.Fingerprint(withService(nil)), written)

	row, _ := traceToParquet(&meta, traceID, tr, nil)
	tsc := TraceSchemaConfig{DedicatedColumns: meta.DedicatedColumns}
	read := readTraceTestFile(t, tsc, writeTraceTestFile(t, tsc, []*Trace{row}), 1)
	stored := f.Fingerprint(ParquetTraceToTempopbTrace(&meta, read[0]))
	require.Equal(t, f.Fingerprint(withService(cart)), stored)
	require.NotEqual(t, written, stored)
}

// TestCreateBlockWritesFingerprints checks that CreateBlock, asked to compute
// fingerprints, fills the Similarity columns with the fingerprint of each
// trace as it is read back and flags the block nocompact. It covers a trace
// with no spans and a block that stores service.name in a dedicated column.
func TestCreateBlockWritesFingerprints(t *testing.T) {
	serviceNameColumn := backend.DedicatedColumn{Scope: backend.DedicatedColumnScopeResource, Name: LabelServiceName, Type: backend.DedicatedColumnTypeString}
	for _, tc := range []struct {
		name    string
		columns backend.DedicatedColumns
	}{
		{"generic service.name", test.MakeDedicatedColumns()},
		{"dedicated service.name", backend.DedicatedColumns{serviceNameColumn}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			array := &v1.AnyValue{Value: &v1.AnyValue_ArrayValue{ArrayValue: &v1.ArrayValue{Values: []*v1.AnyValue{{Value: &v1.AnyValue_StringValue{StringValue: "cart"}}}}}}
			arrayService := test.MakeTrace(2, nil)
			arrayService.ResourceSpans[0].Resource.Attributes = []*v1.KeyValue{{Key: LabelServiceName, Value: array}}
			traces := []*tempopb.Trace{test.MakeTrace(3, nil), {}, arrayService, test.MakeTrace(1, nil)}

			iter := newTestIterator()
			for _, tr := range traces {
				iter.Add(tr, 0, 0)
			}
			rawR, rawW, _, err := local.New(&local.Config{Path: t.TempDir()})
			require.NoError(t, err)
			meta := backend.NewBlockMeta("fake", uuid.New(), VersionString)
			meta.TotalObjects = int64(len(traces))
			meta.DedicatedColumns = tc.columns
			meta.Options = map[string]string{TraceSignatureVersionOption: TraceSignatureVersionV1}
			cfg := &common.BlockConfig{BloomFP: 0.01, BloomShardSizeBytes: 100 * 1024, RowGroupSizeBytes: 1 << 30, ComputeTraceFingerprints: true}

			outMeta, err := CreateBlock(context.Background(), cfg, meta, iter, backend.NewReader(rawR), backend.NewWriter(rawW))
			require.NoError(t, err)
			require.Equal(t, TraceSignatureVersionV1, outMeta.Options[TraceSignatureVersionOption])
			noCompact, err := backend.NewReader(rawR).HasNoCompactFlag(context.Background(), uuid.UUID(outMeta.BlockID), outMeta.TenantID)
			require.NoError(t, err)
			require.True(t, noCompact, "compaction would keep one fragment's fingerprint for a merged trace")

			data, err := backend.NewReader(rawR).Read(context.Background(), DataFileName, uuid.UUID(outMeta.BlockID), outMeta.TenantID, nil)
			require.NoError(t, err)
			tsc := TraceSchemaConfig{DedicatedColumns: outMeta.DedicatedColumns, Options: outMeta.Options}
			rows := readTraceTestFile(t, tsc, data, len(traces))

			var f tracefingerprint.Fingerprinter
			computed := 0
			for _, row := range rows {
				res := f.Fingerprint(ParquetTraceToTempopbTrace(outMeta, row))
				if res.Status != tracefingerprint.StatusComputed {
					require.Zero(t, row.Similarity.ShingleCount)
					require.Zero(t, row.Similarity.Signature)
					continue
				}
				computed++
				require.Equal(t, res.ShingleCount, row.Similarity.ShingleCount)
				require.Equal(t, res.Fingerprint, row.Similarity.Signature)
			}
			require.Equal(t, len(traces)-1, computed, "every trace with spans has a fingerprint")
		})
	}
}

// TestCreateBlockFingerprintRequest checks that only an explicit request
// computes fingerprints, and that a request CreateBlock cannot honor fails
// before anything is written.
func TestCreateBlockFingerprintRequest(t *testing.T) {
	v1Options := map[string]string{TraceSignatureVersionOption: TraceSignatureVersionV1}
	protoTraces := func() common.Iterator {
		iter := newTestIterator()
		iter.Add(test.MakeTrace(2, nil), 0, 0)
		return iter
	}
	for _, tc := range []struct {
		name    string
		options map[string]string
		compute bool
		iter    common.Iterator
		wantErr string
	}{
		{name: "v1 label alone stores no fingerprints", options: v1Options, iter: protoTraces()},
		{name: "request without the v1 label", compute: true, iter: protoTraces(), wantErr: "needs the block option minhash=v1"},
		{name: "request with stored rows", options: v1Options, compute: true, iter: &commonIterator{}, wantErr: "only be computed from protobuf traces"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rawR, rawW, _, err := local.New(&local.Config{Path: t.TempDir()})
			require.NoError(t, err)
			meta := backend.NewBlockMeta("fake", uuid.New(), VersionString)
			meta.TotalObjects = 1
			meta.Options = tc.options
			cfg := &common.BlockConfig{BloomFP: 0.01, BloomShardSizeBytes: 100 * 1024, RowGroupSizeBytes: 1 << 30, ComputeTraceFingerprints: tc.compute}

			outMeta, err := CreateBlock(context.Background(), cfg, meta, tc.iter, backend.NewReader(rawR), backend.NewWriter(rawW))
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				_, err := backend.NewReader(rawR).BlockMeta(context.Background(), uuid.UUID(meta.BlockID), meta.TenantID)
				require.ErrorIs(t, err, backend.ErrDoesNotExist)
				return
			}
			require.NoError(t, err)
			noCompact, err := backend.NewReader(rawR).HasNoCompactFlag(context.Background(), uuid.UUID(outMeta.BlockID), outMeta.TenantID)
			require.NoError(t, err)
			require.False(t, noCompact)

			data, err := backend.NewReader(rawR).Read(context.Background(), DataFileName, uuid.UUID(outMeta.BlockID), outMeta.TenantID, nil)
			require.NoError(t, err)
			rows := readTraceTestFile(t, TraceSchemaConfig{DedicatedColumns: outMeta.DedicatedColumns, Options: outMeta.Options}, data, 1)
			require.Equal(t, TraceSimilarity{}, rows[0].Similarity)
		})
	}
}

// TestFingerprintRowGroupEstimate checks that the row group size estimate
// counts computed fingerprints, which do not compress.
func TestFingerprintRowGroupEstimate(t *testing.T) {
	tr, _ := traceToParquet(&backend.BlockMeta{}, test.ValidTraceID(nil), test.MakeTrace(2, nil), nil)
	for _, compute := range []bool{false, true} {
		_, rawW, _, err := local.New(&local.Config{Path: t.TempDir()})
		require.NoError(t, err)
		meta := backend.NewBlockMeta("fake", uuid.New(), VersionString)
		meta.Options = map[string]string{TraceSignatureVersionOption: TraceSignatureVersionV1}
		cfg := &common.BlockConfig{BloomFP: 0.01, BloomShardSizeBytes: 100 * 1024, ComputeTraceFingerprints: compute}
		s, _ := newStreamingBlock(context.Background(), cfg, meta, nil, backend.NewWriter(rawW), tempo_io.NewBufferedWriter)
		require.NoError(t, s.Add(tr, 0, 0))

		want := estimateMarshalledSizeFromTrace(tr)
		if compute {
			want += tracefingerprint.Size + 4
		}
		require.Equal(t, want, s.EstimatedBufferedBytes(), "compute=%v", compute)
	}
}

// TestTraceSignatureVersionMatchesLibrary fails when the library changes its
// fingerprint version. CreateBlock labels the fingerprints it computes
// TraceSignatureVersionV1, so a new library version needs a new storage label
// rather than writing new fingerprints under the old one.
func TestTraceSignatureVersionMatchesLibrary(t *testing.T) {
	require.Equal(t, TraceSignatureVersionV1, tracefingerprint.Version)
}
