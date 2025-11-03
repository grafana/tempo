package traceql

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompareSeriesMapKey(t *testing.T) {
	tests := []struct {
		name     string
		a        SeriesMapKey
		b        SeriesMapKey
		expected int
	}{
		{
			name: "equal keys",
			a: SeriesMapKey{
				{Name: "foo", Value: NewStaticString("bar").MapKey()},
			},
			b: SeriesMapKey{
				{Name: "foo", Value: NewStaticString("bar").MapKey()},
			},
			expected: 0,
		},
		{
			name: "different label names",
			a: SeriesMapKey{
				{Name: "aaa", Value: NewStaticString("bar").MapKey()},
			},
			b: SeriesMapKey{
				{Name: "bbb", Value: NewStaticString("bar").MapKey()},
			},
			expected: -1,
		},
		{
			name: "different label values",
			a: SeriesMapKey{
				{Name: "foo", Value: NewStaticString("aaa").MapKey()},
			},
			b: SeriesMapKey{
				{Name: "foo", Value: NewStaticString("bbb").MapKey()},
			},
			expected: -1,
		},
		{
			name: "different types",
			a: SeriesMapKey{
				{Name: "foo", Value: NewStaticInt(1).MapKey()},
			},
			b: SeriesMapKey{
				{Name: "foo", Value: NewStaticString("1").MapKey()},
			},
			expected: -1, // int type < string type
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compareSeriesMapKey(tt.a, tt.b)
			require.Equal(t, tt.expected, result)

			// Test symmetry
			if tt.expected != 0 {
				require.Equal(t, -tt.expected, compareSeriesMapKey(tt.b, tt.a))
			}
		})
	}
}

func TestCompareSeriesValues(t *testing.T) {
	key1 := SeriesMapKey{{Name: "aaa", Value: NewStaticString("bar").MapKey()}}
	key2 := SeriesMapKey{{Name: "bbb", Value: NewStaticString("bar").MapKey()}}

	tests := []struct {
		name     string
		a        seriesValue
		b        seriesValue
		expected int
	}{
		{
			name:     "a less than b",
			a:        seriesValue{key: key1, value: 1.0},
			b:        seriesValue{key: key1, value: 2.0},
			expected: -1,
		},
		{
			name:     "a greater than b",
			a:        seriesValue{key: key1, value: 2.0},
			b:        seriesValue{key: key1, value: 1.0},
			expected: 1,
		},
		{
			name:     "equal values, equal keys",
			a:        seriesValue{key: key1, value: 1.0},
			b:        seriesValue{key: key1, value: 1.0},
			expected: 0,
		},
		{
			name:     "equal values, different keys - tiebreaker",
			a:        seriesValue{key: key1, value: 1.0},
			b:        seriesValue{key: key2, value: 1.0},
			expected: -1, // key1 < key2 alphabetically
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compareSeriesValues(tt.a, tt.b)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestDataPointGreaterThan(t *testing.T) {
	key := SeriesMapKey{{Name: "foo", Value: NewStaticString("bar").MapKey()}}
	keyEarlier := SeriesMapKey{{Name: "aaa", Value: NewStaticString("bar").MapKey()}}
	val := seriesValue{key: key, value: 5.0}

	tests := []struct {
		name     string
		newValue float64
		newKey   SeriesMapKey
		expected bool
	}{
		{
			name:     "new value greater",
			newValue: 10.0,
			newKey:   key,
			expected: true,
		},
		{
			name:     "new value less",
			newValue: 1.0,
			newKey:   key,
			expected: false,
		},
		{
			name:     "equal value, equal key",
			newValue: 5.0,
			newKey:   key,
			expected: false,
		},
		{
			name:     "equal value, alphabetically earlier key",
			newValue: 5.0,
			newKey:   keyEarlier,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := dataPointGreaterThan(tt.newValue, tt.newKey, val)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestDataPointLessThan(t *testing.T) {
	key := SeriesMapKey{{Name: "foo", Value: NewStaticString("bar").MapKey()}}
	keyEarlier := SeriesMapKey{{Name: "aaa", Value: NewStaticString("bar").MapKey()}}
	val := seriesValue{key: key, value: 5.0}

	tests := []struct {
		name     string
		newValue float64
		newKey   SeriesMapKey
		expected bool
	}{
		{
			name:     "new value less",
			newValue: 1.0,
			newKey:   key,
			expected: true,
		},
		{
			name:     "new value greater",
			newValue: 10.0,
			newKey:   key,
			expected: false,
		},
		{
			name:     "equal value, equal key",
			newValue: 5.0,
			newKey:   key,
			expected: false,
		},
		{
			name:     "equal value, alphabetically earlier key",
			newValue: 5.0,
			newKey:   keyEarlier,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := dataPointLessThan(tt.newValue, tt.newKey, val)
			require.Equal(t, tt.expected, result)
		})
	}
}
