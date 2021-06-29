package ingester

import (
	"context"
	"encoding/hex"
	"fmt"
	"hash"
	"hash/fnv"
	"sync"
	"time"

	"github.com/cortexproject/cortex/pkg/util/log"
	"github.com/go-kit/kit/log/level"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/status"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/atomic"
	"google.golang.org/grpc/codes"

	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/model"
	tempofb "github.com/grafana/tempo/pkg/tempofb/Tempo"
	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/validation"
	"github.com/grafana/tempo/tempodb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/grafana/tempo/tempodb/wal"
)

// Errors returned on Query.
var (
	ErrTraceMissing = errors.New("Trace missing")
)

var (
	metricTracesCreatedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "ingester_traces_created_total",
		Help:      "The total number of traces created per tenant.",
	}, []string{"tenant"})
	metricBytesWrittenTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "ingester_bytes_written_total",
		Help:      "The total bytes written per tenant.",
	}, []string{"tenant"})
	metricBlocksClearedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "ingester_blocks_cleared_total",
		Help:      "The total number of blocks cleared.",
	})
)

type instance struct {
	tracesMtx  sync.Mutex
	traces     map[uint32]*trace
	traceCount atomic.Int32

	blocksMtx        sync.RWMutex
	headBlock        *wal.AppendBlock
	completingBlocks []*wal.AppendBlock
	completeBlocks   []*wal.LocalBlock

	//headBlockSearch *searchData
	searchAppendBlocks map[*wal.AppendBlock]*searchData
	searchTagLookups   tempofb.SearchDataMap

	lastBlockCut time.Time

	instanceID         string
	tracesCreatedTotal prometheus.Counter
	bytesWrittenTotal  prometheus.Counter
	limiter            *Limiter
	writer             tempodb.Writer
	local              *local.Backend

	hash hash.Hash32
}

func newInstance(instanceID string, limiter *Limiter, writer tempodb.Writer, l *local.Backend) (*instance, error) {
	i := &instance{
		traces:             map[uint32]*trace{},
		searchAppendBlocks: map[*wal.AppendBlock]*searchData{},
		searchTagLookups:   tempofb.SearchDataMap{},

		instanceID:         instanceID,
		tracesCreatedTotal: metricTracesCreatedTotal.WithLabelValues(instanceID),
		bytesWrittenTotal:  metricBytesWrittenTotal.WithLabelValues(instanceID),
		limiter:            limiter,
		writer:             writer,
		local:              l,

		hash: fnv.New32(),
	}
	err := i.resetHeadBlock()
	if err != nil {
		return nil, err
	}
	return i, nil
}

// Push is used to push an entire tempopb.PushRequest. It is depecrecated and only required
// for older protocols.
func (i *instance) Push(ctx context.Context, req *tempopb.PushRequest) error {
	// check for max traces before grabbing the lock to better load shed
	err := i.limiter.AssertMaxTracesPerUser(i.instanceID, int(i.traceCount.Load()))
	if err != nil {
		return status.Errorf(codes.FailedPrecondition, "%s max live traces per tenant exceeded: %v", overrides.ErrorPrefixLiveTracesExceeded, err)
	}

	i.tracesMtx.Lock()
	defer i.tracesMtx.Unlock()

	id, err := pushRequestTraceID(req)
	if err != nil {
		return err
	}

	t := &tempopb.Trace{
		Batches: []*v1.ResourceSpans{req.Batch},
	}

	// traceBytes eventually end up back into the bytepool
	// allocating like this prevents panics by only putting slices into the bytepool
	// that were retrieved from there
	buffer := tempopb.SliceFromBytePool(t.Size())
	_, err = t.MarshalToSizedBuffer(buffer)
	if err != nil {
		return err
	}

	trace := i.getOrCreateTrace(id)
	return trace.Push(ctx, buffer, nil)
}

