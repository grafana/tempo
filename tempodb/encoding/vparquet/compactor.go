package vparquet

import (
	"context"
	"fmt"
	"io"
	"runtime"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/segmentio/parquet-go"

	tempo_io "github.com/grafana/tempo/pkg/io"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

/*type slicePool struct {
	mtx  sync.Mutex
	free []*Trace
}

func (p *slicePool) Get() *Trace {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	if len(p.free) > 0 {
		tr := p.free[0]
		p.free = p.free[1:]
		return tr
	}

	tr := &Trace{}
	tr.ResourceSpans = make([]ResourceSpans, 10)
	for i := range tr.ResourceSpans {
		rs := &tr.ResourceSpans[i]
		rs.InstrumentationLibrarySpans = make([]ILS, 10)
		rs.Resource.Attrs = make([]Attribute, 10)

		for j := range rs.InstrumentationLibrarySpans {
			ils := &rs.InstrumentationLibrarySpans[j]
			ils.Spans = make([]Span, 10)
			for k := range ils.Spans {
				s := &ils.Spans[k]
				s.Attrs = make([]Attribute, 10)
				s.Events = make([]Event, 10)
				for l := range s.Events {
					s.Events[l].Attrs = make([]EventAttribute, 10)
				}
			}
		}
	}
	return tr
}

func (p *slicePool) Put(tr *Trace) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	if len(p.free) < 100 {
		p.free = append(p.free, tr)
	}
}*/

func NewCompactor(opts common.CompactionOptions) common.Compactor {
	return &Compactor{opts: opts}
}

type Compactor struct {
	opts common.CompactionOptions
}

func (c *Compactor) Compact(ctx context.Context, l log.Logger, r backend.Reader, writerCallback func(*backend.BlockMeta, time.Time) backend.Writer, inputs []*backend.BlockMeta) (newCompactedBlocks []*backend.BlockMeta, err error) {

	var minBlockStart, maxBlockEnd time.Time
	bookmarks := make([]*bookmarkRaw, 0, len(inputs))

	var compactionLevel uint8
	var totalRecords int
	for _, blockMeta := range inputs {
		totalRecords += blockMeta.TotalObjects

		if blockMeta.CompactionLevel > compactionLevel {
			compactionLevel = blockMeta.CompactionLevel
		}

		if blockMeta.StartTime.Before(minBlockStart) || minBlockStart.IsZero() {
			minBlockStart = blockMeta.StartTime
		}
		if blockMeta.EndTime.After(maxBlockEnd) {
			maxBlockEnd = blockMeta.EndTime
		}

		block := newBackendBlock(blockMeta, r)

		iter, err := block.RawIterator(ctx)
		if err != nil {
			return nil, err
		}

		// wrap bookmark with a prefetch iterator
		bookmarks = append(bookmarks, newBookmarkRaw(iter))
	}

	nextCompactionLevel := compactionLevel + 1

	recordsPerBlock := (totalRecords / int(c.opts.OutputBlocks))

	sch := parquet.SchemaOf(new(Trace))
	re := func(row parquet.Row) *Trace {
		tr := new(Trace)
		sch.Reconstruct(tr, row)
		return tr
	}
	de := func(tr *Trace) parquet.Row {
		return sch.Deconstruct(nil, tr)
	}

	objsCombined := func() {
		c.opts.ObjectsCombined(int(compactionLevel), 1)
	}

	var currentBlock *streamingBlock
	m := newMultiblockIteratorRaw(re, de, bookmarks, objsCombined)
	defer m.Close()

	for {
		lowestID, lowestObject, err := m.Next(ctx)
		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, errors.Wrap(err, "error iterating input blocks")
		}

		// make a new block if necessary
		if currentBlock == nil {
			// Start with a copy and then customize
			newMeta := &backend.BlockMeta{
				BlockID:         uuid.New(),
				TenantID:        inputs[0].TenantID,
				CompactionLevel: nextCompactionLevel,
				TotalObjects:    recordsPerBlock, // Just an estimate
			}
			w := writerCallback(newMeta, time.Now())

			currentBlock = newStreamingBlock(ctx, &c.opts.BlockConfig, newMeta, r, w, tempo_io.NewBufferedWriter)
			currentBlock.meta.CompactionLevel = nextCompactionLevel
			newCompactedBlocks = append(newCompactedBlocks, currentBlock.meta)
		}

		// Write trace.
		// Note - not specifying trace start/end here, we set the overall block start/stop
		// times from the input metas.
		err = currentBlock.AddRaw(lowestID, lowestObject, 0, 0)
		if err != nil {
			return nil, err
		}

		// write partial block
		//if currentBlock.CurrentBufferLength() >= int(opts.FlushSizeBytes) {
		if currentBlock.CurrentBufferedObjects() > 3000 {
			runtime.GC()
			err = c.appendBlock(currentBlock)
			if err != nil {
				return nil, errors.Wrap(err, "error writing partial block")
			}
		}

		// ship block to backend if done
		if currentBlock.meta.TotalObjects >= recordsPerBlock {
			currentBlockPtrCopy := currentBlock
			currentBlockPtrCopy.meta.StartTime = minBlockStart
			currentBlockPtrCopy.meta.EndTime = maxBlockEnd
			err := c.finishBlock(currentBlockPtrCopy, l)
			if err != nil {
				return nil, errors.Wrap(err, fmt.Sprintf("error shipping block to backend, blockID %s", currentBlockPtrCopy.meta.BlockID.String()))
			}
			currentBlock = nil
		}

		// add trace object into pool for reuse by parquet row reader
		/*if spanCount(lowestObject) < 100_000 {
			reset(lowestObject)
			// Only put small traces back in the pool
			pool.Put(lowestObject)
		}*/
	}

	// ship final block to backend
	if currentBlock != nil {
		currentBlock.meta.StartTime = minBlockStart
		currentBlock.meta.EndTime = maxBlockEnd
		err := c.finishBlock(currentBlock, l)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("error shipping block to backend, blockID %s", currentBlock.meta.BlockID.String()))
		}
	}

	return newCompactedBlocks, nil
}

