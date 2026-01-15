package livetraces

import (
	"hash"
	"hash/fnv"
	"time"

	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
)

type LiveTraceBatchT interface {
	*v1.ResourceSpans | []byte
}

type LiveTrace[T LiveTraceBatchT] struct {
	ID      []byte
	Batches []T

	lastAppend time.Time
	createdAt  time.Time
	sz         uint64
}

type LiveTraces[T LiveTraceBatchT] struct {
	hash   hash.Hash64
	Traces map[uint64]*LiveTrace[T]

	maxIdleTime time.Duration
	maxLiveTime time.Duration

	sz     uint64
	szFunc func(T) uint64
}

func New[T LiveTraceBatchT](sizeFunc func(T) uint64, maxIdleTime, maxLiveTime time.Duration) *LiveTraces[T] {
	return &LiveTraces[T]{
		hash:        fnv.New64(),
		Traces:      make(map[uint64]*LiveTrace[T]),
		szFunc:      sizeFunc,
		maxIdleTime: maxIdleTime,
		maxLiveTime: maxLiveTime,
	}
}

func (l *LiveTraces[T]) token(traceID []byte) uint64 {
	l.hash.Reset()
	l.hash.Write(traceID)
	return l.hash.Sum64()
}

func (l *LiveTraces[T]) Len() uint64 {
	return uint64(len(l.Traces))
}

func (l *LiveTraces[T]) Size() uint64 {
	return l.sz
}

func (l *LiveTraces[T]) Push(traceID []byte, batch T, max uint64) bool {
	return l.PushWithTimestampAndLimits(time.Now(), traceID, batch, max, 0)
}

func (l *LiveTraces[T]) PushWithTimestampAndLimits(ts time.Time, traceID []byte, batch T, maxLiveTraces, maxTraceSize uint64) (ok bool) {
	token := l.token(traceID)

	tr := l.Traces[token]
	if tr == nil {

		// Before adding this check against max
		// Zero means no limit
		if maxLiveTraces > 0 && uint64(len(l.Traces)) >= maxLiveTraces {
			return false
		}

		tr = &LiveTrace[T]{
			ID:        traceID,
			createdAt: ts,
		}
		l.Traces[token] = tr
	}

	sz := l.szFunc(batch)

	// Before adding check against max trace size
	if maxTraceSize > 0 && (tr.sz+sz > maxTraceSize) {
		return false
	}

	tr.sz += sz
	l.sz += sz

	tr.Batches = append(tr.Batches, batch)
	tr.lastAppend = ts
	return true
}

func (l *LiveTraces[T]) CutIdle(now time.Time, immediate bool) []*LiveTrace[T] {
	res := []*LiveTrace[T]{}

	idleSince := now.Add(-l.maxIdleTime)
	liveSince := now.Add(-l.maxLiveTime)

	for k, tr := range l.Traces {
		if tr.lastAppend.Before(idleSince) || tr.createdAt.Before(liveSince) || immediate {
			res = append(res, tr)
			l.sz -= tr.sz
			delete(l.Traces, k)
		}
	}

	return res
}
