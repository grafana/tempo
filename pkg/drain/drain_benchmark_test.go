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
