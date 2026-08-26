package drain

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func BenchmarkDrain_TrainExtractsPatterns(b *testing.B) {
	tests := []struct {
		inputFile string
	}{
		{inputFile: `dev1.json`},
		{inputFile: `ops1.json`},
		{inputFile: `prod1.json`},
		{inputFile: `prod2.json`},
		{inputFile: `prod3.json`},
	}

	for _, tt := range tests {
		b.Run(tt.inputFile, func(b *testing.B) {
			linesJSON, err := os.ReadFile(filepath.Join("testdata", tt.inputFile))
			require.NoError(b, err)

			var lines []string
			err = json.Unmarshal(linesJSON, &lines)
			require.NoError(b, err)

			drain := New("test-tenant", DefaultConfig())

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				for _, line := range lines {
					drain.Train(line)
				}
			}
		})
	}
}

func BenchmarkDrain_Prune(b *testing.B) {
	tests := []struct {
		name       string
		clusters   int
		tokenCount int
	}{
		{name: "clusters=1000/tokens=4", clusters: 1_000, tokenCount: 4},
		{name: "clusters=10000/tokens=4", clusters: 10_000, tokenCount: 4},
		{name: "clusters=100000/tokens=4", clusters: 100_000, tokenCount: 4},
		{name: "clusters=10000/tokens=30", clusters: 10_000, tokenCount: 30},
		{name: "clusters=100000/tokens=30", clusters: 100_000, tokenCount: 30},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			config := DefaultConfig()
			config.MaxClusters = tt.clusters
			drain := New("benchmark-prune-"+tt.name, config)

			side := 1
			for side*side < tt.clusters {
				side++
			}
			tokens := make([]string, tt.tokenCount)
			for i := range tokens {
				tokens[i] = benchmarkAlphaToken(i)
			}
			for i := range tt.clusters {
				tokens[0] = benchmarkAlphaToken(i / side)
				tokens[1] = benchmarkAlphaToken(i % side)
				drain.train(tokens)
			}
			require.Len(b, drain.Clusters(), tt.clusters)

			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				drain.Prune()
			}
		})
	}
}

func benchmarkAlphaToken(n int) string {
	const alphabet = "abcdefghijklmnopqrstuvwxyz"

	var token [4]byte
	for i := range token {
		token[i] = alphabet[n%len(alphabet)]
		n /= len(alphabet)
	}
	return string(token[:])
}
