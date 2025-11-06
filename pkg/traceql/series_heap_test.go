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
			expected: 1, // reversed: aaa comes before bbb alphabetically, but comparison is reversed
		},
		{
			name: "different label values",
			a: SeriesMapKey{
				{Name: "foo", Value: NewStaticString("aaa").MapKey()},
			},
			b: SeriesMapKey{
				{Name: "foo", Value: NewStaticString("bbb").MapKey()},
			},
			expected: 1, // reversed: aaa comes before bbb alphabetically, but comparison is reversed
		},
		{
			name: "different types",
			a: SeriesMapKey{
				{Name: "foo", Value: NewStaticInt(1).MapKey()},
			},
			b: SeriesMapKey{
				{Name: "foo", Value: NewStaticString("1").MapKey()},
			},
			expected: 1, // reversed: string comparison takes precedence and is reversed
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

func TestDataPointGreaterThan(t *testing.T) {
	key := SeriesMapKey{{Name: "foo", Value: NewStaticString("bar").MapKey()}}
	keyEarlier := SeriesMapKey{{Name: "aaa", Value: NewStaticString("bar").MapKey()}}

	tests := []struct {
		name     string
		a        seriesValue
		b        seriesValue
		expected bool
	}{
		{
			name:     "a value greater than b",
			a:        seriesValue{key: key, value: 10.0},
			b:        seriesValue{key: key, value: 5.0},
			expected: true,
		},
		{
			name:     "a value less than b",
			a:        seriesValue{key: key, value: 1.0},
			b:        seriesValue{key: key, value: 5.0},
			expected: false,
		},
		{
			name:     "equal values, equal keys",
			a:        seriesValue{key: key, value: 5.0},
			b:        seriesValue{key: key, value: 5.0},
			expected: false,
		},
		{
			name:     "equal values, a key alphabetically earlier than b",
			a:        seriesValue{key: keyEarlier, value: 5.0},
			b:        seriesValue{key: key, value: 5.0},
			expected: true, // reversed: alphabetically earlier key now compares as greater
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := dataPointGreaterThan(tt.a, tt.b)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestDataPointLessThan(t *testing.T) {
	key := SeriesMapKey{{Name: "foo", Value: NewStaticString("bar").MapKey()}}
	keyEarlier := SeriesMapKey{{Name: "aaa", Value: NewStaticString("bar").MapKey()}}

	tests := []struct {
		name     string
		a        seriesValue
		b        seriesValue
		expected bool
	}{
		{
			name:     "a value less than b",
			a:        seriesValue{key: key, value: 1.0},
			b:        seriesValue{key: key, value: 5.0},
			expected: true,
		},
		{
			name:     "a value greater than b",
			a:        seriesValue{key: key, value: 10.0},
			b:        seriesValue{key: key, value: 5.0},
			expected: false,
		},
		{
			name:     "equal values, equal keys",
			a:        seriesValue{key: key, value: 5.0},
			b:        seriesValue{key: key, value: 5.0},
			expected: false,
		},
		{
			name:     "equal values, a key alphabetically earlier than b",
			a:        seriesValue{key: keyEarlier, value: 5.0},
			b:        seriesValue{key: key, value: 5.0},
			expected: false, // reversed: alphabetically earlier key now compares as greater, not less
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := dataPointLessThan(tt.a, tt.b)
			require.Equal(t, tt.expected, result)
		})
	}
}
