package localblocks

import (
	"errors"
	"hash"
	"hash/fnv"
	"time"

	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
)

var errMaxExceeded = errors.New("asdf")

type liveTrace struct {
	id        []byte
	timestamp time.Time
	Batches   []*v1.ResourceSpans
}

type liveTraces struct {
	hash   hash.Hash64
	traces map[uint64]*liveTrace
}

func newLiveTraces() *liveTraces {
	return &liveTraces{
		hash:   fnv.New64(),
		traces: map[uint64]*liveTrace{},
	}
}

func (l *liveTraces) token(traceID []byte) uint64 {
	l.hash.Reset()
	l.hash.Write(traceID)
	return l.hash.Sum64()
}

func (l *liveTraces) Len() uint64 {
	return uint64(len(l.traces))
}

func (l *liveTraces) Push(batch *v1.ResourceSpans, max uint64) error {
	if len(batch.ScopeSpans) == 0 || len(batch.ScopeSpans[0].Spans) == 0 {
		return nil
	}

	traceID := batch.ScopeSpans[0].Spans[0].TraceId
	token := l.token(traceID)

	tr := l.traces[token]
	if tr == nil {

		// Before adding this check against max
		// Zero means no limit
		if max > 0 && uint64(len(l.traces)) >= max {
			return errMaxExceeded
		}

		tr = &liveTrace{
			id: traceID,
		}
		l.traces[token] = tr
	}

	tr.Batches = append(tr.Batches, batch)
	tr.timestamp = time.Now()
	return nil
}

func (l *liveTraces) CutIdle(idleSince time.Time) []*liveTrace {
	res := []*liveTrace{}

	for k, tr := range l.traces {
		if tr.timestamp.Before(idleSince) {
			res = append(res, tr)
			delete(l.traces, k)
		}
	}

	return res
}
