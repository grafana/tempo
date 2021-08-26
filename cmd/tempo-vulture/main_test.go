package main

import (
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/go-test/deep"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/grafana/tempo/pkg/tempopb"
	v1common "github.com/grafana/tempo/pkg/tempopb/common/v1"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/jaegertracing/jaeger/thrift-gen/jaeger"
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
	now := time.Now()
	cases := []struct {
		start    time.Time
		stop     time.Time
		interval time.Duration
		count    int
	}{
		{
			start:    now.Add(-1 * time.Minute),
			stop:     now,
			interval: 11 * time.Second,
			count:    6,
		},
		{
			start:    now.Add(-1 * time.Hour),
			stop:     now,
			interval: 33 * time.Second,
			count:    110,
		},
	}

	for _, tc := range cases {
		result := intervalsBetween(tc.start, tc.stop, tc.interval)
		require.Equal(t, tc.count, len(result))

		if tc.count > 0 {
			require.Equal(t, tc.start, result[0])
			require.True(t, result[len(result)-1].Before(tc.stop))
		}
	}
}

func TestTrimOutdatedIntervals(t *testing.T) {
	now := time.Now()

	cases := []struct {
		start     time.Time
		stop      time.Time
		interval  time.Duration
		retention time.Duration
		count     int
	}{
		{
			start:     now.Add(-1 * time.Minute),
			stop:      now,
			interval:  11 * time.Second,
			count:     3,
			retention: 30 * time.Second,
		},
		{
			start:     now.Add(-1 * time.Hour),
			stop:      now,
			interval:  33 * time.Second,
			count:     110,
			retention: 24 * time.Hour,
		},
		{
			start:     now.Add(-25 * time.Hour),
			stop:      now,
			interval:  33 * time.Second,
			count:     2619,
			retention: 24 * time.Hour,
		},
	}

	for _, tc := range cases {
		intervals := intervalsBetween(tc.start, tc.stop, tc.interval)
		intervals = trimOutdatedIntervals(intervals, tc.retention)

		require.NotNil(t, intervals)
		require.Equal(t, tc.count, len(intervals))

		require.True(t, intervals[len(intervals)-1].Before(tc.stop))
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

func TestJagerBatchToPbTrace(t *testing.T) {
	nowish, err := time.Parse(time.RFC3339, "2021-08-25T10:15:31-06:00")
	require.NoError(t, err)
	r1 := newRand(nowish)

	cases := []struct {
		batch    *jaeger.Batch
		expected *tempopb.Trace
	}{
		{
			batch: makeThriftBatch(r1.Int63(), r1.Int63(), r1, nowish),
			expected: &tempopb.Trace{
				Batches: []*v1.ResourceSpans{
					{
						InstrumentationLibrarySpans: []*v1.InstrumentationLibrarySpans{
							{
								Spans: []*v1.Span{
									{
										Name:    "tgvrGx",
										TraceId: []byte("0855b1342fd8f95914c1a43aae1b6c73"),
										SpanId:  []byte("39d2aec5556367e6"),
										Events: []*v1.Span_Event{
											{
												TimeUnixNano: 1629908131,
												Attributes: []*v1common.KeyValue{
													{
														Key: "fQKOABsE",
														Value: &v1common.AnyValue{
															Value: &v1common.AnyValue_StringValue{StringValue: "rcEoMvnzLV"},
														},
													},
													{
														Key: "IAeBiaSjF",
														Value: &v1common.AnyValue{
															Value: &v1common.AnyValue_StringValue{StringValue: "YRmKJvUVSWD"},
														},
													},
												},
											},
											{
												TimeUnixNano: 1629908131,
												Attributes: []*v1common.KeyValue{
													{
														Key: "hirzqJYPlSpH",
														Value: &v1common.AnyValue{
															Value: &v1common.AnyValue_StringValue{StringValue: "GNowiq"},
														},
													},
													{
														Key: "obDItNCcENJowMkHL",
														Value: &v1common.AnyValue{
															Value: &v1common.AnyValue_StringValue{StringValue: "xmcFdTYhG"},
														},
													},
												},
											},
											{
												TimeUnixNano: 1629908131,
												Attributes: []*v1common.KeyValue{
													{
														Key: "GrGxVkxyURMAFXv",
														Value: &v1common.AnyValue{
															Value: &v1common.AnyValue_StringValue{StringValue: "dZiIwT"},
														},
													},
													{
														Key: "HXfssLZ",
														Value: &v1common.AnyValue{
															Value: &v1common.AnyValue_StringValue{StringValue: "njhpnYnxIaSiU"},
														},
													},
													{
														Key: "kvuXiofamXq",
														Value: &v1common.AnyValue{
															Value: &v1common.AnyValue_StringValue{StringValue: "bcxZfGtAvySamMkU"},
														},
													},
													{
														Key: "AlSiwOjoJRVL",
														Value: &v1common.AnyValue{
															Value: &v1common.AnyValue_StringValue{StringValue: "KiEJFRaxYF"},
														},
													},
												},
											},
										},
										Attributes: []*v1common.KeyValue{
											{
												Key: "yZJMlOghq",
												Value: &v1common.AnyValue{
													Value: &v1common.AnyValue_StringValue{StringValue: "paPjzXSe"},
												},
											},
											{
												Key: "QDbZDAMYeQ",
												Value: &v1common.AnyValue{
													Value: &v1common.AnyValue_StringValue{StringValue: "rXBYKuwWuCS"},
												},
											},
										},
									},
									{
										Name:    "BUZFbNNaAauNncS",
										TraceId: []byte("0855b1342fd8f95914c1a43aae1b6c73"),
										SpanId:  []byte("3d278f63b44225b7"),
										Events: []*v1.Span_Event{
											{
												TimeUnixNano: 1629908131,
												Attributes: []*v1common.KeyValue{
													{
														Key: "KwxnCyqkJ",
														Value: &v1common.AnyValue{
															Value: &v1common.AnyValue_StringValue{StringValue: "CfZmobCxYfqCkWR"},
														},
													},
													{
														Key: "KQWDjfuUkSsLnxecMIC",
														Value: &v1common.AnyValue{
															Value: &v1common.AnyValue_StringValue{StringValue: "zSgonMaGYEhS"},
														},
													},
													{
														Key: "yOuqOyqCOvsdFGfW",
														Value: &v1common.AnyValue{
															Value: &v1common.AnyValue_StringValue{StringValue: "ueDVCAMFiqMNPTr"},
														},
													},
												},
											},
											{
												TimeUnixNano: 1629908131,
												Attributes: []*v1common.KeyValue{
													{
														Key: "vbgCDBMw",
														Value: &v1common.AnyValue{
															Value: &v1common.AnyValue_StringValue{StringValue: "NnCdKhtWYsLq"},
														},
													},
													{
														Key: "qborQobPng",
														Value: &v1common.AnyValue{
															Value: &v1common.AnyValue_StringValue{StringValue: "jKsNTBoFbpvlaKkS"},
														},
													},
													{
														Key: "FDRdKcKVtOAUvtKHnA",
														Value: &v1common.AnyValue{
															Value: &v1common.AnyValue_StringValue{StringValue: "wrqriKCgwG"},
														},
													},
													{
														Key: "IFSlfd",
														Value: &v1common.AnyValue{
															Value: &v1common.AnyValue_StringValue{StringValue: "LbsCDuTrkW"},
														},
													},
												},
											},
											{
												TimeUnixNano: 1629908131,
												Attributes: []*v1common.KeyValue{
													{
														Key: "iPSWrw",
														Value: &v1common.AnyValue{
															Value: &v1common.AnyValue_StringValue{StringValue: "VOHOFrBpolYmHozHW"},
														},
													},
													{
														Key: "HoUnCBfEAQMFGW",
														Value: &v1common.AnyValue{
															Value: &v1common.AnyValue_StringValue{StringValue: "pncMfPLHwyqA"},
														},
													},
												},
											},
										},
										Attributes: []*v1common.KeyValue{
											{
												Key: "XfdGaWAVs",
												Value: &v1common.AnyValue{
													Value: &v1common.AnyValue_StringValue{StringValue: "NEoskqRbDSmtzKvZkp"},
												},
											},
											{
												Key: "VLqUydNkPt",
												Value: &v1common.AnyValue{
													Value: &v1common.AnyValue_StringValue{StringValue: "vPxSkJq"},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range cases {
		result := jaegerBatchToPbTrace(tc.batch)
		if diff := deep.Equal(tc.expected, result); diff != nil {
			t.Logf("expected: %+v", tc.expected)
			t.Logf("result: %+v", result)
			for _, d := range diff {
				t.Error(d)
			}
		}
		// require.Equal(t, tc.expected, result)
	}
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
