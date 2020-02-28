package compactor

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/google/uuid"
	"github.com/grafana/frigg/friggdb"
	"github.com/grafana/frigg/friggdb/backend"
	"github.com/grafana/frigg/friggdb/backend/gcs"
	"github.com/grafana/frigg/friggdb/backend/local"
	"github.com/grafana/frigg/friggdb/pool"
	"github.com/grafana/frigg/pkg/storage"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	metricMaintenanceTotal = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "friggdb",
		Name:      "maintenance_total",
		Help:      "Total number of times the maintenance cycle has occurred.",
	})
	metricCompactionDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: "friggdb",
		Name:      "compaction_duration_seconds",
		Help:      "Records the amount of time to compact a set of blocks.",
		Buckets:   prometheus.ExponentialBuckets(1, 2, 10),
	})
	metricCompactionStopDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: "friggdb",
		Name:      "compaction_duration_stop_seconds",
		Help:      "Records the amount of time waiting on compaction jobs to stop.",
		Buckets:   prometheus.ExponentialBuckets(1, 2, 10),
	})
	metricCompactionErrors = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "friggdb",
		Name:      "compaction_errors_total",
		Help:      "Total number of errors occurring during compaction.",
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
	metricRetentionDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: "friggdb",
		Name:      "retention_duration_seconds",
		Help:      "Records the amount of time to perform retention tasks.",
		Buckets:   prometheus.ExponentialBuckets(.25, 2, 6),
	})
	metricRetentionErrors = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "friggdb",
		Name:      "retention_errors_total",
		Help:      "Total number of times an error occurred while performing retention tasks.",
	})
	metricMarkedForDeletion = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "friggdb",
		Name:      "retention_marked_for_deletion_total",
		Help:      "Total number of blocks marked for deletion.",
	})
	metricDeleted = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "friggdb",
		Name:      "retention_deleted_total",
		Help:      "Total number of blocks deleted.",
	})
)

const (
	outputBlocks = 2
	cursorDone   = -1
)

type Compactor struct {
	cfg        *Config
	bCfg       friggdb.Config
	b          backend.Compactor
	store      storage.Store
	pool       *pool.Pool
	jobStopper *pool.Stopper
	logger     log.Logger
}

// New makes a new Compactor.
func New(cfg Config, store storage.Store, logger log.Logger) (*Compactor, error) {
	var err error
	var b backend.Compactor

	s := store.GetBackendConfig()
	switch s.Backend {
	case "local":
		b, err = local.NewCompactor(s.Local)
	case "gcs":
		b, err = gcs.NewCompactor(s.GCS)
	default:
		err = fmt.Errorf("unknown local %s", s.Backend)
	}

	if err != nil {
		return nil, err
	}

	c := &Compactor{
		cfg:    &cfg,
		bCfg:   s,
		b:      b,
		store:  store,
		pool:   pool.NewPool(s.Pool),
		logger: logger,
	}

	go c.maintenanceLoop()

	return c, nil
}

func (c *Compactor) maintenanceLoop() {
	if c.cfg.MaintenanceCycle == 0 {
		level.Warn(c.logger).Log("msg", "blocklist Refresh Rate unset.  friggdb querying, compaction and retention effectively disabled.")
		return
	}

	c.doMaintenance()

	ticker := time.NewTicker(c.cfg.MaintenanceCycle)
	for range ticker.C {
		c.doMaintenance()
	}
}

func (c *Compactor) doMaintenance() {
	metricMaintenanceTotal.Inc()

	if c.cfg != nil {
		c.doCompaction()
		c.doRetention()
	}
}

