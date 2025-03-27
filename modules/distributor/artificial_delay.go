package distributor

import (
	"math/rand"
	"time"
)

func (d *Distributor) padWithArtificialDelay(reqStart time.Time, userID string) {
	artificialDelay := d.cfg.ArtificialDelay
	if artificialDelayOverride, ok := d.overrides.IngestionArtificialDelay(userID); ok {
		artificialDelay = artificialDelayOverride
	}

	if artificialDelay <= 0 {
		return
	}

	// delay = targetDelay - time spent processing the request up until now
	// If the request took longer than the target delay, we don't delay at all as sleep will return immediately for a negative value
	reqDuration := d.now().Sub(reqStart)
	delay := artificialDelay - reqDuration
	d.sleep(durationWithJitter(delay, 0.10))
}

// durationWithJitter returns random duration from "input - input*variance" to "input + input*variance" interval.
func durationWithJitter(input time.Duration, variancePerc float64) time.Duration {
	variance := int64(float64(input) * variancePerc)
	if variance <= 0 {
		return 0
	}

	jitter := rand.Int63n(variance*2) - variance

	return input + time.Duration(jitter)
}