// PushBytes is used to push an unmarshalled tempopb.Trace to the instance
func (i *instance) PushBytes(ctx context.Context, id []byte, traceBytes []byte, searchData []byte) error {
	if !validation.ValidTraceID(id) {
		return status.Errorf(codes.InvalidArgument, "%s is not a valid traceid", hex.EncodeToString(id))
	}

	// check for max traces before grabbing the lock to better load shed
	err := i.limiter.AssertMaxTracesPerUser(i.instanceID, int(i.traceCount.Load()))
	if err != nil {
		return status.Errorf(codes.FailedPrecondition, "%s max live traces per tenant exceeded: %v", overrides.ErrorPrefixLiveTracesExceeded, err)
	}

	i.tracesMtx.Lock()
	defer i.tracesMtx.Unlock()

	i.RecordSearchLookupValues(searchData)

	trace := i.getOrCreateTrace(id)
	return trace.Push(ctx, traceBytes, searchData)
}

// Moves any complete traces out of the map to complete traces
func (i *instance) CutCompleteTraces(cutoff time.Duration, immediate bool) error {
	tracesToCut := i.tracesToCut(cutoff, immediate)

	for _, t := range tracesToCut {
		model.SortTraceBytes(t.traceBytes)

		out, err := proto.Marshal(t.traceBytes)
		if err != nil {
			return err
		}

		err = i.writeTraceToHeadBlock(t.traceID, out)
		if err != nil {
			return err
		}
		i.bytesWrittenTotal.Add(float64(len(out)))

		// return trace byte slices to be reused by proto marshalling
		//  WARNING: can't reuse traceid's b/c the appender takes ownership of byte slices that are passed to it
		tempopb.ReuseTraceBytes(t.traceBytes)

		err = i.searchAppendBlocks[i.headBlock].Append(context.TODO(), t)
		if err != nil {
			return err
		}
	}

	return nil
}

// CutBlockIfReady cuts a completingBlock from the HeadBlock if ready
// Returns a bool indicating if a block was cut along with the error (if any).
func (i *instance) CutBlockIfReady(maxBlockLifetime time.Duration, maxBlockBytes uint64, immediate bool) (uuid.UUID, error) {
	i.blocksMtx.Lock()
	defer i.blocksMtx.Unlock()

	if i.headBlock == nil || i.headBlock.DataLength() == 0 {
		return uuid.Nil, nil
	}

	now := time.Now()
	if i.lastBlockCut.Add(maxBlockLifetime).Before(now) || i.headBlock.DataLength() >= maxBlockBytes || immediate {
		completingBlock := i.headBlock

		i.completingBlocks = append(i.completingBlocks, completingBlock)

		err := i.resetHeadBlock()
		if err != nil {
			return uuid.Nil, fmt.Errorf("failed to resetHeadBlock: %w", err)
		}

		return completingBlock.BlockID(), nil
	}

	return uuid.Nil, nil
}

// CompleteBlock() moves a completingBlock to a completeBlock. The new completeBlock has the same ID
func (i *instance) CompleteBlock(blockID uuid.UUID) error {
	i.blocksMtx.Lock()

	var completingBlock *wal.AppendBlock
	for _, iterBlock := range i.completingBlocks {
		if iterBlock.BlockID() == blockID {
			completingBlock = iterBlock
			break
		}
	}
	i.blocksMtx.Unlock()

	if completingBlock == nil {
		return fmt.Errorf("error finding completingBlock")
	}

	ctx := context.Background()

	backendBlock, err := i.writer.CompleteBlockWithBackend(ctx, completingBlock, model.ObjectCombiner, i.local, i.local)
	if err != nil {
		return errors.Wrap(err, "error completing wal block with local backend")
	}

	ingesterBlock, err := wal.NewLocalBlock(ctx, backendBlock, i.local)
	if err != nil {
		return errors.Wrap(err, "error creating ingester block")
	}

	i.blocksMtx.Lock()
	i.completeBlocks = append(i.completeBlocks, ingesterBlock)
	i.blocksMtx.Unlock()

	return nil
}

