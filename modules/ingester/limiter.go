package ingester

import (
	"fmt"
	"math"

	"github.com/grafana/tempo/v2/modules/overrides"
)

const (
	errMaxTracesPerUserLimitExceeded = "per-user traces limit (local: %d global: %d actual local: %d) exceeded"
)

// RingCount is the interface exposed by a ring implementation which allows
// to count members
type RingCount interface {
	HealthyInstancesCount() int
}

// Limiter implements primitives to get the maximum number of traces
// an ingester can handle for a specific tenant
type Limiter struct {
	limits            overrides.Interface
	ring              RingCount
	replicationFactor int
}

// NewLimiter makes a new limiter
func NewLimiter(limits overrides.Interface, ring RingCount, replicationFactor int) *Limiter {
	return &Limiter{
		limits:            limits,
		ring:              ring,
		replicationFactor: replicationFactor,
	}
}

// AssertMaxTracesPerUser ensures limit has not been reached compared to the current
// number of streams in input and returns an error if so.
func (l *Limiter) AssertMaxTracesPerUser(userID string, traces int) error {
	actualLimit := l.maxTracesPerUser(userID)
	if traces < actualLimit {
		return nil
	}

	localLimit := l.limits.MaxLocalTracesPerUser(userID)
	globalLimit := l.limits.MaxGlobalTracesPerUser(userID)

	return fmt.Errorf(errMaxTracesPerUserLimitExceeded, localLimit, globalLimit, actualLimit)
}

func (l *Limiter) maxTracesPerUser(userID string) int {
	localLimit := l.limits.MaxLocalTracesPerUser(userID)

	// We can assume that traces are evenly distributed across ingesters
	// so we do convert the global limit into a local limit
	globalLimit := l.limits.MaxGlobalTracesPerUser(userID)
	localLimit = l.minNonZero(localLimit, l.convertGlobalToLocalLimit(userID, globalLimit))

	// If both the local and global limits are disabled, we just
	// use the largest int value
	if localLimit == 0 {
		localLimit = math.MaxInt32
	}

	return localLimit
}

func (l *Limiter) convertGlobalToLocalLimit(userID string, globalLimit int) int {
	if globalLimit == 0 {
		return 0
	}

	// Given we don't need a super accurate count (ie. when the ingesters
	// topology changes) and we prefer to always be in favor of the tenant,
	// we can use a per-ingester limit equal to:
	// (global limit / number of ingesters) * replication factor
	ingestionShardSize := l.limits.IngestionTenantShardSize(userID)
	totalIngesters := l.ring.HealthyInstancesCount()
	numIngesters := l.minNonZero(ingestionShardSize, totalIngesters)

	// May happen because the number of ingesters is asynchronously updated.
	// If happens, we just temporarily ignore the global limit.
	if numIngesters > 0 {
		return int((float64(globalLimit) / float64(numIngesters)) * float64(l.replicationFactor))
	}

	return 0
}

func (l *Limiter) minNonZero(first, second int) int {
	if first == 0 || (second != 0 && first > second) {
		return second
	}

	return first
}
