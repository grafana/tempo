package storage

import (
	"context"

	bloom "github.com/dgraph-io/ristretto/z"
	"github.com/dgryski/go-farm"
	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"

	"github.com/grafana/frigg/pkg/friggpb"
	"github.com/grafana/frigg/pkg/ingester/wal"
	"github.com/grafana/frigg/pkg/storage/trace_backend"
)

type TraceWriter interface {
	WriteBlock(ctx context.Context, block wal.CompleteBlock) error
}

type TraceReader interface {
	FindTrace(tenantID string, id wal.ID) (*friggpb.Trace, FindMetrics, error)
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
	r trace_backend.Reader
	w trace_backend.Writer

	bloomFP float64
}

func (rw *readerWriter) WriteBlock(ctx context.Context, block wal.CompleteBlock) error {
	uuid, tenantID, records, blockFilePath := block.Identity()
	indexBytes, bloomBytes, err := wal.MarshalRecords(records, rw.bloomFP)

	if err != nil {
		return err
	}

	return rw.w.Write(ctx, uuid, tenantID, bloomBytes, indexBytes, blockFilePath)
}

func (rw *readerWriter) FindTrace(tenantID string, id wal.ID) (*friggpb.Trace, FindMetrics, error) {
	metrics := FindMetrics{}
	var found *friggpb.Trace

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

			record, err := wal.FindRecord(id, indexBytes)
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

				out := &friggpb.Trace{}
				err = proto.Unmarshal(traceBytes, out)
				if err != nil {
					return false, err
				}

				// got it
				found = out
				return false, nil
			}

			return true, nil
		}

		return true, nil
	})

	if err != nil {
		return nil, metrics, err
	}

	return found, metrics, nil
}