// nolint:interfacer
func (i *instance) ClearCompletingBlock(blockID uuid.UUID) error {
	i.blocksMtx.Lock()
	var completingBlock *wal.AppendBlock
	for j, iterBlock := range i.completingBlocks {
		if iterBlock.BlockID() == blockID {
			completingBlock = iterBlock
			i.completingBlocks = append(i.completingBlocks[:j], i.completingBlocks[j+1:]...)
			break
		}
	}
	i.blocksMtx.Unlock()

	if completingBlock != nil {
		searchBlock := i.searchAppendBlocks[completingBlock]
		if searchBlock != nil {
			searchBlock.Clear()
			delete(i.searchAppendBlocks, completingBlock)
		}

		return completingBlock.Clear()
	}

	return fmt.Errorf("Error finding wal completingBlock to clear")
}

// GetBlockToBeFlushed gets a list of blocks that can be flushed to the backend
func (i *instance) GetBlockToBeFlushed(blockID uuid.UUID) *wal.LocalBlock {
	i.blocksMtx.Lock()
	defer i.blocksMtx.Unlock()

	for _, c := range i.completeBlocks {
		if c.BlockMeta().BlockID == blockID && c.FlushedTime().IsZero() {
			return c
		}
	}

	return nil
}

func (i *instance) ClearFlushedBlocks(completeBlockTimeout time.Duration) error {
	var err error

	i.blocksMtx.Lock()
	defer i.blocksMtx.Unlock()

	for idx, b := range i.completeBlocks {
		flushedTime := b.FlushedTime()
		if flushedTime.IsZero() {
			continue
		}

		if flushedTime.Add(completeBlockTimeout).Before(time.Now()) {
			i.completeBlocks = append(i.completeBlocks[:idx], i.completeBlocks[idx+1:]...)
			err = i.local.ClearBlock(b.BlockMeta().BlockID, i.instanceID)
			if err == nil {
				metricBlocksClearedTotal.Inc()
			}
			break
		}
	}

	return err
}

func (i *instance) FindTraceByID(ctx context.Context, id []byte) (*tempopb.Trace, error) {
	var err error
	var allBytes []byte

	// live traces
	i.tracesMtx.Lock()
	if liveTrace, ok := i.traces[i.tokenForTraceID(id)]; ok {
		allBytes, err = proto.Marshal(liveTrace.traceBytes)
		if err != nil {
			i.tracesMtx.Unlock()
			return nil, fmt.Errorf("unable to marshal liveTrace: %w", err)
		}
	}
	i.tracesMtx.Unlock()

	i.blocksMtx.Lock()
	defer i.blocksMtx.Unlock()

	// headBlock
	foundBytes, err := i.headBlock.Find(id, model.ObjectCombiner)
	if err != nil {
		return nil, fmt.Errorf("headBlock.Find failed: %w", err)
	}
	allBytes, _, err = model.CombineTraceBytes(allBytes, foundBytes, model.CurrentEncoding, i.headBlock.Meta().DataEncoding)
	if err != nil {
		return nil, fmt.Errorf("post headBlock combine failed: %w", err)
	}

	// completingBlock
	for _, c := range i.completingBlocks {
		foundBytes, err = c.Find(id, model.ObjectCombiner)
		if err != nil {
			return nil, fmt.Errorf("completingBlock.Find failed: %w", err)
		}
		allBytes, _, err = model.CombineTraceBytes(allBytes, foundBytes, model.CurrentEncoding, c.Meta().DataEncoding)
		if err != nil {
			return nil, fmt.Errorf("post completingBlocks combine failed: %w", err)
		}
	}

	// completeBlock
	for _, c := range i.completeBlocks {
		foundBytes, err = c.Find(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("completeBlock.Find failed: %w", err)
		}
		allBytes, _, err = model.CombineTraceBytes(allBytes, foundBytes, model.CurrentEncoding, c.BlockMeta().DataEncoding)
		if err != nil {
			return nil, fmt.Errorf("post completeBlocks combine failed: %w", err)
		}
	}

	// now marshal it all
	if allBytes != nil {
		out, err := model.Unmarshal(allBytes, model.CurrentEncoding)
		if err != nil {
			return nil, err
		}

		return out, nil
	}

	return nil, nil
}

