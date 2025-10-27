package localentitylimiter

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLocalEntityLimiter_TrackEntities(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		maxFunc := func(string) uint32 {
			return 10
		}
		limiter := NewLocalEntityLimiter(maxFunc, 1*time.Hour)

		ticks := time.NewTicker(1 * time.Minute)
		t.Cleanup(ticks.Stop)

		go func() {
			for {
				select {
				case <-ticks.C:
					limiter.Prune(t.Context())
				case <-t.Context().Done():
					return
				}
			}
		}()

		// First we fill the limiter
		rejected, err := limiter.TrackEntities(t.Context(), "test", []uint64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10})
		require.NoError(t, err)
		require.Empty(t, rejected)

		// Now check that it rejects new entities
		rejected, err = limiter.TrackEntities(t.Context(), "test", []uint64{11})
		require.NoError(t, err)
		require.Equal(t, []uint64{11}, rejected)

		// After half the stale duration, we update 9/10 entities to they stick around
		time.Sleep(30 * time.Minute)
		rejected, err = limiter.TrackEntities(t.Context(), "test", []uint64{1, 2, 3, 4, 5, 6, 7, 8, 9})
		require.NoError(t, err)
		require.Empty(t, rejected)

		// After the stale duration, the entity 10 should have been removed Note
		// we add 2 minutes here to account for any delay in starting the prune
		// loop. The test is still fully deterministic due to synctest.
		time.Sleep(32 * time.Minute)

		// Now we can admit entity 11
		rejected, err = limiter.TrackEntities(t.Context(), "test", []uint64{11})
		require.NoError(t, err)
		require.Empty(t, rejected)

		// And finally, check that entity 10 is rejected
		rejected, err = limiter.TrackEntities(t.Context(), "test", []uint64{10})
		require.NoError(t, err)
		require.Equal(t, []uint64{10}, rejected)
	})
}
