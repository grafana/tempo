package friggdb

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	bloom "github.com/dgraph-io/ristretto/z"
	"github.com/dgryski/go-farm"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/grafana/frigg/friggdb/backend"
	"github.com/grafana/frigg/friggdb/backend/cache"
	"github.com/grafana/frigg/friggdb/backend/gcs"
	"github.com/grafana/frigg/friggdb/backend/local"
	"github.com/grafana/frigg/friggdb/encoding"
	"github.com/grafana/frigg/friggdb/pool"
	"github.com/grafana/frigg/friggdb/wal"
)

var (
	metricBlocklistPoll = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "friggdb",
		Name:      "blocklist_poll_total",
		Help:      "Total number of times blocklist polling has occurred.",
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
)

type Writer interface {
	WriteBlock(ctx context.Context, block wal.CompleteBlock) error
	WAL() (wal.WAL, error)
}

type Reader interface {
	Find(tenantID string, id encoding.ID) ([]byte, EstimatedMetrics, error)
	Shutdown()
}

type EstimatedMetrics struct {
	BloomFilterReads     int
	BloomFilterBytesRead int
	IndexReads           int
	IndexBytesRead       int
	BlockReads           int
	BlockBytesRead       int
}

type readerWriter struct {
	r backend.Reader
	w backend.Writer
	c backend.Compactor

	pool *pool.Pool

	logger              log.Logger
	cfg                 *Config
	blockLists          map[string][]encoding.SearchableBlockMeta
	compactedBlockLists map[string][]encoding.SearchableBlockMeta
	blockListsMtx       sync.Mutex
}

func New(cfg *Config, logger log.Logger) (Reader, Writer, error) {
	var err error
	var r backend.Reader
	var w backend.Writer
	var c backend.Compactor

	switch cfg.Backend {
	case "local":
		r, w, c, err = local.New(cfg.Local)
	case "gcs":
		r, w, c, err = gcs.New(cfg.GCS)
	default:
		err = fmt.Errorf("unknown local %s", cfg.Backend)
	}

	if err != nil {
		return nil, nil, err
	}

	if cfg.Cache != nil {
		r, err = cache.New(r, cfg.Cache, logger)

		if err != nil {
			return nil, nil, err
		}
	}

	rw := &readerWriter{
		r:                   r,
		w:                   w,
		c:                   c,
		cfg:                 cfg,
		logger:              logger,
		pool:                pool.NewPool(cfg.Pool),
		blockLists:          make(map[string][]encoding.SearchableBlockMeta),
		compactedBlockLists: make(map[string][]encoding.SearchableBlockMeta),
	}

	go rw.pollBlocklist()

	return rw, rw, nil
}

func (rw *readerWriter) WriteBlock(ctx context.Context, c wal.CompleteBlock) error {
	uuid, tenantID, records, blockFilePath := c.WriteInfo()
	indexBytes, err := encoding.MarshalRecords(records)
	if err != nil {
		return err
	}

	metaBytes, err := json.Marshal(c.BlockMeta())
	if err != nil {
		return err
	}

	bloomBytes := c.BloomFilter().JSONMarshal()

	err = rw.w.Write(ctx, uuid, tenantID, metaBytes, bloomBytes, indexBytes, blockFilePath)
	if err != nil {
		return err
	}

	c.BlockWroteSuccessfully(time.Now())

	return nil
}

func (rw *readerWriter) WAL() (wal.WAL, error) {
	return wal.New(rw.cfg.WAL)
}

func (rw *readerWriter) Find(tenantID string, id encoding.ID) ([]byte, EstimatedMetrics, error) {
	metrics := EstimatedMetrics{} // we are purposefully not locking when updating this struct.  that's why they are "estimated"

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
		meta := payload.(encoding.SearchableBlockMeta)

		bloomBytes, err := rw.r.Bloom(meta.BlockID, tenantID)
		if err != nil {
			return nil, err
		}

		filter := bloom.JSONUnmarshal(bloomBytes)
		metrics.BloomFilterReads++
		metrics.BloomFilterBytesRead += len(bloomBytes)
		if !filter.Has(farm.Fingerprint64(id)) {
			return nil, nil
		}

		indexBytes, err := rw.r.Index(meta.BlockID, tenantID)
		metrics.IndexReads++
		metrics.IndexBytesRead += len(indexBytes)
		if err != nil {
			return nil, err
		}

		record, err := encoding.FindRecord(id, indexBytes)
		if err != nil {
			return nil, err
		}

		if record == nil {
			return nil, nil
		}

		objectBytes, err := rw.r.Object(meta.BlockID, tenantID, record.Start, record.Length)
		metrics.BlockReads++
		metrics.BlockBytesRead += len(objectBytes)
		if err != nil {
			return nil, err
		}

		var foundObject []byte
		err = encoding.IterateObjects(bytes.NewReader(objectBytes), func(iterID encoding.ID, iterObject []byte) (bool, error) {
			if bytes.Equal(iterID, id) {
				foundObject = iterObject
				return false, nil
			}

			return true, nil

		})
		if err != nil {
			return nil, err
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

func (rw *readerWriter) pollBlocklist() {
	if rw.cfg.BlocklistRefreshRate == 0 {
		level.Info(rw.logger).Log("msg", "blocklist Refresh Rate unset.  friggdb querying effectively disabled.")
		return
	}

	rw.actuallyPollBlocklist()

	ticker := time.NewTicker(rw.cfg.BlocklistRefreshRate)
	for range ticker.C {
		start := time.Now()
		rw.actuallyPollBlocklist()
		metricBlocklistPollDuration.Observe(time.Since(start).Seconds())
	}
}

func (rw *readerWriter) actuallyPollBlocklist() {
	metricBlocklistPoll.Inc()

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
		blocklist := make([]encoding.SearchableBlockMeta, 0, len(blockIDs))
		compactedBlocklist := make([]encoding.SearchableBlockMeta, 0, len(blockIDs)) // jpe: this is dumb. put both kinds of block metas in the same list?
		_, err = rw.pool.RunJobs(interfaceSlice, func(payload interface{}) ([]byte, error) {
			blockID := payload.(uuid.UUID)
			meta := &encoding.SearchableBlockMeta{}
			isCompacted := false

			metaBytes, err := rw.r.BlockMeta(blockID, tenantID)
			// if the normal meta doesn't exist maybe it's compacted.
			if os.IsNotExist(err) {
				metaBytes, err = rw.c.CompactedBlockMeta(blockID, tenantID)
				isCompacted = true
			}

			if err != nil {
				metricBlocklistErrors.WithLabelValues(tenantID).Inc()
				level.Error(rw.logger).Log("msg", "failed to retrieve block meta", "tenantID", tenantID, "blockID", blockID, "err", err)
				return nil, nil
			}

			err = json.Unmarshal(metaBytes, meta)
			if err != nil {
				metricBlocklistErrors.WithLabelValues(tenantID).Inc()
				level.Error(rw.logger).Log("msg", "failed to unmarshal json blocklist", "tenantID", tenantID, "blockID", blockID, "err", err)
				return nil, nil
			}

			// todo:  make this not terrible. this mutex is dumb we should be returning results with a channel. shoehorning this into the worker pool is silly.
			//        make the worker pool more generic? and reusable in this case
			listMutex.Lock()
			if isCompacted {
				compactedBlocklist = append(compactedBlocklist, *meta)
			} else {
				blocklist = append(blocklist, *meta)
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

		rw.blockListsMtx.Lock()
		rw.blockLists[tenantID] = blocklist
		rw.compactedBlockLists[tenantID] = compactedBlocklist
		rw.blockListsMtx.Unlock()
	}
}
