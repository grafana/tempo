package friggdb

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"sync"
	"time"

	bloom "github.com/dgraph-io/ristretto/z"
	"github.com/dgryski/go-farm"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/atomic"

	"github.com/grafana/frigg/friggdb/backend"
	"github.com/grafana/frigg/friggdb/backend/cache"
	"github.com/grafana/frigg/friggdb/backend/gcs"
	"github.com/grafana/frigg/friggdb/backend/local"
	"github.com/grafana/frigg/friggdb/pool"
	"github.com/grafana/frigg/friggdb/wal"
	"github.com/grafana/frigg/pkg/util"
)

var (
	metricBlockListPollTotal = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "friggdb",
		Name:      "blocklist_poll_count_total",
		Help:      "Total number of times the blocklist poll has occurred.",
	})
	metricBlocklistErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "friggdb",
		Name:      "blocklist_poll_errors_total",
		Help:      "Total number of times an error occurred while polling the blocklist.",
	}, []string{"tenant"})
	metricBlocklistPollDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: "friggdb",
		Name:      "blocklist_poll_duration_seconds",
		Help:      "Records the amount of time to poll and update the blocklist.",
		Buckets:   prometheus.ExponentialBuckets(.25, 2, 6),
	})
	metricBlocklistLength = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "friggdb",
		Name:      "blocklist_length",
		Help:      "Total number of blocks per tenant.",
	}, []string{"tenant"})
	metricCompactedBlocklistLength = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "friggdb",
		Name:      "compaction_blocklist_length",
		Help:      "Total number of compacted blocks per tenant.",
	}, []string{"tenant"})
)

const (
	inputBlocks  = 4
	outputBlocks = 2

	cursorDone = -1
)

type BlockStore interface {
	// Writer interface
	Write(ctx context.Context, blockID uuid.UUID, tenantID string, meta *backend.BlockMeta, bBloom []byte, bIndex []byte, objectFilePath string) error
	WriteBlock(ctx context.Context, block wal.CompleteBlock) error
	WAL() wal.WAL

	// Reader interface
	Find(tenantID string, id backend.ID) ([]byte, FindMetrics, error)
	Shutdown()

	// BlockStore utilities
	Backend() string
	BlocklistTenants() []interface{}
	Blocklist(tenantID string) []*backend.BlockMeta
	CompactedBlocklist(tenantID string) []*backend.CompactedBlockMeta
	BlocksToCompact(tenantID string, cursor int, maxCompactionRange time.Duration) ([]*backend.BlockMeta, int) // Technically BlocksInGivenTimeRange
	GetBackendReader() backend.Reader
}

type FindMetrics struct {
	BloomFilterReads     *atomic.Int32
	BloomFilterBytesRead *atomic.Int32
	IndexReads           *atomic.Int32
	IndexBytesRead       *atomic.Int32
	BlockReads           *atomic.Int32
	BlockBytesRead       *atomic.Int32
}

type readerWriter struct {
	r backend.Reader
	w backend.Writer

	wal  wal.WAL
	pool *pool.Pool

	logger              log.Logger
	cfg                 *Config
	blockLists          map[string][]*backend.BlockMeta
	blockListsMtx       sync.Mutex
	compactedBlockLists map[string][]*backend.CompactedBlockMeta
}

func New(cfg *Config, logger log.Logger) (BlockStore, error) {
	var err error
	var r backend.Reader
	var w backend.Writer

	switch cfg.Backend {
	case "local":
		r, w, err = local.New(cfg.Local)
	case "gcs":
		r, w, err = gcs.New(cfg.GCS)
	default:
		err = fmt.Errorf("unknown local %s", cfg.Backend)
	}

	if err != nil {
		return nil, err
	}

	if cfg.Cache != nil {
		r, err = cache.New(r, cfg.Cache, logger)

		if err != nil {
			return nil, err
		}
	}

	rw := &readerWriter{
		compactedBlockLists: make(map[string][]*backend.CompactedBlockMeta),
		r:                   r,
		w:                   w,
		cfg:                 cfg,
		logger:              logger,
		pool:                pool.NewPool(cfg.Pool),
		blockLists:          make(map[string][]*backend.BlockMeta),
	}

	rw.wal, err = wal.New(rw.cfg.WAL)
	if err != nil {
		return nil, err
	}

	go rw.runBlockListPollLoop()

	return rw, nil
}

func (rw *readerWriter) Backend() string {
	return rw.cfg.Backend
}

func (rw *readerWriter) runBlockListPollLoop() {
	metricBlockListPollTotal.Inc()
	if rw.cfg.BlockListPollDuration == 0 {
		level.Warn(rw.logger).Log("msg", "blocklist Refresh Rate unset.  friggdb querying, compaction and retention effectively disabled.")
		return
	}

	rw.pollBlocklist()

	ticker := time.NewTicker(rw.cfg.BlockListPollDuration)
	for range ticker.C {
		rw.pollBlocklist()
	}
}

