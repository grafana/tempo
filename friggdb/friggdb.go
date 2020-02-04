package friggdb

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	bloom "github.com/dgraph-io/ristretto/z"
	"github.com/dgryski/go-farm"
	"github.com/golang/protobuf/proto"

	"github.com/grafana/frigg/friggdb/backend"
	"github.com/grafana/frigg/friggdb/backend/local"
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

	bloomFP float64
	cfg     *Config
}

func New(cfg *Config) (Reader, Writer, error) {
	var err error
	var r backend.Reader
	var w backend.Writer

	switch cfg.Backend {
	case "local":
		r, w, err = local.New(cfg.Local)
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
		r:   r,
		w:   w,
		cfg: cfg,
	}

	return rw, rw, nil
}

func (rw *readerWriter) WriteBlock(ctx context.Context, c CompleteBlock) error {
	uuid, tenantID, records, blockFilePath := c.Identity()
	indexBytes, bloomBytes, err := marshalRecords(records, rw.cfg.BloomFilterFalsePositive)
	if err != nil {
		return err
	}

	metaBytes, err := json.Marshal(c.blockMeta())
	if err != nil {
		return err
	}

	return rw.w.Write(uuid, tenantID, metaBytes, bloomBytes, indexBytes, blockFilePath)
}

func (rw *readerWriter) WAL() (WAL, error) {
	return newWAL(&walConfig{
		filepath:        rw.cfg.WALFilepath,
		indexDownsample: rw.cfg.IndexDownsample,
	})
}

func (rw *readerWriter) Find(tenantID string, id ID, out proto.Message) (FindMetrics, bool, error) {
	metrics := FindMetrics{}

	blocklists, err := rw.r.Blocklist(tenantID)
	if err != nil {
		return metrics, false, err
	}

	for _, b := range blocklists {
		meta := &searchableBlockMeta{}
		err = json.Unmarshal(b, meta)
		if err != nil {
			return metrics, false, err
		}

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
