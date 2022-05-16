package parquetquery

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTracker(t *testing.T) {
	tr := NewTracker()
	require.Equal(t, tracker{-1, -1, -1, -1, -1, -1}, tr)

	steps := []struct {
		repetitionLevel int
		definitionLevel int
		expected        tracker
	}{
		// Name.Language.Country examples from the Dremel whitepaper
		{0, 3, tracker{0, 0, 0, 0, -1, -1}},
		{2, 2, tracker{0, 0, 1, -1, -1, -1}},
		{1, 1, tracker{0, 1, -1, -1, -1, -1}},
		{1, 3, tracker{0, 2, 0, 0, -1, -1}},
		{0, 1, tracker{1, 0, -1, -1, -1, -1}},
	}

	for _, step := range steps {
		tr.Next(step.repetitionLevel, step.definitionLevel)
		require.Equal(t, step.expected, tr)
	}
}

func TestCompareTracker(t *testing.T) {
	testCases := []struct {
		a, b     tracker
		expected int
	}{
		{tracker{-1}, tracker{0}, -1},
		{tracker{0}, tracker{0}, 0},
		{tracker{1}, tracker{0}, 1},

		{tracker{0, 1}, tracker{0, 2}, -1},
		{tracker{0, 2}, tracker{0, 1}, 1},
	}

	for _, tc := range testCases {
		require.Equal(t, tc.expected, CompareTrackers(5, tc.a, tc.b))
	}
}
