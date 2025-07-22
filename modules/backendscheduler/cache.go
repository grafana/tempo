package backendscheduler

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/go-kit/log/level"
	"github.com/grafana/tempo/modules/backendscheduler/work"
	"github.com/grafana/tempo/pkg/util/log"
	"github.com/grafana/tempo/tempodb/backend"
)

func (s *BackendScheduler) flushWorkCacheToBackend(ctx context.Context) error {
	_, span := tracer.Start(ctx, "flushWorkCacheToBackend")
	defer span.End()

	s.mtx.Lock()
	defer s.mtx.Unlock()

	if shardedWork, ok := work.AsSharded(s.work); ok {
		b, err := shardedWork.Marshal()
		if err != nil {
			metricWorkFlushesFailed.Inc()
			return fmt.Errorf("failed to marshal sharded work cache: %w", err)
		}

		err = s.writer.Write(ctx, backend.ShardedWorkFileName, []string{}, bytes.NewReader(b), int64(len(b)), nil)
		if err != nil {
			return fmt.Errorf("failed to flush sharded work cache: %w", err)
		}

		// If we're sharded, don't write the legacy work file
		return nil
	}

	b, err := s.work.Marshal()
	if err != nil {
		metricWorkFlushesFailed.Inc()
		return fmt.Errorf("failed to marshal work cache: %w", err)
	}

	err = s.writer.Write(ctx, backend.WorkFileName, []string{}, bytes.NewReader(b), int64(len(b)), nil)
	if err != nil {
		return fmt.Errorf("failed to flush work cache: %w", err)
	}

	return nil
}

func (s *BackendScheduler) loadWorkCacheFromBackend(ctx context.Context) error {
	ctx, span := tracer.Start(ctx, "loadWorkCacheFromBackend")
	defer span.End()

	if ok := work.IsSharded(s.work); ok {
		reader, _, err := s.reader.Read(ctx, backend.ShardedWorkFileName, backend.KeyPath{}, nil)
		if err != nil {
			return err
		}
		defer func() {
			if closeErr := reader.Close(); err != nil {
				level.Error(log.Logger).Log("msg", "failed to close reader", "err", closeErr)
			}
		}()

		data, err := io.ReadAll(reader)
		if err != nil {
			return err
		}

		err = s.work.Unmarshal(data)
		if err != nil {
			return fmt.Errorf("failed to unmarshal sharded work cache: %w", err)
		}

		return s.replayWorkOnBlocklist(ctx)
	}

	reader, _, err := s.reader.Read(ctx, backend.WorkFileName, backend.KeyPath{}, nil)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := reader.Close(); err != nil {
			level.Error(log.Logger).Log("msg", "failed to close reader", "err", closeErr)
		}
	}()

	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}

	err = s.work.Unmarshal(data)
	if err != nil {
		return fmt.Errorf("failed to unmarshal work cache: %w", err)
	}

	return s.replayWorkOnBlocklist(ctx)
}
