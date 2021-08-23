package main

import (
	"math/rand"
	"testing"
	"time"

	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	thrift "github.com/jaegertracing/jaeger/thrift-gen/jaeger"
	"github.com/stretchr/testify/require"
)

func TestHasMissingSpans(t *testing.T) {
	cases := []struct {
		trace   *tempopb.Trace
		expeted bool
	}{
		{
			&tempopb.Trace{
				Batches: []*v1.ResourceSpans{
					{
						InstrumentationLibrarySpans: []*v1.InstrumentationLibrarySpans{
							{
								Spans: []*v1.Span{
									{
										ParentSpanId: []byte("01234"),
									},
								},
							},
						},
					},
				},
			},
			true,
		},
		{
			&tempopb.Trace{
				Batches: []*v1.ResourceSpans{
					{
						InstrumentationLibrarySpans: []*v1.InstrumentationLibrarySpans{
							{
								Spans: []*v1.Span{
									{
										SpanId: []byte("01234"),
									},
									{
										ParentSpanId: []byte("01234"),
									},
								},
							},
						},
					},
				},
			},
			false,
		},
	}

	for _, tc := range cases {
		require.Equal(t, tc.expeted, hasMissingSpans(tc.trace))
	}
}

func TestGenerateRandomInt(t *testing.T) {
	r := rand.New(rand.NewSource(1))

	cases := []struct {
		min    int64
		max    int64
		result int64
	}{
		{
			min:    1,
			max:    5,
			result: 3,
		},
		{
			min:    10,
			max:    50,
			result: 41,
		},
		{
			min:    1,
			max:    3,
			result: 2,
		},
	}

	for _, tc := range cases {
		result := generateRandomInt(tc.min, tc.max, r)
		require.Equal(t, tc.result, result)
	}
}

func TestGenerateRandomString(t *testing.T) {
	r := rand.New(rand.NewSource(1))

	strings := []string{
		"XVlBzgbaiC",
	}

	for _, s := range strings {
		result := generateRandomString(r)
		require.Equal(t, s, result)
	}
}

func TestGenerateRandomTags(t *testing.T) {
	r := rand.New(rand.NewSource(1))

	expected := []*thrift.Tag{
		{
			Key:  "hTHctcuAxhx",
			VStr: stringPointer("MRAjWw"),
		},
		{
			Key:  "FfRsWxP",
			VStr: stringPointer("KQFDaFpLSjFbcXoE"),
		},
		{
			Key:  "lgTeMaPE",
			VStr: stringPointer("LDnJObCsNV"),
		},
	}
	result := generateRandomTags(r)
	require.Equal(t, expected, result)
}

func TestGenerateRandomLogs(t *testing.T) {
	r := rand.New(rand.NewSource(1))

	expected := []*thrift.Log{
		{
			Timestamp: time.Now().Unix(),
			Fields: []*thrift.Tag{
				{
					Key:  "WJjPjzpfRFEgmota",
					VStr: stringPointer("ZQleQYhYzRy"),
				},
				{
					Key:  "RjxAwnwekr",
					VStr: stringPointer("FetHsbZ"),
				},
				{
					Key:  "EkXBAkjQZLCtT",
					VStr: stringPointer("BEmfdzdc"),
				},
				{
					Key:  "eKJyiXJrscctNswYNsG",
					VStr: stringPointer("MtTCoaNatyyiNKAR"),
				},
			},
		},
		{
			Timestamp: time.Now().Unix(),
			Fields: []*thrift.Tag{
				{
					Key:  "FZBsbOJiFQG",
					VStr: stringPointer("RussVmaoz"),
				},
				{
					Key:  "iGLOpbUOpEdKu",
					VStr: stringPointer("ZsnwTKSmVo"),
				},
				{
					Key:  "TXYeUC",
					VStr: stringPointer("pdOMeRVjaRzLN"),
				},
				{
					Key:  "mBTvKSJfjza",
					VStr: stringPointer("WKsXbGyRAO"),
				},
			},
		},
		{
			Timestamp: time.Now().Unix(),
			Fields: []*thrift.Tag{
				{
					Key:  "DQiYCOhgHOv",
					VStr: stringPointer("LbtZsyMGeuDtRzQM"),
				},
				{
					Key:  "fNjJhhjUVRuSqfgqVM",
					VStr: stringPointer("gSeycJPJHYNu"),
				},
			},
		},
	}
	result := generateRandomLogs(r)
	require.Equal(t, expected, result)
}

func stringPointer(s string) *string { return &s }
