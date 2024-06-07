package util

import (
	"errors"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHexStringToTraceID(t *testing.T) {
	tc := []struct {
		id          string
		expected    []byte
		expectError error
	}{
		{
			id:       "12",
			expected: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x12},
		},
		{
			id:       "1234567890abcdef", // 64 bit
			expected: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
		},
		{
			id:       "1234567890abcdef1234567890abcdef", // 128 bit
			expected: []byte{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef, 0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
		},
		{
			id:          "121234567890abcdef1234567890abcdef", // value too long
			expected:    []byte{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef, 0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
			expectError: errors.New("trace IDs can't be larger than 128 bits"),
		},
		{
			id:       "234567890abcdef", // odd length
			expected: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
		},
		{
			id:          "1234567890abcdef ", // trailing space
			expected:    nil,
			expectError: errors.New("trace IDs can only contain hex characters: invalid character ' ' at position 17"),
		},
	}

	for _, tt := range tc {
		t.Run(tt.id, func(t *testing.T) {
			actual, err := HexStringToTraceID(tt.id)

			if tt.expectError != nil {
				assert.Equal(t, tt.expectError, err)
				assert.Nil(t, actual)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestTraceIDToHexString(t *testing.T) {
	tc := []struct {
		byteID  []byte
		traceID string
	}{
		{
			byteID:  []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x12},
			traceID: "12",
		},
		{
			byteID:  []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
			traceID: "1234567890abcdef", // 64 bit
		},
		{
			byteID:  []byte{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef, 0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
			traceID: "1234567890abcdef1234567890abcdef", // 128 bit
		},
		{
			byteID:  []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x12, 0xa0},
			traceID: "12a0", // trailing zero
		},
	}

	for _, tt := range tc {
		t.Run(tt.traceID, func(t *testing.T) {
			actual := TraceIDToHexString(tt.byteID)

			assert.Equal(t, tt.traceID, actual)
		})
	}
}

func TestSpanIDToHexString(t *testing.T) {
	tc := []struct {
		byteID []byte
		spanID string
	}{
		{
			byteID: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x12},
			spanID: "0000000000000012",
		},
		{
			byteID: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
			spanID: "1234567890abcdef", // 64 bit
		},
		{
			byteID: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x12, 0xa0},
			spanID: "00000000000012a0", // trailing zero
		},
		{
			byteID: []byte{0x12, 0xa0},
			spanID: "00000000000012a0", // less than 64 bytes
		},
		{
			byteID: []byte{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef, 0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
			spanID: "1234567890abcdef1234567890abcdef", // 128 bit
		},
		{
			byteID: []byte{0x00, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef, 0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
			spanID: "34567890abcdef1234567890abcdef", // 128 bit with leading zeroes
		},
	}

	for _, tt := range tc {
		t.Run(tt.spanID, func(t *testing.T) {
			actual := SpanIDToHexString(tt.byteID)

			assert.Equal(t, tt.spanID, actual)
		})
	}
}

func TestSpanIDToUint64(t *testing.T) {
	tc := []struct {
		spanID   []byte
		expected uint64
	}{
		{
			spanID:   []byte{0x60, 0xd8, 0xa9, 0xbd},
			expected: 0xbd_a9_d8_60,
		},
		{
			spanID:   []byte{0x8e, 0xf6, 0x37, 0x90, 0x22, 0x57, 0xb7, 0x43},
			expected: 0x43_b7_57_22_90_37_f6_8e,
		},
		{
			spanID:   []byte{0x18, 0xcc, 0xd9, 0x6d, 0x70, 0xc1, 0xbd, 0xf9},
			expected: 0xf9_bd_c1_70_6d_d9_cc_18,
		},
		{
			spanID:   []byte{0x8e, 0xf6, 0x37, 0x90, 0x22, 0x57, 0xb7, 0x43, 0xff},
			expected: 0x43_b7_57_22_90_37_f6_8e,
		},
	}

	for _, tt := range tc {
		token := SpanIDToUint64(tt.spanID)
		assert.Equalf(t, tt.expected, token, "SpanIDToToken(%v) reurned 0x%x but 0x%x was expected", tt.spanID, token, tt.expected)
	}
}

func TestSpanIDAndKindToToken(t *testing.T) {
	tc := []struct {
		spanID   []byte
		expected uint64
	}{
		{
			spanID: []byte{0x60, 0xd8, 0xa9, 0xbd},
		},
		{
			spanID: []byte{0x8e, 0xf6, 0x37, 0x90, 0x22, 0x57, 0xb7, 0x43},
		},
		{
			spanID: []byte{0x18, 0xcc, 0xd9, 0x6d, 0x70, 0xc1, 0xbd, 0xf9},
		},
		{
			spanID: []byte{0x8e, 0xf6, 0x37, 0x90, 0x22, 0x57, 0xb7, 0x43, 0xff},
		},
	}

	for _, tt := range tc {
		tokenIDOnly := SpanIDToUint64(tt.spanID)
		tokensForKind := map[uint64]struct{}{}

		for kind := 0; kind < 8; kind++ {
			token := SpanIDAndKindToToken(tt.spanID, kind)

			_, exists := tokensForKind[token]
			assert.False(t, exists, "token expected to be unique for different span kind")
			assert.NotEqual(t, tokenIDOnly, token)
			tokensForKind[token] = struct{}{}
		}
	}
}

var tokenToPreventOptimization uint64

func BenchmarkSpanIDAndKindToToken(b *testing.B) {
	type testDataSpanID struct {
		SpanID []byte
		Kind   int
	}

	randomTestCasesSpanID := func(n int, idLen int) []testDataSpanID {
		testCases := make([]testDataSpanID, 0, n)
		for i := 0; i < n; i++ {
			id := make([]byte, idLen)
			for j := range id {
				id[j] = byte(rand.Intn(256))
			}
			testCases = append(testCases, testDataSpanID{SpanID: id, Kind: rand.Intn(6)})
		}
		return testCases
	}

	benchmarks := []struct {
		name string
		data []testDataSpanID
	}{
		{
			name: "id length 4",
			data: randomTestCasesSpanID(1_000, 4),
		},
		{
			name: "id length 8",
			data: randomTestCasesSpanID(1_000, 8),
		},
	}
	for _, bc := range benchmarks {
		b.Run(bc.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				d := bc.data[i%len(bc.data)]
				tokenToPreventOptimization = SpanIDAndKindToToken(d.SpanID, d.Kind)
			}
			b.ReportAllocs()
		})
	}
}

func TestEqualHexStringTraceIDs(t *testing.T) {
	a := "82f6471b46d25e23418a0a99d4c2cda"
	b := "082f6471b46d25e23418a0a99d4c2cda"

	v, err := EqualHexStringTraceIDs(a, b)
	assert.Nil(t, err)
	assert.True(t, v)
}

func TestPadTraceIDTo16Bytes(t *testing.T) {
	tc := []struct {
		name     string
		tid      []byte
		expected []byte
	}{
		{
			name:     "small",
			tid:      []byte{0x01, 0x02},
			expected: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x02},
		},
		{
			name:     "exact",
			tid:      []byte{0x01, 0x02, 0x01, 0x02, 0x01, 0x02, 0x01, 0x02, 0x01, 0x02, 0x01, 0x02, 0x01, 0x02, 0x01, 0x02},
			expected: []byte{0x01, 0x02, 0x01, 0x02, 0x01, 0x02, 0x01, 0x02, 0x01, 0x02, 0x01, 0x02, 0x01, 0x02, 0x01, 0x02},
		},
		{ // least significant bits are preserved
			name:     "large",
			tid:      []byte{0x05, 0x05, 0x01, 0x02, 0x01, 0x02, 0x01, 0x02, 0x01, 0x02, 0x01, 0x02, 0x01, 0x02, 0x01, 0x02, 0x01, 0x02},
			expected: []byte{0x01, 0x02, 0x01, 0x02, 0x01, 0x02, 0x01, 0x02, 0x01, 0x02, 0x01, 0x02, 0x01, 0x02, 0x01, 0x02},
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, PadTraceIDTo16Bytes(tt.tid))
		})
	}
}

func TestHexStringToSpanID(t *testing.T) {
	tc := []struct {
		id          string
		expected    []byte
		expectError error
	}{
		{
			id:       "000eda96db732100",
			expected: []byte{0x0e, 0xda, 0x96, 0xdb, 0x73, 0x21, 0x00},
		},
		{
			id:       "0000000000000002",
			expected: []byte{0x02},
		},
		{
			id:       "1234567890abcdef", // 64 bit
			expected: []byte{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
		},
	}

	for _, tt := range tc {
		t.Run(tt.id, func(t *testing.T) {
			actual, err := HexStringToSpanID(tt.id)

			if tt.expectError != nil {
				assert.Equal(t, tt.expectError, err)
				assert.Nil(t, actual)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, actual)
		})
	}
}
