package main

import (
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/go-test/deep"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	thrift "github.com/jaegertracing/jaeger/thrift-gen/jaeger"
	"github.com/stretchr/testify/assert"
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
		"VlBzgbaiCM",
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
			Key:  "CMRAjWwhTHctcuAx",
			VStr: stringPointer("lBzgba"),
		},
		{
			Key:  "oEFfRsWxPLDnJOb",
			VStr: stringPointer("xKQFDaFpLSjFbc"),
		},
		{
			Key:  "eQYhYzRyWJjP",
			VStr: stringPointer("sNVlgTeMaPEZQ"),
		},
	}
	result := generateRandomTags(r)
	require.Equal(t, expected, result)
}

func TestGenerateRandomLogs(t *testing.T) {
	now := time.Now()

	r := rand.New(rand.NewSource(1))

	expected := []*thrift.Log{
		{
			Timestamp: now.Unix(),
			Fields: []*thrift.Tag{
				{
					Key:  "whTHctcuAx",
					VStr: stringPointer("BzgbaiCMRAj"),
				},
				{
					Key:  "oEFfRsWxPLDnJOb",
					VStr: stringPointer("xKQFDaFpLSjFbc"),
				},
				{
					Key:  "eQYhYzRyWJjP",
					VStr: stringPointer("sNVlgTeMaPEZQ"),
				},
				{
					Key:  "ZRjxAwnwekrBEmf",
					VStr: stringPointer("zpfRFEgmotaFetHs"),
				},
			},
		},
		{
			Timestamp: now.Unix(),
			Fields: []*thrift.Tag{
				{
					Key:  "TMtTCoaN",
					VStr: stringPointer("cEkXBAkjQZLC"),
				},
				{
					Key:  "tNswYNsGRussVmaozFZ",
					VStr: stringPointer("tyyiNKAReKJyiXJrsc"),
				},
				{
					Key:  "GLOpbU",
					VStr: stringPointer("sbOJiFQGZsnwTKSmVo"),
				},
				{
					Key:  "VjaRzLNTXYeUCWKs",
					VStr: stringPointer("pEdKupdOMe"),
				},
			},
		},
		{
			Timestamp: now.Unix(),
			Fields: []*thrift.Tag{
				{
					Key:  "SJfjzaLbtZsyMGe",
					VStr: stringPointer("yRAOmBTv"),
				},
				{
					Key:  "HOvgSeycJPJHYNufN",
					VStr: stringPointer("DtRzQMDQiYCOh"),
				},
				{
					Key:  "qfgqVMkPYVkU",
					VStr: stringPointer("JhhjUVRu"),
				},
				{
					Key:  "rKCtzkjkZIvaBj",
					VStr: stringPointer("UpiFvIZRgBmy"),
				},
			},
		},
	}
	result := generateRandomLogs(r, now)
	require.Equal(t, expected, result)
}

func TestIntervalsBetween(t *testing.T) {
	nowish := time.Unix(1630010049, 0)

	cases := []struct {
		start     time.Time
		stop      time.Time
		interval  time.Duration
		retention time.Duration
		count     int
	}{
		{
			start:     nowish.Add(-1 * time.Minute),
			stop:      nowish,
			interval:  11 * time.Second,
			retention: 1 * time.Hour,
			count:     7,
		},
		{
			start:     nowish.Add(-1 * time.Hour),
			stop:      nowish,
			interval:  33 * time.Second,
			retention: 1 * time.Hour,
			count:     110,
		},
	}

	for _, tc := range cases {
		result := intervalsBetween(tc.start, tc.stop, tc.interval, tc.retention)
		require.Equal(t, tc.count, len(result))

		if tc.count > 0 {
			require.Equal(t, tc.start, result[0])
			require.True(t, result[len(result)-1].Before(tc.stop))
		}
	}
}

func TestEqualTraces(t *testing.T) {
	now := time.Now()

	require.Equal(t, newRand(now).Int(), newRand(now).Int())

	r1 := newRand(now)
	batch1 := makeThriftBatch(r1.Int63(), r1.Int63(), r1, now)
	pb1 := jaegerBatchToPbTrace(batch1)

	r2 := newRand(now)
	batch2 := makeThriftBatch(r2.Int63(), r2.Int63(), r2, now)
	pb2 := jaegerBatchToPbTrace(batch2)

	require.Equal(t, pb1, pb2)
	require.True(t, equalTraces(pb1, pb2))
}

func TestNewRand(t *testing.T) {
	now := time.Now()

	r1 := newRand(now)
	r2 := newRand(now)
	r3 := newRand(now)
	r4 := newRand(now)

	for _, x := range []*rand.Rand{r1, r2, r3, r4} {
		x.Int63()
		x.Int63()
		x.Int63()
		x.Int63()
		generateRandomString(x)
		generateRandomString(x)
		generateRandomString(x)
		generateRandomString(x)
		generateRandomString(x)
	}

	v := generateRandomString(r1)
	for _, x := range []*rand.Rand{r2, r3, r4} {
		require.Equal(t, v, generateRandomString(x))
	}
}

func TestResponseFixture(t *testing.T) {
	f, err := os.Open("testdata/trace.json")
	require.NoError(t, err)
	defer f.Close()

	response := &tempopb.Trace{}
	err = jsonpb.Unmarshal(f, response)
	require.NoError(t, err)

	seed := time.Unix(1630010049, 0)
	expected := constructTraceFromEpoch(seed)

	assert.True(t, equalTraces(expected, response))

	if diff := deep.Equal(expected, response); diff != nil {
		for _, d := range diff {
			t.Error(d)
		}
	}
}

func stringPointer(s string) *string { return &s }
