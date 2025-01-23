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
	ID        []byte
	timestamp time.Time
	Batches   []T

	sz uint64
}

type LiveTraces[T LiveTraceBatchT] struct {
	hash   hash.Hash64
	Traces map[uint64]*LiveTrace[T]

	sz     uint64
	szFunc func(T) uint64
}

func New[T LiveTraceBatchT](sizeFunc func(T) uint64) *LiveTraces[T] {
	return &LiveTraces[T]{
		hash:   fnv.New64(),
		Traces: make(map[uint64]*LiveTrace[T]),
		szFunc: sizeFunc,
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
	return l.PushWithTimestamp(time.Now(), traceID, batch, max)
}

func (l *LiveTraces[T]) PushWithTimestamp(ts time.Time, traceID []byte, batch T, max uint64) bool {
	token := l.token(traceID)

	tr := l.Traces[token]
	if tr == nil {

		// Before adding this check against max
		// Zero means no limit
		if max > 0 && uint64(len(l.Traces)) >= max {
			return false
		}

		tr = &LiveTrace[T]{
			ID: traceID,
		}
		l.Traces[token] = tr
	}

	sz := l.szFunc(batch)
	tr.sz += sz
	l.sz += sz

	tr.Batches = append(tr.Batches, batch)
	tr.timestamp = ts
	return true
}

func (l *LiveTraces[T]) CutIdle(idleSince time.Time, immediate bool) []*LiveTrace[T] {
	res := []*LiveTrace[T]{}

	for k, tr := range l.Traces {
		if tr.timestamp.Before(idleSince) || immediate {
			res = append(res, tr)
			l.sz -= tr.sz
			delete(l.Traces, k)
		}
	}

	return res
}
