package drain

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
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
				drain.train(drain.tokenizer.Join(tokens), tokens)
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

func BenchmarkDrain_ExactMatch(b *testing.B) {
	d := New(b.Name(), DefaultConfig())
	target := "GET /api/users"
	d.Train(target)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		d.Train(target)
	}
}

func BenchmarkDrain_CollapsedLeafExactMatch(b *testing.B) {
	for _, population := range []int{100, 1000, 10000} {
		b.Run(strconv.Itoa(population), func(b *testing.B) {
			d, lines, _ := newCollapsedDrain(b, population)
			target := lines[population-1]

			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				d.Train(target)
			}
		})
	}
}

func BenchmarkDrain_DefaultConfigSteadyState(b *testing.B) {
	linesJSON, err := os.ReadFile(filepath.Join("testdata", "prod3.json"))
	require.NoError(b, err)
	var lines []string
	require.NoError(b, json.Unmarshal(linesJSON, &lines))

	d := New(b.Name(), DefaultConfig())
	for _, line := range lines {
		d.Train(line)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		for _, line := range lines {
			d.Train(line)
		}
	}
	b.ReportMetric(float64(len(d.exactLocations)), "indexed_clusters")
}

func BenchmarkDrain_PruneMaxClusters(b *testing.B) {
	cfg := DefaultConfig()
	d := New(b.Name(), cfg)
	leaf := newNode()
	leaf.exactIndexBuilt = true
	leaf.exactClusters = make(map[uint64]int, cfg.MaxClusters)
	d.rootNode.keyToChildNode["2"] = leaf

	for i := 1; i <= cfg.MaxClusters; i++ {
		cluster := &LogCluster{
			id:          i,
			Tokens:      []string{"GET", "<END>"},
			Stringer:    d.tokenizer.Join,
			ParamString: cfg.ParamString,
		}
		d.idToCluster.Put(cluster)
		leaf.clusterIDs = append(leaf.clusterIDs, i)
		hash := uint64(i)
		leaf.exactClusters[hash] = i
		d.exactLocations[i] = exactLocation{node: leaf, hash: hash}
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		d.Prune()
	}
}
