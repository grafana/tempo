package main

import (
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
		result := generateRandomInt(tc.min, tc.max)
		require.Equal(t, tc.result, result)
	}
}

func TestGenerateRandomString(t *testing.T) {

	strings := []string{
		"zgbaiCMRAjWwhTHc",
	}

	for _, s := range strings {
		result := generateRandomString()
		require.Equal(t, s, result)
	}
}

func TestGenerateRandomTags(t *testing.T) {
	expected := []*thrift.Tag{
		{
			Key:  "cXoEFfRsWxPLDnJOb",
			VStr: stringPointer("uAxhxKQFDaFpLSjF"),
		},
		{
			Key:  "eQYhYzRyWJjP",
			VStr: stringPointer("sNVlgTeMaPEZQ"),
		},
	}
	result := generateRandomTags()
	require.Equal(t, expected, result)
}

func TestGenerateRandomLogs(t *testing.T) {
	expected := []*thrift.Log{
		{
			Timestamp: time.Now().Unix(),
			Fields: []*thrift.Tag{
				{
					Key:  "tHsbZRjxAwnwekrBEmf",
					VStr: stringPointer("fRFEgmotaF"),
				},
				{
					Key:  "QZLCtTMt",
					VStr: stringPointer("zdcEkXBAk"),
				},
				{
					Key:  "AReKJy",
					VStr: stringPointer("CoaNatyyiN"),
				},
				{
					Key:  "ussVma",
					VStr: stringPointer("XJrscctNswYNsG"),
				},
			},
		},
		{
			Timestamp: time.Now().Unix(),
			Fields: []*thrift.Tag{
				{
					Key:  "GZsnwTKSmVoiG",
					VStr: stringPointer("FZBsbOJiF"),
				},
				{
					Key:  "VjaRzLNTXYeUCWKs",
					VStr: stringPointer("OpbUOpEdKupdOMe"),
				},
				{
					Key:  "LbtZsyMGeu",
					VStr: stringPointer("bGyRAOmBTvKSJfjz"),
				},
			},
		},
	}
	result := generateRandomLogs()
	require.Equal(t, expected, result)
}

func stringPointer(s string) *string { return &s }
