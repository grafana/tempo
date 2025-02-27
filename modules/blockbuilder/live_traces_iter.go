package blockbuilder

import (
	"bytes"
	"context"
	"slices"
	"sync"

	"github.com/grafana/tempo/pkg/livetraces"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

type entry struct {
	id   common.ID
	hash uint64
}

type chEntry struct {
	id  common.ID
	tr  *tempopb.Trace
	err error
}

// liveTracesIter iterates through a liveTraces proto bytes map, exposing
// a common.Iterator interface. Uses channel internally so that unmarshaling
// can be done concurrently with the consumption of the iterator.
// Tracks the min/max timestamps seen across all traces that can be accessed
// once all traces are iterated (unmarshaled), since this can't be known upfront.
type liveTracesIter struct {
	mtx        sync.Mutex
	liveTraces *livetraces.LiveTraces[[]byte]
	ch         chan []chEntry
	chBuf      []chEntry
	cancel     func()
	start, end uint64
}

func newLiveTracesIter(liveTraces *livetraces.LiveTraces[[]byte]) *liveTracesIter {
	ctx, cancel := context.WithCancel(context.Background())

	l := &liveTracesIter{
		liveTraces: liveTraces,
		ch:         make(chan []chEntry, 1),
		cancel:     cancel,
	}

	go l.iter(ctx)

	return l
}

func (i *liveTracesIter) Next(ctx context.Context) (common.ID, *tempopb.Trace, error) {
	if len(i.chBuf) == 0 {
		select {
		case entries, ok := <-i.ch:
			if !ok {
				return nil, nil, nil
			}
			i.chBuf = entries
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		}
	}

	// Pop next entry
	if len(i.chBuf) > 0 {
		entry := i.chBuf[0]
		i.chBuf = i.chBuf[1:]
		return entry.id, entry.tr, entry.err
	}

	// Channel is open but buffer is empty?
	return nil, nil, nil
}

func (i *liveTracesIter) iter(ctx context.Context) {
	i.mtx.Lock()
	defer i.mtx.Unlock()
	defer close(i.ch)

	// Get the list of all traces sorted by ID
	entries := make([]entry, 0, len(i.liveTraces.Traces))
	for hash, t := range i.liveTraces.Traces {
		entries = append(entries, entry{t.ID, hash})
	}
	slices.SortFunc(entries, func(a, b entry) int {
		return bytes.Compare(a.id, b.id)
	})

	// Begin sending to channel in chunks to reduce channel overhead.
	seq := slices.Chunk(entries, 10)
	for entries := range seq {
		output := make([]chEntry, 0, len(entries))

		for _, e := range entries {

			entry := i.liveTraces.Traces[e.hash]

			tr := new(tempopb.Trace)

			for _, b := range entry.Batches {
				// This unmarshal appends the batches onto the existing tempopb.Trace
				// so we don't need to allocate another container temporarily
				err := tr.Unmarshal(b)
				if err != nil {
					i.ch <- []chEntry{{err: err}}
					return
				}
			}

			// Update block timestamp bounds
			for _, b := range tr.ResourceSpans {
				for _, ss := range b.ScopeSpans {
					for _, s := range ss.Spans {
						if i.start == 0 || s.StartTimeUnixNano < i.start {
							i.start = s.StartTimeUnixNano
						}
						if s.EndTimeUnixNano > i.end {
							i.end = s.EndTimeUnixNano
						}
					}
				}
			}

			tempopb.ReuseByteSlices(entry.Batches)
			delete(i.liveTraces.Traces, e.hash)

			output = append(output, chEntry{
				id:  entry.ID,
				tr:  tr,
				err: nil,
			})
		}

		select {
		case i.ch <- output:
		case <-ctx.Done():
			return
		}
	}
}

// MinMaxTimestamps returns the earliest start, and latest end span timestamps,
// which can't be known until all contents are unmarshaled. The iterator must
// be exhausted before this can be accessed.
func (i *liveTracesIter) MinMaxTimestamps() (uint64, uint64) {
	i.mtx.Lock()
	defer i.mtx.Unlock()

	return i.start, i.end
}

func (i *liveTracesIter) Close() {
	i.cancel()
}

var _ common.Iterator = (*liveTracesIter)(nil)
