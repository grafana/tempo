package backendscheduler

import (
	"bytes"
	"context"
	"fmt"
	"io"

	jsoniter "github.com/json-iterator/go"

	"github.com/go-kit/log/level"
	"github.com/grafana/tempo/pkg/util/log"
	"github.com/grafana/tempo/tempodb/backend"
)

func (s *BackendScheduler) flushWorkCache(ctx context.Context) error {
	s.workMtx.RLock()
	defer s.workMtx.RUnlock()

	b, err := jsoniter.Marshal(s.work)
	if err != nil {
		return fmt.Errorf("failed to marshal work cache: %w", err)
	}

	return s.writer.Write(ctx, backend.WorkFileName, []string{}, bytes.NewReader(b), -1, nil)
}

// readSeedFile reads the cluster seed file from the object store.
func (s *BackendScheduler) loadWorkCache(ctx context.Context) error {
	s.workMtx.Lock()
	defer s.workMtx.Unlock()

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

	err = jsoniter.Unmarshal(data, &s.work)
	if err != nil {
		return fmt.Errorf("failed to unmarshal work cache: %w", err)
	}

	return nil
}