func (c *Compactor) doCompaction() {
	// ctx := context.Background()

	// stop any existing compaction jobs
	if c.jobStopper != nil {
		start := time.Now()
		err := c.jobStopper.Stop()
		if err != nil {
			level.Warn(c.logger).Log("msg", "error during compaction cycle", "err", err)
			metricCompactionErrors.Inc()
		}
		metricCompactionStopDuration.Observe(time.Since(start).Seconds())
	}

	// start crazy jobs to do compaction with new list
	tenants := c.store.BlocklistTenants()

	var err error
	c.jobStopper, err = c.pool.RunStoppableJobs(tenants, func(payload interface{}, stopCh <-chan struct{}) error {
		var warning error
		tenantID := payload.(string)

		cursor := 0
	L:
		for {
			select {
			case <-stopCh:
				return warning
			default:
				var blocks []*backend.BlockMeta
				blocks, cursor = c.store.BlocksToCompact(tenantID, cursor, c.cfg.MaxCompactionRange) // todo: pass a context with a deadline?
				if cursor == cursorDone {
					break L
				}
				if blocks == nil {
					continue
				}
				err := c.compact(blocks, tenantID)
				if err != nil {
					warning = err
					metricCompactionErrors.Inc()
				}
			}
		}

		return warning
	})

	if err != nil {
		level.Error(c.logger).Log("msg", "failed to start compaction.  compaction broken until next maintenance cycle.", "err", err)
		metricCompactionErrors.Inc()
	}
}

func (c *Compactor) doRetention() {
	tenants := c.store.BlocklistTenants()

	// todo: continued abuse of runJobs.  need a runAllJobs() method or something
	_, err := c.pool.RunJobs(tenants, func(payload interface{}) ([]byte, error) {
		start := time.Now()
		defer func() { metricRetentionDuration.Observe(time.Since(start).Seconds()) }()

		tenantID := payload.(string)

		// iterate through block list.  make compacted anything that is past retention.
		cutoff := time.Now().Add(-c.cfg.BlockRetention)
		blocklist := c.store.Blocklist(tenantID)
		for _, b := range blocklist {
			if b.EndTime.Before(cutoff) {
				err := c.b.MarkBlockCompacted(b.BlockID, tenantID)
				if err != nil {
					level.Error(c.logger).Log("msg", "failed to mark block compacted during retention", "blockID", b.BlockID, "tenantID", tenantID, "err", err)
					metricRetentionErrors.Inc()
				} else {
					metricMarkedForDeletion.Inc()
				}
			}
		}

		// iterate through compacted list looking for blocks ready to be cleared
		cutoff = time.Now().Add(-c.cfg.CompactedBlockRetention)
		compactedBlocklist := c.store.CompactedBlocklist(tenantID)
		for _, b := range compactedBlocklist {
			if b.CompactedTime.Before(cutoff) {
				err := c.b.ClearBlock(b.BlockID, tenantID)
				if err != nil {
					level.Error(c.logger).Log("msg", "failed to clear compacted block during retention", "blockID", b.BlockID, "tenantID", tenantID, "err", err)
					metricRetentionErrors.Inc()
				} else {
					metricDeleted.Inc()
				}
			}
		}

		return nil, nil
	})

	if err != nil {
		level.Error(c.logger).Log("msg", "failure to start retention.  retention disabled until the next maintenance cycle", "err", err)
		metricRetentionErrors.Inc()
	}
}

