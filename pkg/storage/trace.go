package storage

import (
	"context"

	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	"github.com/willf/bloom"

	"github.com/joe-elliott/frigg/pkg/friggpb"
	"github.com/joe-elliott/frigg/pkg/storage/trace_backend"
)

type TraceWriter interface {
	WriteBlock(ctx context.Context, blockID uuid.UUID, tenantID string, records []*TraceRecord, blockFilePath string) error
}

type TraceReader interface {
	FindTrace(tenantID string, id TraceID) (*friggpb.Trace, error)
}

type readerWriter struct {
	r trace_backend.Reader
	w trace_backend.Writer
}

func (rw *readerWriter) WriteBlock(ctx context.Context, blockID uuid.UUID, tenantID string, records []*TraceRecord, blockFilePath string) error {
	bloomBytes, indexBytes, err := EncodeRecords(records)

	if err != nil {
		return err
	}

	return rw.w.Write(ctx, blockID, tenantID, bloomBytes, indexBytes, blockFilePath)
}

func (rw *readerWriter) FindTrace(tenantID string, id TraceID) (*friggpb.Trace, error) {
	out := &friggpb.Trace{}

	err := rw.r.Bloom(tenantID, func(bloomBytes []byte, blockID uuid.UUID) (bool, error) {
		filter := bloom.New(1, 1)
		filter.GobDecode(bloomBytes)

		if filter.Test(id) {
			indexBytes, err := rw.r.Index(blockID, tenantID)
			if err != nil {
				return false, err
			}

			record, err := FindRecord(id, indexBytes)
			if err != nil {
				return false, err
			}

			if record != nil {
				traceBytes, err := rw.r.Trace(blockID, tenantID, record.Start, record.Length)
				if err != nil {
					return false, err
				}

				err = proto.Unmarshal(traceBytes, out)
				if err != nil {
					return false, err
				}

				// got it
				return false, nil
			}

			return true, nil
		}

		return true, nil
	})

	if err != nil {
		return nil, err
	}

	return out, nil
}
