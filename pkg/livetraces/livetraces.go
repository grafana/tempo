package livetraces

import (
	"hash"
	"hash/fnv"
	"time"

	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
)

type LiveTrace struct {
	ID        []byte
	timestamp time.Time
	Batches   []*v1.ResourceSpans

	sz uint64
}

type LiveTraces struct {
	hash   hash.Hash64
	Traces map[uint64]*LiveTrace

	sz uint64
}

func New() *LiveTraces {
	return &LiveTraces{
		hash:   fnv.New64(),
		Traces: make(map[uint64]*LiveTrace),
	}
}

func (l *LiveTraces) token(traceID []byte) uint64 {
	l.hash.Reset()
	l.hash.Write(traceID)
	return l.hash.Sum64()
}

func (l *LiveTraces) Len() uint64 {
	return uint64(len(l.Traces))
}

func (l *LiveTraces) Size() uint64 {
	return l.sz
}

func (l *LiveTraces) Push(traceID []byte, batch *v1.ResourceSpans, max uint64) bool {
	return l.PushWithTimestamp(time.Now(), traceID, batch, max)
}

func (l *LiveTraces) PushWithTimestamp(ts time.Time, traceID []byte, batch *v1.ResourceSpans, max uint64) bool {
	token := l.token(traceID)

	tr := l.Traces[token]
	if tr == nil {

		// Before adding this check against max
		// Zero means no limit
		if max > 0 && uint64(len(l.Traces)) >= max {
			return false
		}

		tr = &LiveTrace{
			ID: traceID,
		}
		l.Traces[token] = tr
	}

	sz := uint64(batch.Size())
	tr.sz += sz
	l.sz += sz

	tr.Batches = append(tr.Batches, batch)
	tr.timestamp = ts
	return true
}

func (l *LiveTraces) CutIdle(idleSince time.Time, immediate bool) []*LiveTrace {
	res := []*LiveTrace{}

	for k, tr := range l.Traces {
		if tr.timestamp.Before(idleSince) || immediate {
			res = append(res, tr)
			l.sz -= tr.sz
			delete(l.Traces, k)
		}
	}

	return res
}