// AddCompletingBlock adds an AppendBlock directly to the slice of completing blocks.
// This is used during wal replay. It is expected that calling code will add the appropriate
// jobs to the queue to eventually flush these.
func (i *instance) AddCompletingBlock(b *wal.AppendBlock) {
	i.blocksMtx.Lock()
	defer i.blocksMtx.Unlock()

	i.completingBlocks = append(i.completingBlocks, b)
}

// getOrCreateTrace will return a new trace object for the given request
//  It must be called under the i.tracesMtx lock
func (i *instance) getOrCreateTrace(traceID []byte) *trace {
	fp := i.tokenForTraceID(traceID)
	trace, ok := i.traces[fp]
	if ok {
		return trace
	}

	maxBytes := i.limiter.limits.MaxBytesPerTrace(i.instanceID)
	trace = newTrace(maxBytes, traceID)
	i.traces[fp] = trace
	i.tracesCreatedTotal.Inc()
	i.traceCount.Inc()

	return trace
}

// tokenForTraceID hash trace ID, should be called under lock
func (i *instance) tokenForTraceID(id []byte) uint32 {
	i.hash.Reset()
	_, _ = i.hash.Write(id)
	return i.hash.Sum32()
}

// resetHeadBlock() should be called under lock
func (i *instance) resetHeadBlock() error {
	var err error
	i.headBlock, err = i.writer.WAL().NewBlock(uuid.New(), i.instanceID, model.CurrentEncoding)
	if err != nil {
		return err
	}

	i.lastBlockCut = time.Now()

	i.searchAppendBlocks[i.headBlock], err = NewSearchDataForAppendBlock(i, i.headBlock)
	return err
}

func (i *instance) tracesToCut(cutoff time.Duration, immediate bool) []*trace {
	i.tracesMtx.Lock()
	defer i.tracesMtx.Unlock()

	cutoffTime := time.Now().Add(cutoff)
	tracesToCut := make([]*trace, 0, len(i.traces))

	for key, trace := range i.traces {
		if cutoffTime.After(trace.lastAppend) || immediate {
			tracesToCut = append(tracesToCut, trace)
			delete(i.traces, key)
		}
	}
	i.traceCount.Store(int32(len(i.traces)))

	return tracesToCut
}

func (i *instance) writeTraceToHeadBlock(id common.ID, b []byte) error {
	i.blocksMtx.Lock()
	defer i.blocksMtx.Unlock()

	return i.headBlock.Write(id, b)
}

// pushRequestTraceID gets the TraceID of the first span in the batch and assumes its the trace ID throughout
//  this assumption should hold b/c the distributors make sure each batch all belong to the same trace
func pushRequestTraceID(req *tempopb.PushRequest) ([]byte, error) {
	if req == nil || req.Batch == nil {
		return nil, errors.New("req or req.Batch nil")
	}

	if len(req.Batch.InstrumentationLibrarySpans) == 0 {
		return nil, errors.New("InstrumentationLibrarySpans has length 0")
	}

	if len(req.Batch.InstrumentationLibrarySpans[0].Spans) == 0 {
		return nil, errors.New("Spans has length 0")
	}

	return req.Batch.InstrumentationLibrarySpans[0].Spans[0].TraceId, nil
}

