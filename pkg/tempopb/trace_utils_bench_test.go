package tempopb_test

import (
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/grafana/tempo/v3/pkg/tempopb"
	"github.com/grafana/tempo/v3/pkg/util"
)

func BenchmarkMarshalToJSONV1(b *testing.B) {
	for _, size := range []struct {
		name  string
		bytes int
	}{{"3MB", 3_000_000}, {"30MB", 30_000_000}, {"120MB", 120_000_000}} {
		b.Run(size.name, func(b *testing.B) {
			trace, jsonBytes := benchmarkJSONTrace(b, size.bytes)
			b.ReportAllocs()
			b.SetBytes(int64(jsonBytes))
			for b.Loop() {
				encoded, err := tempopb.MarshalToJSONV1(trace)
				if err != nil {
					b.Fatal(err)
				}
				if len(encoded) != jsonBytes {
					b.Fatalf("JSON size = %d, expected %d", len(encoded), jsonBytes)
				}
			}
		})
	}
}

// Clone a fixed-seed TraceInfo trace to reach the target decimal MB size.
// IDs intentionally repeat: this benchmarks encoding, not deduplication.
// Two small encodes establish the exact size without encoding the full trace
// during setup, which would inflate process peak RSS measurements.
func benchmarkJSONTrace(tb testing.TB, targetBytes int) (*tempopb.Trace, int) {
	tb.Helper()
	seed, err := util.NewTraceInfo(time.Unix(0, 1632146180), "").ConstructTraceFromEpoch()
	if err != nil {
		tb.Fatal(err)
	}
	first, err := tempopb.MarshalToJSONV1(seed)
	if err != nil {
		tb.Fatal(err)
	}
	sample := proto.Clone(seed).(*tempopb.Trace)
	sample.ResourceSpans = append(sample.ResourceSpans, seed.ResourceSpans...)
	second, err := tempopb.MarshalToJSONV1(sample)
	if err != nil {
		tb.Fatal(err)
	}
	increment := len(second) - len(first)
	copies := 1
	if targetBytes > len(first) {
		copies += (targetBytes - len(first) + increment - 1) / increment
	}
	trace := &tempopb.Trace{}
	for range copies {
		clone := proto.Clone(seed).(*tempopb.Trace)
		trace.ResourceSpans = append(trace.ResourceSpans, clone.ResourceSpans...)
	}
	return trace, len(first) + (copies-1)*increment
}
