package backendscheduler

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"

	"github.com/go-kit/log/level"
	"github.com/grafana/tempo/pkg/util/log"
	"github.com/grafana/tempo/tempodb/backend"
)

func (s *BackendScheduler) flushWorkCacheToBackend(ctx context.Context) error {
	_, span := tracer.Start(ctx, "flushWorkCacheToBackend")
	defer span.End()

	s.mtx.Lock()
	defer s.mtx.Unlock()

	// Always use sharded approach since we only have one implementation now
	b, err := s.work.Marshal()
	if err != nil {
		metricWorkFlushesFailed.Inc()
		return fmt.Errorf("failed to marshal sharded work cache: %w", err)
	}

	err = s.writer.Write(ctx, backend.WorkFileName, []string{}, bytes.NewReader(b), int64(len(b)), nil)
	if err != nil {
		return fmt.Errorf("failed to flush sharded work cache: %w", err)
	}

	return nil
}

func (s *BackendScheduler) loadWorkCacheFromBackend(ctx context.Context) error {
	ctx, span := tracer.Start(ctx, "loadWorkCacheFromBackend")
	defer span.End()

	// Always use sharded work file since we only have one implementation now
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
		return fmt.Errorf("failed to unmarshal sharded work cache: %w", err)
	}

	return s.replayWorkOnBlocklist(ctx)
}

// loadWorkCache loads the work cache using the configured implementation
func (s *BackendScheduler) loadWorkCache(ctx context.Context) error {
	ctx, span := tracer.Start(ctx, "loadWorkCache")
	defer span.End()

	// Check if shard files exist before attempting to load them
	if s.checkForShardFiles() {
		// Try to load the local work cache first, falling back to the backend if it doesn't exist.
		err := s.work.LoadFromLocal(ctx, s.cfg.LocalWorkPath)
		if err != nil {
			if !os.IsNotExist(err) {
				level.Error(log.Logger).Log("msg", "failed to read work cache from local path", "path", s.cfg.LocalWorkPath, "error", err)
			}

			return s.loadWorkCacheFromBackend(ctx)
		}

		return s.replayWorkOnBlocklist(ctx)
	}

	// No shard files found locally, load from backend
	return s.loadWorkCacheFromBackend(ctx)
}