func (i *instance) rediscoverLocalBlocks(ctx context.Context) error {
	ids, err := i.local.Blocks(ctx, i.instanceID)
	if err != nil {
		return err
	}

	for _, id := range ids {
		meta, err := i.local.BlockMeta(ctx, id, i.instanceID)
		if err != nil {
			if err == backend.ErrMetaDoesNotExist {
				// Partial/incomplete block found, remove, it will be recreated from data in the wal.
				level.Warn(log.Logger).Log("msg", "Unable to reload meta for local block. This indicates an incomplete block and will be deleted", "tenant", i.instanceID, "block", id.String())
				err = i.local.ClearBlock(id, i.instanceID)
				if err != nil {
					return errors.Wrapf(err, "deleting bad local block tenant %v block %v", i.instanceID, id.String())
				}
				continue
			}

			return err
		}

		b, err := encoding.NewBackendBlock(meta, i.local)
		if err != nil {
			return err
		}

		ib, err := wal.NewLocalBlock(ctx, b, i.local)
		if err != nil {
			return err
		}

		i.blocksMtx.Lock()
		i.completeBlocks = append(i.completeBlocks, ib)
		i.blocksMtx.Unlock()

		level.Info(log.Logger).Log("msg", "reloaded local block", "tenantID", i.instanceID, "block", id.String(), "flushed", ib.FlushedTime())
	}

	return nil
}

func (i *instance) Search(ctx context.Context, req *tempopb.SearchRequest) ([]*tempopb.TraceSearchMetadata, error) {

	var results []*tempopb.TraceSearchMetadata

	p := NewSearchPipeline(req)

	// Search live traces
	func() {
		i.tracesMtx.Lock()
		defer i.tracesMtx.Unlock()

		for _, t := range i.traces {

			for _, s := range t.searchData {
				searchData := tempofb.SearchDataFromBytes(s)
				if p.Matches(searchData) {
					results = append(results, &tempopb.TraceSearchMetadata{
						TraceID:           t.traceID,
						RootServiceName:   tempofb.SearchDataGet(searchData, "root.service.name"),
						RootTraceName:     tempofb.SearchDataGet(searchData, "root.name"),
						StartTimeUnixNano: searchData.StartTimeUnixNano(),
						DurationMs:        uint32((searchData.EndTimeUnixNano() - searchData.StartTimeUnixNano()) / 1_000_000),
					})
					continue
				}
			}
		}

		fmt.Println("Found", len(results), "matches in live traces")
	}()

	// Search append blocks
	err := func() error {
		i.blocksMtx.Lock()
		defer i.blocksMtx.Unlock()

		for b, s := range i.searchAppendBlocks {
			headResults, err := s.Search(ctx, p)
			if err != nil {
				return err
			}

			if len(headResults) > 0 {
				results = append(results, headResults...)
				fmt.Println("Found", len(headResults), "matches in block", b.BlockID().String())
			}
		}

		return nil
	}()
	if err != nil {
		return nil, errors.Wrap(err, "searching head block")
	}

	return results, err
}

func (i *instance) GetSearchTags() []string {
	tags := make([]string, 0, len(i.searchTagLookups))
	for k := range i.searchTagLookups {
		tags = append(tags, k)
	}
	return tags
}

func (i *instance) GetSearchTagValues(tagName string) []string {
	return i.searchTagLookups[tagName]
}

var recordableSearchLookupTags map[string]struct{} = map[string]struct{}{
	"root.name":         {},
	"root.service.name": {},
}

func (i *instance) RecordSearchLookupValues(b []byte) {
	kv := &tempofb.KeyValues{}

	s := tempofb.SearchDataFromBytes(b)
	for j := 0; j < s.TagsLength(); j++ {
		s.Tags(kv, j)
		key := string(kv.Key())
		if _, ok := recordableSearchLookupTags[key]; ok {
			for k := 0; k < kv.ValueLength(); k++ {
				tempofb.SearchDataAppend(i.searchTagLookups, key, string(kv.Value(k)))
			}
		}
	}
}
