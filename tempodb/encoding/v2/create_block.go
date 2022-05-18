package v2

import (
	"context"
	"io"

	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/pkg/errors"
)

const DefaultFlushSizeBytes int = 30 * 1024 * 1024 // 30 MiB

func CreateBlock(ctx context.Context, cfg *common.BlockConfig, meta *backend.BlockMeta, i common.Iterator, dec model.ObjectDecoder, to backend.Writer) (*backend.BlockMeta, error) {
	defer i.Close()

	newBlock, err := NewStreamingBlock(cfg, meta.BlockID, meta.TenantID, []*backend.BlockMeta{meta}, meta.TotalObjects)
	if err != nil {
		return nil, errors.Wrap(err, "error creating streaming block")
	}

	var tracker backend.AppendTracker
	for {
		id, trBytes, err := i.Next(ctx)
		if err != nil && err != io.EOF {
			return nil, errors.Wrap(err, "error iterating")
		}

		if id == nil {
			break
		}

		// This assumes the incoming bytes are the same data encoding.
		err = newBlock.AddObject(id, trBytes)
		if err != nil {
			return nil, errors.Wrap(err, "error adding object to compactor block")
		}

		if newBlock.CurrentBufferLength() > DefaultFlushSizeBytes {
			tracker, _, err = newBlock.FlushBuffer(ctx, tracker, to)
			if err != nil {
				return nil, errors.Wrap(err, "error flushing compactor block")
			}
		}
	}

	_, err = newBlock.Complete(ctx, tracker, to)
	if err != nil {
		return nil, errors.Wrap(err, "error completing compactor block")
	}

	return newBlock.BlockMeta(), nil
}
