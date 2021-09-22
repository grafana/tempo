package main

import (
	"math/rand"
	"os"
	"strings"
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
			result: 4,
		},
		{
			min:    10,
			max:    50,
			result: 33,
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
		"VlBzgbaiCMRAjWwhTH",
	}

	for _, s := range strings {
		result := generateRandomString(r)
		require.Equal(t, s, result)
	}
}

func TestGenerateRandomTags(t *testing.T) {
	r := rand.New(rand.NewSource(1))

	result := generateRandomTags(r)

	for _, k := range result {
		assertStandardVultureKey(t, k)
	}
}

func TestGenerateRandomLogs(t *testing.T) {
	now := time.Now()
	r := rand.New(rand.NewSource(1))
	result := generateRandomLogs(r, now)

	for _, l := range result {
		require.NotNil(t, l.Timestamp)
		for _, f := range l.Fields {
			assertStandardVultureKey(t, f)
		}
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

	seed := time.Unix(1632146180, 0)
	expected := constructTraceFromEpoch(seed)

	assert.True(t, equalTraces(expected, response))

	if diff := deep.Equal(expected, response); diff != nil {
		for _, d := range diff {
			t.Error(d)
		}
	}
}

func TestEqualTraces(t *testing.T) {
	seed := time.Now()
	a := constructTraceFromEpoch(seed)
	b := constructTraceFromEpoch(seed)
	require.True(t, equalTraces(a, b))
}

func TestTraceInfo(t *testing.T) {
	seed := time.Unix(1632146180, 0)
	info := newTraceInfo(seed)
	assert.False(t, info.ready(seed))
	assert.False(t, info.ready(seed.Add(tempoLongWriteBackoffDuration)))
	assert.False(t, info.ready(seed.Add(tempoLongWriteBackoffDuration).Add(1*time.Second)))
	assert.True(t, info.ready(seed.Add(2*tempoLongWriteBackoffDuration)))
}

func assertStandardVultureKey(t *testing.T, tag *thrift.Tag) {
	if !strings.HasPrefix(tag.Key, "vulture-") {
		t.Errorf("prefix vulture- is wanted, have: %s", tag.Key)
	}

	require.NotNil(t, tag.VStr)
	require.GreaterOrEqual(t, len(tag.VType.String()), 5)
	require.LessOrEqual(t, len(tag.VType.String()), 20)
}
