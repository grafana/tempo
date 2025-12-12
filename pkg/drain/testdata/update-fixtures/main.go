package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/grafana/tempo/pkg/drain"
	"github.com/prometheus/client_golang/prometheus"
)

type TestData struct {
	OriginalCount  int
	FinalCount     int
	ReductionPct   float64
	PatternMapping map[string]string
}

// This is a tool to generate test fixtures for the drain package.
func main() {
	// For each .txt file in the testdata directory, read the file and run DRAIN on it.
	// The output should be written to a new file in the testdata directory with the same name but with the suffix .drain

	files, err := os.ReadDir("pkg/drain/testdata")
	if err != nil {
		log.Fatalf("failed to read testdata directory: %v", err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		if !strings.HasSuffix(file.Name(), ".json") {
			continue
		}
		linesJSON, err := os.ReadFile(filepath.Join("pkg/drain/testdata", file.Name()))
		if err != nil {
			log.Fatalf("failed to read file: %v", err)
		}
		var lines []string
		err = json.Unmarshal(linesJSON, &lines)
		if err != nil {
			log.Fatalf("failed to unmarshal file: %v", err)
		}
		patternMapping := make(map[string]string)
		reg := prometheus.NewPedanticRegistry()
		metrics := drain.NewMetrics(reg)
		drain := drain.New(drain.DefaultConfig(), metrics)
		for _, line := range lines {
			cluster := drain.Train(line)
			if cluster == nil {
				patternMapping[line] = "<nil> (possibly too many tokens)"
			} else {
				patternMapping[line] = cluster.String()
			}
		}
		clusters := drain.Clusters()
		testData := TestData{
			OriginalCount:  len(lines),
			FinalCount:     len(clusters),
			ReductionPct:   100 * float64(len(lines)-len(clusters)) / float64(len(lines)),
			PatternMapping: patternMapping,
		}

		fmt.Printf("finished %s\n reduction: %f%%\n", file.Name(), testData.ReductionPct) //nolint:forbidigo

		json, err := json.MarshalIndent(testData, "", "  ")
		if err != nil {
			log.Fatalf("failed to marshal test data: %v", err)
		}
		err = os.WriteFile(filepath.Join("pkg/drain/testdata", strings.TrimSuffix(file.Name(), ".json")+".drain"), json, 0600)
		if err != nil {
			log.Fatalf("failed to write file: %v", err)
		}
	}
}
