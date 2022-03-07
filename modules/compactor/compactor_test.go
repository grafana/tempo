package compactor

import (
	"math"
	"testing"

	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/pkg/model/trace"
	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCombineLimitsNotHit(t *testing.T) {
	o, err := overrides.NewOverrides(overrides.Limits{
		MaxBytesPerTrace: math.MaxInt,
	})
	require.NoError(t, err)

	c := &Compactor{
		overrides: o,
	}

	trace := test.MakeTraceWithSpanCount(2, 10, nil)
	t1 := &tempopb.Trace{
		Batches: []*v1.ResourceSpans{
			trace.Batches[0],
		},
	}
	t2 := &tempopb.Trace{
		Batches: []*v1.ResourceSpans{
			trace.Batches[1],
		},
	}
	obj1 := encode(t, t1)
	obj2 := encode(t, t2)

	actual, wasCombined, err := c.Combine(model.CurrentEncoding, "test", obj1, obj2)
	assert.NoError(t, err)
	assert.Equal(t, true, wasCombined)
	assert.Equal(t, encode(t, trace), actual) // entire trace should be returned
}

func TestCombineLimitsHit(t *testing.T) {
	o, err := overrides.NewOverrides(overrides.Limits{
		MaxBytesPerTrace: 1,
	})
	require.NoError(t, err)

	c := &Compactor{
		overrides: o,
	}

	trace := test.MakeTraceWithSpanCount(2, 10, nil)
	t1 := &tempopb.Trace{
		Batches: []*v1.ResourceSpans{
			trace.Batches[0],
		},
	}
	t2 := &tempopb.Trace{
		Batches: []*v1.ResourceSpans{
			trace.Batches[1],
		},
	}
	obj1 := encode(t, t1)
	obj2 := encode(t, t2)

	actual, wasCombined, err := c.Combine(model.CurrentEncoding, "test", obj1, obj2)
	assert.NoError(t, err)
	assert.Equal(t, true, wasCombined)
	assert.Equal(t, encode(t, t1), actual) // only t1 was returned b/c the combined trace was greater than the threshold
}

func TestCombineDoesntEnforceZero(t *testing.T) {
	o, err := overrides.NewOverrides(overrides.Limits{
		MaxBytesPerTrace: 0,
	})
	require.NoError(t, err)

	c := &Compactor{
		overrides: o,
	}

	trace := test.MakeTraceWithSpanCount(2, 10, nil)
	t1 := &tempopb.Trace{
		Batches: []*v1.ResourceSpans{
			trace.Batches[0],
		},
	}
	t2 := &tempopb.Trace{
		Batches: []*v1.ResourceSpans{
			trace.Batches[1],
		},
	}
	obj1 := encode(t, t1)
	obj2 := encode(t, t2)

	actual, wasCombined, err := c.Combine(model.CurrentEncoding, "test", obj1, obj2)
	assert.NoError(t, err)
	assert.Equal(t, true, wasCombined)
	assert.Equal(t, encode(t, trace), actual) // entire trace should be returned
}

func TestCountSpans(t *testing.T) {
	t1 := test.MakeTraceWithSpanCount(1, 10, nil)
	t2 := test.MakeTraceWithSpanCount(2, 13, nil)
	t1ExpectedSpans := 10
	t2ExpectedSpans := 26

	b1 := encode(t, t1)
	b2 := encode(t, t2)

	assert.Equal(t, t1ExpectedSpans, countSpans(model.CurrentEncoding, b1))
	assert.Equal(t, t2ExpectedSpans, countSpans(model.CurrentEncoding, b2))
	assert.Equal(t,
		t1ExpectedSpans+t2ExpectedSpans,
		countSpans(model.CurrentEncoding, b1, b2))
}

func encode(t *testing.T, tr *tempopb.Trace) []byte {
	trace.SortTrace(tr)

	sd := model.MustNewSegmentDecoder(model.CurrentEncoding)

	segment, err := sd.PrepareForWrite(tr, 0, 0)
	require.NoError(t, err)

	obj, err := sd.ToObject([][]byte{segment})
	require.NoError(t, err)

	return obj
}