func (rw *readerWriter) Blocklist(tenantID string) []*backend.BlockMeta {
	rw.blockListsMtx.Lock()
	defer rw.blockListsMtx.Unlock()

	return rw.blockLists[tenantID]
}

// todo:  make separate compacted list mutex?
func (rw *readerWriter) CompactedBlocklist(tenantID string) []*backend.CompactedBlockMeta {
	rw.blockListsMtx.Lock()
	defer rw.blockListsMtx.Unlock()

	return rw.compactedBlockLists[tenantID]
}

func (rw *readerWriter) BlocklistTenants() []interface{} {
	rw.blockListsMtx.Lock()
	defer rw.blockListsMtx.Unlock()

	tenants := make([]interface{}, 0, len(rw.blockLists))
	for tenant := range rw.blockLists {
		tenants = append(tenants, tenant)
	}

	return tenants
}

func (rw *readerWriter) CompactedBlockMeta(blockID uuid.UUID, tenantID string) (*backend.CompactedBlockMeta, error) {
	filename := util.CompactedMetaFileName(blockID, tenantID)

	fi, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return nil, backend.ErrMetaDoesNotExist
	}
	if err != nil {
		return nil, err
	}

	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	out := &backend.CompactedBlockMeta{}
	err = json.Unmarshal(bytes, out)
	if err != nil {
		return nil, err
	}
	out.CompactedTime = fi.ModTime()

	return out, err
}

func (rw *readerWriter) Write(ctx context.Context, blockID uuid.UUID, tenantID string, meta *backend.BlockMeta, bBloom []byte, bIndex []byte, objectFilePath string) error {
	return rw.Write(ctx, blockID, tenantID, meta, bBloom, bIndex, objectFilePath)
}

func (rw *readerWriter) GetBackendReader() backend.Reader {
	return rw.r
}

func (rw *readerWriter) pollBlocklist() {
	start := time.Now()
	defer func() { metricBlocklistPollDuration.Observe(time.Since(start).Seconds()) }()

	tenants, err := rw.r.Tenants()
	if err != nil {
		metricBlocklistErrors.WithLabelValues("").Inc()
		level.Error(rw.logger).Log("msg", "error retrieving tenants while polling blocklist", "err", err)
	}

	for _, tenantID := range tenants {
		blockIDs, err := rw.r.Blocks(tenantID)
		if err != nil {
			metricBlocklistErrors.WithLabelValues(tenantID).Inc()
			level.Error(rw.logger).Log("msg", "error polling blocklist", "tenantID", tenantID, "err", err)
		}

		interfaceSlice := make([]interface{}, 0, len(blockIDs))
		for _, id := range blockIDs {
			interfaceSlice = append(interfaceSlice, id)
		}

		listMutex := sync.Mutex{}
		blocklist := make([]*backend.BlockMeta, 0, len(blockIDs))
		compactedBlocklist := make([]*backend.CompactedBlockMeta, 0, len(blockIDs))
		_, err = rw.pool.RunJobs(interfaceSlice, func(payload interface{}) ([]byte, error) {
			blockID := payload.(uuid.UUID)

			var compactedBlockMeta *backend.CompactedBlockMeta
			blockMeta, err := rw.r.BlockMeta(blockID, tenantID)
			// if the normal meta doesn't exist maybe it's compacted.
			if err == backend.ErrMetaDoesNotExist {
				blockMeta = nil
				compactedBlockMeta, err = rw.CompactedBlockMeta(blockID, tenantID)
			}

			if err != nil {
				metricBlocklistErrors.WithLabelValues(tenantID).Inc()
				level.Error(rw.logger).Log("msg", "failed to retrieve block meta", "tenantID", tenantID, "blockID", blockID, "err", err)
				return nil, nil
			}

			// todo:  make this not terrible. this mutex is dumb we should be returning results with a channel. shoehorning this into the worker pool is silly.
			//        make the worker pool more generic? and reusable in this case
			listMutex.Lock()
			if blockMeta != nil {
				blocklist = append(blocklist, blockMeta)

			} else if compactedBlockMeta != nil {
				compactedBlocklist = append(compactedBlocklist, compactedBlockMeta)
			}
			listMutex.Unlock()

			return nil, nil
		})

		if err != nil {
			metricBlocklistErrors.WithLabelValues(tenantID).Inc()
			level.Error(rw.logger).Log("msg", "run blocklist jobs", "tenantID", tenantID, "err", err)
			continue
		}

		metricBlocklistLength.WithLabelValues(tenantID).Set(float64(len(blocklist)))
		metricCompactedBlocklistLength.WithLabelValues(tenantID).Set(float64(len(compactedBlocklist)))

		sort.Slice(blocklist, func(i, j int) bool {
			return blocklist[i].StartTime.Before(blocklist[j].StartTime)
		})
		sort.Slice(compactedBlocklist, func(i, j int) bool {
			return compactedBlocklist[i].StartTime.Before(compactedBlocklist[j].StartTime)
		})

		rw.blockListsMtx.Lock()
		rw.blockLists[tenantID] = blocklist
		rw.compactedBlockLists[tenantID] = compactedBlocklist
		rw.blockListsMtx.Unlock()
	}
}

