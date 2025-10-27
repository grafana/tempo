package localentitylimiter

import (
	"context"
	"sync"
	"time"
)

type LocalEntityLimiter struct {
	mtx               sync.Mutex
	entityLastUpdated map[uint64]time.Time
	maxEntityFunc     func(tenant string) uint32
	staleDuration     time.Duration
}

func NewLocalEntityLimiter(maxEntityFunc func(tenant string) uint32, staleDuration time.Duration) *LocalEntityLimiter {
	return &LocalEntityLimiter{
		maxEntityFunc:     maxEntityFunc,
		entityLastUpdated: make(map[uint64]time.Time),
		staleDuration:     staleDuration,
	}
}

func (l *LocalEntityLimiter) TrackEntities(ctx context.Context, tenant string, hashes []uint64) (rejected []uint64, err error) {
	maxEntities := l.maxEntityFunc(tenant)
	if maxEntities == 0 {
		return nil, nil
	}

	now := time.Now()

	l.mtx.Lock()
	defer l.mtx.Unlock()

	for _, hash := range hashes {
		_, exists := l.entityLastUpdated[hash]

		if exists {
			l.entityLastUpdated[hash] = now
			continue
		}

		if uint32(len(l.entityLastUpdated)) >= maxEntities {
			rejected = append(rejected, hash)
			continue
		}

		l.entityLastUpdated[hash] = now
	}

	return rejected, nil
}

func (l *LocalEntityLimiter) Prune(ctx context.Context) {
	l.mtx.Lock()
	defer l.mtx.Unlock()

	for hash, lastUpdated := range l.entityLastUpdated {
		if time.Since(lastUpdated) > l.staleDuration {
			delete(l.entityLastUpdated, hash)
		}
	}
}
