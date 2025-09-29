package livetraces

import (
	"errors"
	"hash"
	"hash/fnv"
	"time"

	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/pkg/util/log"

	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
)

var (
	ErrMaxLiveTracesExceeded = errors.New("max live traces exceeded")
	ErrMaxTraceSizeExceeded  = errors.New("max trace size exceeded")
)

type LiveTraceBatchT interface {
	*v1.ResourceSpans | []byte
}

type LiveTrace[T LiveTraceBatchT] struct {
	ID      []byte
	Batches []T

	LastAppend time.Time
	CreatedAt  time.Time
	sz         uint64
}

type LiveTraces[T LiveTraceBatchT] struct {
	hash   hash.Hash64
	Traces map[uint64]*LiveTrace[T]

	maxIdleTime time.Duration
	maxLiveTime time.Duration

	sz     uint64
	szFunc func(T) uint64

	maxTraceErrorLogger *log.RateLimitedLogger
}

const (
	maxTraceLogLinesPerSecond = 10
)

func New[T LiveTraceBatchT](sizeFunc func(T) uint64, maxIdleTime, maxLiveTime time.Duration, tenantID string) *LiveTraces[T] {
	logger := kitlog.With(log.Logger, "tenant", tenantID)

	return &LiveTraces[T]{
		hash:                fnv.New64(),
		Traces:              make(map[uint64]*LiveTrace[T]),
		szFunc:              sizeFunc,
		maxIdleTime:         maxIdleTime,
		maxLiveTime:         maxLiveTime,
		maxTraceErrorLogger: log.NewRateLimitedLogger(maxTraceLogLinesPerSecond, level.Error(logger)),
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
	return l.PushWithTimestampAndLimits(time.Now(), traceID, batch, max, 0) == nil
}

func (l *LiveTraces[T]) PushWithTimestampAndLimits(ts time.Time, traceID []byte, batch T, maxLiveTraces, maxTraceSize uint64) error {
	token := l.token(traceID)

	tr := l.Traces[token]
	if tr == nil {

		// Before adding this check against max
		// Zero means no limit
		if maxLiveTraces > 0 && uint64(len(l.Traces)) >= maxLiveTraces {
			return ErrMaxLiveTracesExceeded
		}

		tr = &LiveTrace[T]{
			ID:        traceID,
			CreatedAt: ts,
		}
		l.Traces[token] = tr
	}

	sz := l.szFunc(batch)

	// Before adding check against max trace size
	if maxTraceSize > 0 && (tr.sz+sz > maxTraceSize) {
		l.maxTraceErrorLogger.Log("msg", "max trace size exceeded", "max", maxTraceSize, "reqSize", sz, "totalSize", tr.sz, "trace", util.TraceIDToHexString(traceID))
		return ErrMaxTraceSizeExceeded
	}

	tr.sz += sz
	l.sz += sz

	tr.Batches = append(tr.Batches, batch)
	tr.LastAppend = ts
	return nil
}

func (l *LiveTraces[T]) CutIdle(now time.Time, immediate bool) []*LiveTrace[T] {
	res := []*LiveTrace[T]{}

	idleSince := now.Add(-l.maxIdleTime)
	liveSince := now.Add(-l.maxLiveTime)

	for k, tr := range l.Traces {
		if tr.LastAppend.Before(idleSince) || tr.CreatedAt.Before(liveSince) || immediate {
			res = append(res, tr)
			l.sz -= tr.sz
			delete(l.Traces, k)
		}
	}

	return res
}
