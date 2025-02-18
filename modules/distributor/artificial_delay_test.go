package distributor

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/modules/overrides"
)

func Test_outerMaybeDelayMiddleware(t *testing.T) {
	tests := []struct {
		name          string
		userID        string
		delay         time.Duration
		reqDuration   time.Duration
		expectedSleep time.Duration
	}{
		{
			name:          "No delay configured",
			userID:        "user1",
			delay:         0,
			reqDuration:   50 * time.Millisecond,
			expectedSleep: 0,
		},
		{
			name:          "Delay configured but request took longer than delay",
			userID:        "user2",
			delay:         500 * time.Millisecond,
			reqDuration:   750 * time.Millisecond,
			expectedSleep: 0,
		},
		{
			name:          "Delay configured and request took less than delay",
			userID:        "user3",
			delay:         500 * time.Millisecond,
			reqDuration:   50 * time.Millisecond,
			expectedSleep: 450 * time.Millisecond,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			limits := overrides.Overrides{
				Ingestion: overrides.IngestionOverrides{
					ArtificialDelay: tc.delay,
				},
			}
			// Init limits overrides
			overrides, err := overrides.NewOverrides(overrides.Config{
				Defaults: limits,
			}, nil, prometheus.DefaultRegisterer)
			require.NoError(t, err)

			// Mock to capture sleep and advance time.
			timeSource := &MockTimeSource{CurrentTime: time.Now()}
			reqStart := timeSource.CurrentTime

			d := &Distributor{
				overrides: overrides,
				sleep:     timeSource.Sleep,
				now:       timeSource.Now,
			}

			// Add time spent on request
			timeSource.Add(tc.reqDuration)

			d.padWithArtificialDelay(reqStart, tc.userID)

			// Due to the 10% jitter we need to take into account that the number will not be deterministic in tests.
			difference := timeSource.Slept - tc.expectedSleep
			require.LessOrEqual(t, difference.Abs(), tc.expectedSleep/10)
		})
	}
}

type MockTimeSource struct {
	CurrentTime time.Time
	Slept       time.Duration
}

func (m *MockTimeSource) Now() time.Time {
	return m.CurrentTime
}

func (m *MockTimeSource) Sleep(d time.Duration) {
	if d > 0 {
		m.Slept += d
	}
}

func (m *MockTimeSource) Add(d time.Duration) {
	m.CurrentTime = m.CurrentTime.Add(d)
}
