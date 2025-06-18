package backendscheduler

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/go-kit/log/level"
	"github.com/grafana/tempo/pkg/util/log"
	"github.com/grafana/tempo/tempodb/backend"
)

func (s *BackendScheduler) flushWorkCache(ctx context.Context) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	b, err := s.work.Marshal()
	if err != nil {
		return fmt.Errorf("failed to marshal work cache: %w", err)
	}

	workPath := filepath.Join(s.cfg.LocalWorkPath, backend.WorkFileName)

	err = os.MkdirAll(s.cfg.LocalWorkPath, 0o700)
	if err != nil {
		return fmt.Errorf("error creating directory %q: %w", s.cfg.LocalWorkPath, err)
	}

	err = os.WriteFile(workPath, b, 0o600)
	if err != nil {
		return fmt.Errorf("error writing %q: %w", workPath, err)
	}

	return nil
}

func (s *BackendScheduler) flushWorkCacheToBackend(ctx context.Context) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	b, err := s.work.Marshal()
	if err != nil {
		metricWorkFlushesFailed.Inc()
		return fmt.Errorf("failed to marshal work cache: %w", err)
	}

	err = s.writer.Write(ctx, backend.WorkFileName, []string{}, bytes.NewReader(b), -1, nil)
	if err != nil {
		return fmt.Errorf("failed to flush work cache: %w", err)
	}

	return nil
}

// loadWorkCache loads the work cache from the local filesystem
func (s *BackendScheduler) loadWorkCache(ctx context.Context) error {
	// Try to load the local work cache first, falling back to the backend if it doesn't exist.
	workPath := filepath.Join(s.cfg.LocalWorkPath, backend.WorkFileName)
	data, err := os.ReadFile(workPath)
	if err != nil {
		if !os.IsNotExist(err) {
			level.Error(log.Logger).Log("msg", "failed to read work cache from local path", "path", workPath, "error", err)
		}
		return s.loadWorkCacheFromBackend(ctx)
	}

	err = s.work.Unmarshal(data)
	if err != nil {
		level.Error(log.Logger).Log("msg", "failed to unmarshal work cache from local path", "path", workPath, "error", err)
		return s.loadWorkCacheFromBackend(ctx)
	}

	// Once the work cache is loaded, replay the work list on top of the
	// blocklist to ensure we only hand out jobs for blocks which need visiting.
	s.replayWorkOnBlocklist()

	return nil
}

func (s *BackendScheduler) loadWorkCacheFromBackend(ctx context.Context) error {
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

	// Once the work cache is loaded, replay the work list on top of the
	// blocklist to ensure we only hand out jobs for blocks which need visiting.
	s.replayWorkOnBlocklist()

	return nil
}