/*func spanCount(tr *Trace) int {
	c := 0
	for i := range tr.ResourceSpans {
		for j := range tr.ResourceSpans[i].InstrumentationLibrarySpans {
			c += len(tr.ResourceSpans[i].InstrumentationLibrarySpans[j].Spans)
		}
	}
	return c
}

func reset(tr *Trace) {
	//spanCount := 0

	tr.RootServiceName = ""
	tr.RootSpanName = ""
	tr.TraceID = nil
	tr.TraceIDText = ""

	for i := range tr.ResourceSpans {
		rs := &tr.ResourceSpans[i]
		rs.Resource.ServiceName = ""
		rs.Resource.Cluster = nil
		rs.Resource.Namespace = nil
		rs.Resource.Pod = nil
		rs.Resource.Container = nil
		rs.Resource.K8sClusterName = nil
		rs.Resource.K8sNamespaceName = nil
		rs.Resource.K8sPodName = nil
		rs.Resource.K8sContainerName = nil
		rs.Resource.Test = ""

		for a := range rs.Resource.Attrs {
			rs.Resource.Attrs[a] = Attribute{}
		}

		for j := range rs.InstrumentationLibrarySpans {
			ils := &rs.InstrumentationLibrarySpans[j]
			ils.InstrumentationLibrary.Name = ""
			ils.InstrumentationLibrary.Version = ""

			for k := range ils.Spans {
				s := &ils.Spans[k]
				s.Name = ""
				s.ID = nil
				s.ParentSpanID = nil
				s.StatusMessage = ""
				s.TraceState = ""
				s.HttpMethod = nil
				s.HttpStatusCode = nil
				s.HttpUrl = nil

				for a := range s.Attrs {
					s.Attrs[a] = Attribute{}
				}
				for e := range s.Events {
					s.Events[e].Name = ""
					for a := range s.Events[e].Attrs {
						s.Events[e].Attrs[a] = EventAttribute{}
					}
				}
				//spanCount++
			}
		}
	}

	//return spanCount < 100_000
}*/

func (c *Compactor) appendBlock(block *streamingBlock) error {
	compactionLevel := int(block.meta.CompactionLevel - 1)
	if c.opts.ObjectsWritten != nil {
		c.opts.ObjectsWritten(compactionLevel, block.CurrentBufferedObjects())
	}

	bytesFlushed, err := block.Flush()
	if err != nil {
		return err
	}

	if c.opts.BytesWritten != nil {
		c.opts.BytesWritten(compactionLevel, bytesFlushed)
	}

	return nil
}

func (c *Compactor) finishBlock(block *streamingBlock, l log.Logger) error {
	bytesFlushed, err := block.Complete()
	if err != nil {
		return errors.Wrap(err, "error completing block")
	}

	level.Info(l).Log("msg", "wrote compacted block", "meta", fmt.Sprintf("%+v", block.meta))
	compactionLevel := int(block.meta.CompactionLevel) - 1
	if c.opts.BytesWritten != nil {
		c.opts.BytesWritten(compactionLevel, bytesFlushed)
	}
	return nil
}

type bookmark struct {
	iter Iterator

	currentObject *Trace
	currentErr    error
}

func newBookmark(iter Iterator) *bookmark {
	return &bookmark{
		iter: iter,
	}
}

func (b *bookmark) current(ctx context.Context) (*Trace, error) {
	if b.currentErr != nil {
		return nil, b.currentErr
	}

	if b.currentObject != nil {
		return b.currentObject, nil
	}

	b.currentObject, b.currentErr = b.iter.Next(ctx)
	return b.currentObject, b.currentErr
}

func (b *bookmark) done(ctx context.Context) bool {
	obj, err := b.current(ctx)

	return obj == nil || err != nil
}

func (b *bookmark) clear() {
	b.currentObject = nil
}

func (b *bookmark) close() {
	b.iter.Close()
}

type bookmarkRaw struct {
	iter *rawIterator

	currentID     []byte
	currentObject parquet.Row
	currentErr    error
}

func newBookmarkRaw(iter *rawIterator) *bookmarkRaw {
	return &bookmarkRaw{
		iter: iter,
	}
}

func (b *bookmarkRaw) current(ctx context.Context) ([]byte, parquet.Row, error) {
	if b.currentErr != nil {
		return nil, nil, b.currentErr
	}

	if b.currentObject != nil {
		return b.currentID, b.currentObject, nil
	}

	b.currentID, b.currentObject, b.currentErr = b.iter.Next(ctx)
	return b.currentID, b.currentObject, b.currentErr
}

func (b *bookmarkRaw) done(ctx context.Context) bool {
	_, obj, err := b.current(ctx)

	return obj == nil || err != nil
}

func (b *bookmarkRaw) clear() {
	b.currentObject = nil
}

func (b *bookmarkRaw) close() {
	//b.iter.Close()
}
