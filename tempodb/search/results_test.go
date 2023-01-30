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

			for i := 0; i < 100; i++ {
				sr.StartWorker()
				go func() {
					defer sr.FinishWorker()
					for j := 0; j < 10_000; j++ {
						if sr.AddResult(ctx, &tempopb.TraceSearchMetadata{}) {
							break
						}
					}

					if tc.error {
						sr.AddError(ctx, errors.New("test error"))
					}
				}()
			}

			var err error
			go func() {
				for err = range sr.Errors() {
					sr.Close()
					return
				}
			}()

			sr.AllWorkersStarted()

			var resultsCount int
			if tc.consumeResults {
				for range sr.Results() {
					resultsCount++
				}
			}

			if tc.error {
				require.Error(t, err)
				if tc.consumeResults {
					// in case of error, we will bail out early
					require.NotEqual(t, 10_000_00, resultsCount)
					// will read at-least something by the time we have first error
					require.NotEqual(t, 0, resultsCount)
				}
			} else {
				require.NoError(t, err)
				if tc.consumeResults {
					require.Equal(t, 10_000_00, resultsCount)
				}
			}
		})
	}
}