// todo : this method is brittle and has weird failure conditions.  if it fails after it has written a new block then it will not clean up the old
//   in these cases it's possible that the compact method actually will start making more blocks.
func (c *Compactor) compact(blockMetas []*backend.BlockMeta, tenantID string) error {
	start := time.Now()
	defer func() { metricCompactionDuration.Observe(time.Since(start).Seconds()) }()

	var err error
	bookmarks := make([]*bookmark, 0, len(blockMetas))

	var totalRecords uint32
	for _, blockMeta := range blockMetas {
		totalRecords += blockMeta.TotalObjects

		backendReader := c.store.GetBackendReader()
		iter, err := backend.NewLazyIterator(tenantID, blockMeta.BlockID, c.cfg.ChunkSizeBytes, backendReader)
		if err != nil {
			return err
		}

		bookmarks = append(bookmarks, newBookmark(iter))

		_, err = backendReader.BlockMeta(blockMeta.BlockID, tenantID)
		if os.IsNotExist(err) {
			// if meta doesn't exist right now it probably means this block was compacted.  warn and bail
			level.Warn(c.logger).Log("msg", "unable to find meta during compaction", "blockID", blockMeta.BlockID, "tenantID", tenantID, "err", err)
			metricCompactionErrors.Inc()
			return nil
		} else if err != nil {
			return err
		}
	}

	recordsPerBlock := (totalRecords / outputBlocks) + 1
	var currentBlock *compactorBlock

	for !allDone(bookmarks) {
		var lowestID []byte
		var lowestObject []byte
		var lowestBookmark *bookmark

		// find lowest ID of the new object
		for _, b := range bookmarks {
			currentID, currentObject, err := b.current()
			if err == io.EOF {
				continue
			} else if err != nil {
				return err
			}

			// todo:  right now if we run into equal ids we take the larger object in the hopes that it's a more complete trace.
			//   in the future add a callback or something that allows the owning application to make a more intelligent choice
			//   such as combining traces if they're both incomplete
			if bytes.Equal(currentID, lowestID) {
				if len(currentObject) > len(lowestObject) {
					lowestID = currentID
					lowestObject = currentObject
					lowestBookmark = b
				}
			} else if len(lowestID) == 0 || bytes.Compare(currentID, lowestID) == -1 {
				lowestID = currentID
				lowestObject = currentObject
				lowestBookmark = b
			}
		}

		if len(lowestID) == 0 || len(lowestObject) == 0 || lowestBookmark == nil {
			return fmt.Errorf("failed to find a lowest object in compaction")
		}

		// make a new block if necessary
		if currentBlock == nil {
			h, err := c.store.WAL().NewWorkingBlock(uuid.New(), tenantID)
			if err != nil {
				return err
			}

			currentBlock, err = newCompactorBlock(h, c.bCfg.WAL.BloomFP, c.bCfg.WAL.IndexDownsample, blockMetas)
			if err != nil {
				return err
			}
		}

		// write to new block
		err = currentBlock.write(lowestID, lowestObject)
		if err != nil {
			return err
		}
		lowestBookmark.clear()

		// ship block to backend if done
		if uint32(currentBlock.length()) >= recordsPerBlock {
			err = c.writeCompactedBlock(currentBlock, tenantID)
			if err != nil {
				return err
			}
			currentBlock = nil
		}
	}

	// ship final block to backend
	if currentBlock != nil {
		err = c.writeCompactedBlock(currentBlock, tenantID)
		if err != nil {
			return err
		}
	}

	// mark old blocks compacted so they don't show up in polling
	for _, meta := range blockMetas {
		if err := c.b.MarkBlockCompacted(meta.BlockID, tenantID); err != nil {
			level.Error(c.logger).Log("msg", "unable to mark block compacted", "blockID", meta.BlockID, "tenantID", tenantID, "err", err)
			metricCompactionErrors.Inc()
		}
	}

	return nil
}

func (c *Compactor) writeCompactedBlock(b *compactorBlock, tenantID string) error {
	currentMeta := b.meta()
	currentIndex, err := b.index()
	if err != nil {
		return err
	}

	currentBloom, err := b.bloom()
	if err != nil {
		return err
	}

	err = c.store.Write(context.TODO(), b.id(), tenantID, currentMeta, currentBloom, currentIndex, b.objectFilePath())
	if err != nil {
		return err
	}

	err = b.clear()
	if err != nil {
		level.Warn(c.logger).Log("msg", "failed to clear compacted bloc", "blockID", currentMeta.BlockID, "tenantID", tenantID, "err", err)
	}

	return nil
}

func allDone(bookmarks []*bookmark) bool {
	for _, b := range bookmarks {
		if !b.done() {
			return false
		}
	}

	return true
}
