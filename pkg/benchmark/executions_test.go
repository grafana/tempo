package benchmark

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCaseMerges(t *testing.T) {
	outputs := []execOutput{
		{matched: 2, keys: []string{"a", "b"}},
		{matched: 2, keys: []string{"b", "c"}},
		{matched: 1},
	}

	tests := []struct {
		name    string
		merge   func([]execOutput, RunOptions) int64
		outputs []execOutput
		opts    RunOptions
		want    int64
	}{
		{"sum", mergeSum, outputs, RunOptions{}, 5},
		{"search under the limit", mergeSearch, outputs, RunOptions{SearchLimit: 10}, 3},
		{"search capped at the limit", mergeSearch, outputs, RunOptions{SearchLimit: 2}, 2},
		{"distinct", mergeDistinct, outputs, RunOptions{}, 3},
		{"distinct, no keys", mergeDistinct, []execOutput{{matched: 1}}, RunOptions{}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.merge(tt.outputs, tt.opts))
		})
	}
}
