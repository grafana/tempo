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

func (l *liveTraces) Push(traceID []byte, batches []*v1.ResourceSpans, maxTraces uint64) bool {
	token := l.token(traceID)

	tr := l.traces[token]
	if tr == nil {

		// Before adding this check against max
		// Zero means no limit
		if maxTraces > 0 && uint64(len(l.traces)) >= maxTraces {
			return false
		}

		tr = &liveTrace{
			id: traceID,
		}
		l.traces[token] = tr
	}

	tr.Batches = append(tr.Batches, batches...)
	tr.timestamp = time.Now()
	return true
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
