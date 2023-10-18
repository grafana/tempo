package search

import (
	"context"
	"errors"
	"testing"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/stretchr/testify/require"
)

func TestResultsDoesNotRace(t *testing.T) {
	testCases := []struct {
		name           string
		consumeResults bool
		error          bool
	}{
		{"default", true, false},
		{"exit early", false, false},
		{"exit early due to error", true, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			sr := NewResults()
			defer sr.Close()

			workers := 10
			results := 10_000

			for i := 0; i < workers; i++ {
				sr.StartWorker()
				go func() {
					defer sr.FinishWorker()
					for j := 0; j < results; j++ {
						if sr.AddResult(ctx, &tempopb.TraceSearchMetadata{}) {
							break
						}
					}

					if tc.error {
						sr.SetError(errors.New("test error"))
					}
				}()
			}

			sr.AllWorkersStarted()

			resultsCount := 0
			for range sr.Results() {
				resultsCount++
			}

			// Check error after results channel is closed which
			// means all workers have exited.
			err := sr.Error()

			if tc.error {
				require.Error(t, err)
				if tc.consumeResults {
					// in case of error, we will bail out early
					// and not all results are read
					require.Less(t, resultsCount, workers*results)
					// But we always get at least 1 result before
					// the error
					require.Greater(t, resultsCount, 0)
				}
			} else {
				require.NoError(t, err)
				if tc.consumeResults {
					require.Equal(t, workers*results, resultsCount)
				}
			}
		})
	}
}
