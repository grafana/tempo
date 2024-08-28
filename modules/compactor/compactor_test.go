package compactor

import (
	"math"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/pkg/model/trace"
	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/backend"
)

func TestCombineLimitsNotHit(t *testing.T) {
	o, err := overrides.NewOverrides(overrides.Config{
		Defaults: overrides.Overrides{
			Global: overrides.GlobalOverrides{
				MaxBytesPerTrace: math.MaxInt,
			},
		},
	}, nil, prometheus.DefaultRegisterer)
	require.NoError(t, err)

	c := &Compactor{
		overrides: o,
	}

	trace := test.MakeTraceWithSpanCount(2, 10, nil)
	t1 := &tempopb.Trace{
		ResourceSpans: []*v1.ResourceSpans{
			trace.ResourceSpans[0],
		},
	}
	t2 := &tempopb.Trace{
		ResourceSpans: []*v1.ResourceSpans{
			trace.ResourceSpans[1],
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
	o, err := overrides.NewOverrides(overrides.Config{
		Defaults: overrides.Overrides{
			Global: overrides.GlobalOverrides{
				MaxBytesPerTrace: 1,
			},
		},
	}, nil, prometheus.DefaultRegisterer)
	require.NoError(t, err)

	c := &Compactor{
		overrides: o,
	}

	trace := test.MakeTraceWithSpanCount(2, 10, nil)
	t1 := &tempopb.Trace{
		ResourceSpans: []*v1.ResourceSpans{
			trace.ResourceSpans[0],
		},
	}
	t2 := &tempopb.Trace{
		ResourceSpans: []*v1.ResourceSpans{
			trace.ResourceSpans[1],
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
	o, err := overrides.NewOverrides(overrides.Config{
		Defaults: overrides.Overrides{
			Global: overrides.GlobalOverrides{
				MaxBytesPerTrace: math.MaxInt,
			},
		},
	}, nil, prometheus.DefaultRegisterer)
	require.NoError(t, err)

	c := &Compactor{
		overrides: o,
	}

	trace := test.MakeTraceWithSpanCount(2, 10, nil)
	t1 := &tempopb.Trace{
		ResourceSpans: []*v1.ResourceSpans{
			trace.ResourceSpans[0],
		},
	}
	t2 := &tempopb.Trace{
		ResourceSpans: []*v1.ResourceSpans{
			trace.ResourceSpans[1],
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

	b1Total := countSpans(model.CurrentEncoding, b1)
	b2Total := countSpans(model.CurrentEncoding, b2)
	total := countSpans(model.CurrentEncoding, b1, b2)

	assert.Equal(t, t1ExpectedSpans, b1Total)
	assert.Equal(t, t2ExpectedSpans, b2Total)
	assert.Equal(t, t1ExpectedSpans+t2ExpectedSpans, total)
}

func TestDedicatedColumns(t *testing.T) {
	o, err := overrides.NewOverrides(overrides.Config{
		Defaults: overrides.Overrides{
			Storage: overrides.StorageOverrides{
				DedicatedColumns: backend.DedicatedColumns{
					{Scope: "resource", Name: "dedicated.resource.1", Type: "string"},
					{Scope: "span", Name: "dedicated.span.1", Type: "string"},
				},
			},
		},
	}, nil, prometheus.DefaultRegisterer)
	require.NoError(t, err)

	c := &Compactor{overrides: o}

	tr := test.AddDedicatedAttributes(test.MakeTraceWithSpanCount(2, 10, nil))
	t1 := &tempopb.Trace{
		ResourceSpans: []*v1.ResourceSpans{
			tr.ResourceSpans[0],
		},
	}
	t2 := &tempopb.Trace{
		ResourceSpans: []*v1.ResourceSpans{
			tr.ResourceSpans[1],
		},
	}
	obj1 := encode(t, t1)
	obj2 := encode(t, t2)

	actual, wasCombined, err := c.Combine(model.CurrentEncoding, "test", obj1, obj2)
	assert.NoError(t, err)
	assert.Equal(t, true, wasCombined)
	assert.Equal(t, encode(t, tr), actual) // entire trace should be returned
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
