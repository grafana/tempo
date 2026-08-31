package drain

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

type fixtureData struct {
	OriginalCount  int
	FinalCount     int
	PatternMapping map[string]string
}

func TestFixtures(t *testing.T) {
	fixtures := []string{"dev1", "ops1", "prod1", "prod2", "prod3"}
	for _, fixture := range fixtures {
		t.Run(fixture, func(t *testing.T) {
			linesJSON, err := os.ReadFile(filepath.Join("testdata", fixture+".json"))
			require.NoError(t, err)

			var lines []string
			require.NoError(t, json.Unmarshal(linesJSON, &lines))

			expectedJSON, err := os.ReadFile(filepath.Join("testdata", fixture+".drain"))
			require.NoError(t, err)

			var expected fixtureData
			require.NoError(t, json.Unmarshal(expectedJSON, &expected))

			drain := New(t.Name(), DefaultConfig())
			patternMapping := make(map[string]string, len(lines))
			for _, line := range lines {
				cluster := drain.Train(line)
				if cluster == nil {
					patternMapping[line] = "<nil> (possibly too many tokens)"
					continue
				}
				patternMapping[line] = cluster.String()
			}

			require.Equal(t, expected.OriginalCount, len(lines))
			require.Equal(t, expected.FinalCount, len(drain.Clusters()))
			require.Equal(t, expected.PatternMapping, patternMapping)
		})
	}
}
