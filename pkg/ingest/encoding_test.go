package ingest

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util/test"
)

func TestEncoderDecoder(t *testing.T) {
	tests := []struct {
		name        string
		req         *tempopb.PushBytesRequest
		maxSize     int
		expectSplit bool
	}{
		{
			name:        "Small trace, no split",
			req:         generateRequest(10, 100),
			maxSize:     1024 * 1024,
			expectSplit: false,
		},
		{
			name:        "Large trace, expect split",
			req:         generateRequest(1000, 1000),
			maxSize:     1024 * 10,
			expectSplit: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoder := NewDecoder()

			records, err := Encode(0, "test-tenant", tt.req, tt.maxSize)
			require.NoError(t, err)

			if tt.expectSplit {
				require.Greater(t, len(records), 1)
			} else {
				require.Equal(t, 1, len(records))
			}

			var decodedEntries []tempopb.PreallocBytes
			var decodedIDs [][]byte

			for _, record := range records {
				decoder.Reset()
				req, err := decoder.Decode(record.Value)
				require.NoError(t, err)
				decodedEntries = append(decodedEntries, req.Traces...)
				decodedIDs = append(decodedIDs, req.Ids...)
			}

			require.Equal(t, len(tt.req.Traces), len(decodedEntries))
			for i := range tt.req.Traces {
				require.Equal(t, tt.req.Traces[i], decodedEntries[i])
				require.Equal(t, tt.req.Ids[i], decodedIDs[i])
			}
		})
	}
}

func TestEncoderSingleEntryTooLarge(t *testing.T) {
	stream := generateRequest(1, 1000)

	_, err := Encode(0, "test-tenant", stream, 100)
	require.Error(t, err)
	require.Contains(t, err.Error(), "single entry size")
}

func TestDecoderInvalidData(t *testing.T) {
	decoder := NewDecoder()

	_, err := decoder.Decode([]byte("invalid data"))
	require.Error(t, err)
}

func TestEncoderDecoderEmptyStream(t *testing.T) {
	decoder := NewDecoder()

	req := &tempopb.PushBytesRequest{}

	records, err := Encode(0, "test-tenant", req, 10<<20)
	require.NoError(t, err)
	require.Len(t, records, 1)

	decodedReq, err := decoder.Decode(records[0].Value)
	require.NoError(t, err)
	require.Equal(t, req.Traces, decodedReq.Traces)
}

func BenchmarkEncodeDecode(b *testing.B) {
	decoder := NewDecoder()
	stream := generateRequest(1000, 200)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		records, err := Encode(0, "test-tenant", stream, 10<<20)
		if err != nil {
			b.Fatal(err)
		}
		for _, record := range records {
			_, err := decoder.Decode(record.Value)
			if err != nil {
				b.Fatal(err)
			}
		}
	}
}

// Helper function to generate a test trace
func generateRequest(entries, lineLength int) *tempopb.PushBytesRequest {
	stream := &tempopb.PushBytesRequest{
		Traces: make([]tempopb.PreallocBytes, entries),
		Ids:    make([][]byte, entries),
	}

	for i := 0; i < entries; i++ {
		stream.Traces[i].Slice = generateRandomString(lineLength)
		stream.Ids[i] = generateRandomString(lineLength)
	}

	return stream
}

// Helper function to generate a random string
func generateRandomString(length int) []byte {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return b
}

func BenchmarkGeneratorDecoderOTLP(b *testing.B) {
	traceBytes := marshalBatches(b, []*v1.ResourceSpans{
		test.MakeBatch(15, []byte("test batch 1")),
		test.MakeBatch(50, []byte("test batch 2")),
		test.MakeBatch(42, []byte("test batch 3")),
	})

	b.ReportAllocs()
	decoder := NewOTLPDecoder()

	b.ResetTimer()
	for b.Loop() {
		iterator, err := decoder.Decode(traceBytes)
		require.NoError(b, err)
		for range iterator { // nolint:revive // we want to run the side effects of ranging itself
		}
	}
}

func marshalBatches(t testing.TB, batches []*v1.ResourceSpans) []byte {
	t.Helper()

	trace := tempopb.Trace{ResourceSpans: batches}

	m, err := trace.Marshal()
	require.NoError(t, err)

	return m
}

func BenchmarkGeneratorDecoderPushBytes(b *testing.B) {
	stream := generateRequest(1000, 200)
	traceBytes, err := stream.Marshal()
	require.NoError(b, err)

	b.ReportAllocs()
	decoder := NewPushBytesDecoder()

	b.ResetTimer()
	for b.Loop() {
		iterator, err := decoder.Decode(traceBytes)
		require.NoError(b, err)
		for range iterator { // nolint:revive // we want to run the side effects of ranging itself
		}
	}
}
