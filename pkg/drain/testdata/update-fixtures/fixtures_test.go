package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grafana/tempo/pkg/drain"
	"github.com/stretchr/testify/require"
)

func TestFixtures(t *testing.T) {
	files, err := os.ReadDir("..")
	if err != nil {
		t.Fatalf("failed to read testdata directory: %v", err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		if !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		t.Run(file.Name(), func(t *testing.T) {
			linesJSON, err := os.ReadFile(filepath.Join("..", file.Name()))
			if err != nil {
				t.Fatalf("failed to read file: %v", err)
			}
			var lines []string
			err = json.Unmarshal(linesJSON, &lines)
			if err != nil {
				t.Fatalf("failed to unmarshal file: %v", err)
			}
			drain := drain.New("test-tenant", drain.DefaultConfig())
			patternMapping := make(map[string]string)
			for _, line := range lines {
				cluster := drain.Train(line)
				if cluster == nil {
					patternMapping[line] = "<nil> (possibly too many tokens)"
				} else {
					patternMapping[line] = cluster.String()
				}
			}

			expectedBytes, err := os.ReadFile(filepath.Join("..", strings.TrimSuffix(file.Name(), ".json")+".drain"))
			if err != nil {
				t.Fatalf("failed to read file: %v", err)
			}
			var expected TestData
			err = json.Unmarshal(expectedBytes, &expected)
			if err != nil {
				t.Fatalf("failed to unmarshal file: %v", err)
			}
			require.Equal(t, expected.PatternMapping, patternMapping)
		})
	}
}
