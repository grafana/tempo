package friggdb

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	bloom "github.com/dgraph-io/ristretto/z"
	"github.com/dgryski/go-farm"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/golang/protobuf/proto"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/grafana/frigg/friggdb/backend"
	"github.com/grafana/frigg/friggdb/backend/gcs"
	"github.com/grafana/frigg/friggdb/backend/local"
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
)

type Writer interface {
	WriteBlock(ctx context.Context, block CompleteBlock) error
	WAL() (WAL, error)
}

type Reader interface {
	Find(tenantID string, id ID, out proto.Message) (FindMetrics, bool, error)
}

type FindMetrics struct {
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

	logger        log.Logger
	cfg           *Config
	blockLists    map[string][]searchableBlockMeta
	blockListsMtx sync.Mutex
}

func New(cfg *Config, logger log.Logger) (Reader, Writer, error) {
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
		return nil, nil, err
	}

	if cfg.BloomFilterFalsePositive <= 0.0 {
		return nil, nil, fmt.Errorf("invalid bloom filter fp rate %v", cfg.BloomFilterFalsePositive)
	}

	rw := &readerWriter{
		r:          r,
		w:          w,
		cfg:        cfg,
		logger:     logger,
		blockLists: make(map[string][]searchableBlockMeta),
	}

	go rw.pollBlocklist()

	return rw, rw, nil
}

func (rw *readerWriter) WriteBlock(ctx context.Context, c CompleteBlock) error {
	uuid, tenantID, records, blockFilePath := c.writeInfo()
	indexBytes, err := marshalRecords(records)
	if err != nil {
		return err
	}

	metaBytes, err := json.Marshal(c.blockMeta())
	if err != nil {
		return err
	}

	bloomBytes := c.bloomFilter().JSONMarshal()

	err = rw.w.Write(ctx, uuid, tenantID, metaBytes, bloomBytes, indexBytes, blockFilePath)
	if err != nil {
		return err
	}

	c.blockWroteSuccessfully(time.Now())

	return nil
}

func (rw *readerWriter) WAL() (WAL, error) {
	return newWAL(&walConfig{
		filepath:        rw.cfg.WALFilepath,
		indexDownsample: rw.cfg.IndexDownsample,
		bloomFP:         rw.cfg.BloomFilterFalsePositive,
	})
}

func (rw *readerWriter) Find(tenantID string, id ID, out proto.Message) (FindMetrics, bool, error) {
	metrics := FindMetrics{}

	rw.blockListsMtx.Lock()
	blocklist, found := rw.blockLists[tenantID]
	/*copiedBlocklist := make([]searchableBlockMeta, 0, len(blocklist))
	copy(copiedBlocklist, blocklist)*/

	copiedBlocklist := append(make([]searchableBlockMeta, 0, len(blocklist)), blocklist...)
	rw.blockListsMtx.Unlock()

	if !found {
		return metrics, false, fmt.Errorf("tenantID %s not found", tenantID)
	}

	for _, meta := range copiedBlocklist {
		if bytes.Compare(id, meta.MinID) == -1 || bytes.Compare(id, meta.MaxID) == 1 {
			continue
		}

		bloomBytes, err := rw.r.Bloom(meta.BlockID, tenantID)
		if err != nil {
			return metrics, false, err
		}

		filter := bloom.JSONUnmarshal(bloomBytes)
		metrics.BloomFilterReads++
		metrics.BloomFilterBytesRead += len(bloomBytes)
		if !filter.Has(farm.Fingerprint64(id)) {
			continue
		}

		indexBytes, err := rw.r.Index(meta.BlockID, tenantID)
		metrics.IndexReads++
		metrics.IndexBytesRead += len(indexBytes)
		if err != nil {
			return metrics, false, err
		}

		record, err := findRecord(id, indexBytes)
		if err != nil {
			return metrics, false, err
		}

		if record == nil {
			continue
		}

		objectBytes, err := rw.r.Object(meta.BlockID, tenantID, record.Start, record.Length)
		metrics.BlockReads++
		metrics.BlockBytesRead += len(objectBytes)
		if err != nil {
			return metrics, false, err
		}

		found := false
		err = iterateObjects(bytes.NewReader(objectBytes), out, func(foundID ID, msg proto.Message) (bool, error) {
			if bytes.Equal(foundID, id) {
				found = true
				return false, nil
			}

			return true, nil

		})
		if err != nil {
			return metrics, false, err
		}

		if found {
			return metrics, true, nil
		}
	}

	return metrics, false, nil
}

func (rw *readerWriter) pollBlocklist() {
	if rw.cfg.BlocklistRefreshRate == 0 {
		level.Info(rw.logger).Log("msg", "blocklist Refresh Rate unset.  friggdb querying effectively disabled.")
		return
	}

	rw.actuallyPollBlocklist()

	ticker := time.NewTicker(rw.cfg.BlocklistRefreshRate)
	for range ticker.C {
		rw.actuallyPollBlocklist()
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
		blocklistsJSON, err := rw.r.Blocklist(tenantID)
		if err != nil {
			metricBlocklistErrors.WithLabelValues(tenantID).Inc()
			level.Error(rw.logger).Log("msg", "error polling blocklist", "tenantID", tenantID, "err", err)
		}

		meta := &searchableBlockMeta{}
		blocklist := make([]searchableBlockMeta, 0, len(blocklistsJSON))
		for _, j := range blocklistsJSON {
			err = json.Unmarshal(j, meta)
			if err != nil {
				metricBlocklistErrors.WithLabelValues(tenantID).Inc()
				level.Error(rw.logger).Log("msg", "failed to unmarshal json blocklist", "tenantID", tenantID, "err", err)
				continue
			}

			blocklist = append(blocklist, *meta)
		}
		rw.blockListsMtx.Lock()
		rw.blockLists[tenantID] = blocklist
		rw.blockListsMtx.Unlock()
	}
}
