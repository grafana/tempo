package search

import (
	"context"
	"testing"

	"github.com/grafana/tempo/pkg/tempopb"
)

func TestResultsDoesNotRace(t *testing.T) {

	testCases := []struct {
		name           string
		consumeResults bool
	}{
		{"default", true},
		{"exit early", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			sr := NewResults()
			defer sr.Close()

			for i := 0; i < 100; i++ {
				sr.StartWorker()
				go func() {
					defer sr.FinishWorker()

					for j := 0; j < 10_000; j++ {
						if sr.AddResult(ctx, &tempopb.TraceSearchMetadata{}) {
							break
						}
					}
				}()
			}

			sr.AllWorkersStarted()

			if tc.consumeResults {
				for range sr.Results() {
				}
			}
		})
	}
}