// todo: metric to determine "effectiveness" of compaction.  i.e. total key overlap of blocks that is being eliminated?
//       switch to iterator pattern?
func (rw *readerWriter) BlocksToCompact(tenantID string, cursor int, maxCompactionRange time.Duration) ([]*backend.BlockMeta, int) {
	// loop through blocks starting at cursor for the given tenant, blocks are sorted by start date so candidates for compaction should be near each other
	//   - consider candidateBlocks at a time.
	//   - find the blocks with the fewest records that are within the compaction range
	rw.blockListsMtx.Lock() // todo: there's lots of contention on this mutex.  keep an eye on this
	defer rw.blockListsMtx.Unlock()

	blocklist := rw.blockLists[tenantID]
	if inputBlocks > len(blocklist) {
		return nil, cursorDone
	}

	if cursor < 0 {
		return nil, cursorDone
	}

	cursorEnd := cursor + inputBlocks
	for {
		if cursorEnd >= len(blocklist) {
			break
		}

		blockStart := blocklist[cursor]
		blockEnd := blocklist[cursorEnd]

		if blockEnd.EndTime.Sub(blockStart.StartTime) < maxCompactionRange {
			return blocklist[cursor:cursorEnd], cursorEnd + 1
		}

		cursor++
		cursorEnd = cursor + inputBlocks
	}

	return nil, cursorDone
}

func (rw *readerWriter) WriteBlock(ctx context.Context, c wal.CompleteBlock) error {
	uuid, tenantID, records, blockFilePath := c.WriteInfo()
	indexBytes, err := backend.MarshalRecords(records)
	if err != nil {
		return err
	}

	bloomBytes := c.BloomFilter().JSONMarshal()

	err = rw.w.Write(ctx, uuid, tenantID, c.BlockMeta(), bloomBytes, indexBytes, blockFilePath)
	if err != nil {
		return err
	}

	c.BlockWroteSuccessfully(time.Now())

	return nil
}

func (rw *readerWriter) WAL() wal.WAL {
	return rw.wal
}

func (rw *readerWriter) Find(tenantID string, id backend.ID) ([]byte, FindMetrics, error) {
	metrics := FindMetrics{
		BloomFilterReads:     atomic.NewInt32(0),
		BloomFilterBytesRead: atomic.NewInt32(0),
		IndexReads:           atomic.NewInt32(0),
		IndexBytesRead:       atomic.NewInt32(0),
		BlockReads:           atomic.NewInt32(0),
		BlockBytesRead:       atomic.NewInt32(0),
	}

	rw.blockListsMtx.Lock()
	blocklist, found := rw.blockLists[tenantID]
	copiedBlocklist := make([]interface{}, 0, len(blocklist))
	for _, b := range blocklist {
		// if in range copy
		if bytes.Compare(id, b.MinID) != -1 && bytes.Compare(id, b.MaxID) != 1 {
			copiedBlocklist = append(copiedBlocklist, b)
		}
	}
	rw.blockListsMtx.Unlock()

	if !found {
		return nil, metrics, fmt.Errorf("tenantID %s not found", tenantID)
	}

	foundBytes, err := rw.pool.RunJobs(copiedBlocklist, func(payload interface{}) ([]byte, error) {
		meta := payload.(*backend.BlockMeta)

		bloomBytes, err := rw.r.Bloom(meta.BlockID, tenantID)
		if err != nil {
			return nil, err
		}

		filter := bloom.JSONUnmarshal(bloomBytes)
		metrics.BloomFilterReads.Inc()
		metrics.BloomFilterBytesRead.Add(int32(len(bloomBytes)))
		if !filter.Has(farm.Fingerprint64(id)) {
			return nil, nil
		}

		indexBytes, err := rw.r.Index(meta.BlockID, tenantID)
		metrics.IndexReads.Inc()
		metrics.IndexBytesRead.Add(int32(len(indexBytes)))
		if err != nil {
			return nil, err
		}

		record, err := backend.FindRecord(id, indexBytes)
		if err != nil {
			return nil, err
		}

		if record == nil {
			return nil, nil
		}

		objectBytes := make([]byte, record.Length)
		err = rw.r.Object(meta.BlockID, tenantID, record.Start, objectBytes)
		metrics.BlockReads.Inc()
		metrics.BlockBytesRead.Add(int32(len(objectBytes)))
		if err != nil {
			return nil, err
		}

		iter := backend.NewIterator(bytes.NewReader(objectBytes))
		var foundObject []byte
		for {
			iterID, iterObject, err := iter.Next()
			if iterID == nil {
				break
			}
			if err != nil {
				return nil, err
			}
			if bytes.Equal(iterID, id) {
				foundObject = iterObject
				break
			}
		}
		return foundObject, nil
	})

	return foundBytes, metrics, err
}

func (rw *readerWriter) Shutdown() {
	// todo: stop blocklist poll
	rw.pool.Shutdown()
	rw.r.Shutdown()
}
