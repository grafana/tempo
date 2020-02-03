package friggdb

import (
	"context"
	"fmt"

	bloom "github.com/dgraph-io/ristretto/z"
	"github.com/dgryski/go-farm"
	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"

	"github.com/grafana/frigg/friggdb/backend"
	"github.com/grafana/frigg/friggdb/backend/local"
)

type Writer interface {
	WriteBlock(ctx context.Context, block CompleteBlock) error
	WAL() (WAL, error)
}

type Reader interface {
	Find(tenantID string, id ID, out proto.Message) (FindMetrics, error)
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

	return rw.w.Write(ctx, uuid, tenantID, bloomBytes, indexBytes, blockFilePath)
}

func (rw *readerWriter) WAL() (WAL, error) {
	return newWAL(&walConfig{
		filepath:        rw.cfg.WALFilepath,
		indexDownsample: rw.cfg.IndexDownsample,
	})
}

func (rw *readerWriter) Find(tenantID string, id ID, out proto.Message) (FindMetrics, error) {
	metrics := FindMetrics{}

	err := rw.r.Bloom(tenantID, func(bloomBytes []byte, blockID uuid.UUID) (bool, error) {
		filter := bloom.JSONUnmarshal(bloomBytes)
		metrics.BloomFilterReads++
		metrics.BloomFilterBytesRead += len(bloomBytes)

		if filter.Has(farm.Fingerprint64(id)) {
			indexBytes, err := rw.r.Index(blockID, tenantID)
			metrics.IndexReads++
			metrics.IndexBytesRead += len(indexBytes)
			if err != nil {
				return false, err
			}

			record, err := findRecord(id, indexBytes)
			if err != nil {
				return false, err
			}

			if record != nil {
				traceBytes, err := rw.r.Trace(blockID, tenantID, record.Start, record.Length)
				metrics.BlockReads++
				metrics.BlockBytesRead += len(traceBytes)
				if err != nil {
					return false, err
				}

				err = proto.Unmarshal(traceBytes, out)
				if err != nil {
					return false, err
				}

				return false, nil
			}

			return true, nil
		}

		return true, nil
	})

	if err != nil {
		return metrics, err
	}

	return metrics, nil
}
