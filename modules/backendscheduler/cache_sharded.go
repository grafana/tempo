package backendscheduler

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-kit/log/level"
	"github.com/grafana/tempo/modules/backendscheduler/work"
	"github.com/grafana/tempo/pkg/util/log"
	"github.com/grafana/tempo/tempodb/backend"
	"go.opentelemetry.io/otel/attribute"
)

// flushWorkCacheOptimized flushes the work cache using sharding optimizations if available
func (s *BackendScheduler) flushWorkCacheOptimized(ctx context.Context, affectedJobIDs []string) error {
	_, span := tracer.Start(ctx, "flushWorkCacheOptimized")
	defer span.End()

	span.AddEvent("lock.acquire.start")
	s.mtx.Lock()
	span.AddEvent("lock.acquired")
	defer s.mtx.Unlock()

	// Try to use sharded implementation if available
	if shardedWork, ok := work.AsSharded(s.work); ok {
		return s.flushShardedWorkCache(ctx, shardedWork, affectedJobIDs)
	}

	// Fallback to original full marshal
	return s.flushWorkCacheFallback(ctx)
}

// flushShardedWorkCache uses sharding optimizations to flush only affected shards
func (s *BackendScheduler) flushShardedWorkCache(ctx context.Context, shardedWork work.ShardedWorkInterface, affectedJobIDs []string) error {
	_, span := tracer.Start(ctx, "flushShardedWorkCache")
	defer span.End()

	err := os.MkdirAll(s.cfg.LocalWorkPath, 0o700)
	if err != nil {
		return fmt.Errorf("error creating directory %q: %w", s.cfg.LocalWorkPath, err)
	}

	if len(affectedJobIDs) == 0 {
		// No specific jobs affected, do full flush
		return s.flushAllShards(ctx, shardedWork)
	}

	// Only flush affected shards
	shardData, err := shardedWork.MarshalAffectedShards(affectedJobIDs)
	if err != nil {
		return fmt.Errorf("failed to marshal affected shards: %w", err)
	}

	// Write only the affected shard files
	totalBytes := 0
	for shardID, data := range shardData {
		shardPath := s.filenameForShard(shardID)

		span.AddEvent("writeFile.start")
		err = os.WriteFile(shardPath, data, 0o600)
		span.AddEvent("writeFile.complete")
		if err != nil {
			return fmt.Errorf("error writing shard file %q: %w", shardPath, err)
		}
		totalBytes += len(data)
	}

	span.SetAttributes(
		attribute.Int("affected_shards", len(shardData)),
		attribute.Int("total_bytes", totalBytes),
		attribute.Int("affected_jobs", len(affectedJobIDs)),
	)

	metricWorkCacheFileSize.Observe(float64(totalBytes))

	level.Debug(log.Logger).Log(
		"msg", "flushed affected shards",
		"affected_shards", len(shardData),
		"affected_jobs", len(affectedJobIDs),
		"total_bytes", totalBytes,
	)

	return nil
}

// flushAllShards flushes all shards (used for startup/shutdown)
func (s *BackendScheduler) flushAllShards(ctx context.Context, shardedWork work.ShardedWorkInterface) error {
	_, span := tracer.Start(ctx, "flushAllShards")
	defer span.End()

	totalBytes := 0

	span.AddEvent("marshal.shards.start")
	for i := range work.ShardCount {
		shardID := uint8(i)
		data, err := shardedWork.MarshalShard(shardID)
		if err != nil {
			return fmt.Errorf("failed to marshal shard %d: %w", shardID, err)
		}

		shardPath := s.filenameForShard(shardID)
		err = os.WriteFile(shardPath, data, 0o600)
		if err != nil {
			return fmt.Errorf("error writing shard file %q: %w", shardPath, err)
		}
		totalBytes += len(data)
	}
	span.AddEvent("marshal.shards.complete")

	span.SetAttributes(
		attribute.Int("total_shards", work.ShardCount),
		attribute.Int("total_bytes", totalBytes),
	)

	metricWorkCacheFileSize.Observe(float64(totalBytes))

	return nil
}

// flushWorkCacheFallback uses the original marshal approach for non-sharded work
func (s *BackendScheduler) flushWorkCacheFallback(ctx context.Context) error {
	_, span := tracer.Start(ctx, "flushWorkCacheFallback")
	defer span.End()

	span.AddEvent("marshal.start")
	b, err := s.work.Marshal()
	span.AddEvent("marshal.complete")
	if err != nil {
		return fmt.Errorf("failed to marshal work cache: %w", err)
	}

	workPath := filepath.Join(s.cfg.LocalWorkPath, backend.WorkFileName)

	err = os.MkdirAll(s.cfg.LocalWorkPath, 0o700)
	if err != nil {
		return fmt.Errorf("error creating directory %q: %w", s.cfg.LocalWorkPath, err)
	}

	span.AddEvent("writeFile.start")
	err = os.WriteFile(workPath, b, 0o600)
	span.AddEvent("writeFile.complete")
	if err != nil {
		return fmt.Errorf("error writing %q: %w", workPath, err)
	}

	span.SetAttributes(
		attribute.Int("work_cache.file_size_bytes", len(b)),
		attribute.String("work_cache.file_path", workPath),
	)

	metricWorkCacheFileSize.Observe(float64(len(b)))

	return nil
}

// loadWorkCacheOptimized loads the work cache using the configured implementation
func (s *BackendScheduler) loadWorkCacheOptimized(ctx context.Context) error {
	ctx, span := tracer.Start(ctx, "loadWorkCacheOptimized")
	defer span.End()

	var (
		// Check what work implementation we're using
		useSharding = s.cfg.Work.Sharded

		// Check what files exist
		workPath     = s.cfg.LocalWorkPath
		legacyPath   = filepath.Join(workPath, backend.WorkFileName)
		shardsExist  = s.checkForShardFiles()
		legacyExists = s.checkForLegacyFile(legacyPath)
	)

	level.Info(log.Logger).Log("msg", "loading work cache", "use_sharding", useSharding, "shards_exist", shardsExist, "legacy_exists", legacyExists)

	switch {
	case useSharding && shardsExist:
		// Sharding enabled, shard files exist - load shards
		return s.work.LoadFromLocal(ctx, s.cfg.LocalWorkPath)

	case useSharding && legacyExists && !shardsExist:
		// Sharding enabled, only legacy file exists - migrate to shards
		level.Info(log.Logger).Log("msg", "migrating legacy work cache to sharded format")
		return s.migrateWorkCacheLegacyToSharded(ctx, legacyPath)

	case !useSharding && legacyExists:
		// Sharding disabled, legacy file exists - load legacy
		return s.work.LoadFromLocal(ctx, s.cfg.LocalWorkPath)

	case !useSharding && shardsExist && !legacyExists:
		// Sharding disabled, only shard files exist - migrate back to legacy
		level.Info(log.Logger).Log("msg", "migrating sharded work cache back to legacy format")
		return s.migrateWorkCacheShardedToLegacy(ctx)

	default:
		// No files exist - try backend, then start fresh
		return s.loadWorkCacheFromBackend(ctx)
	}
}

// checkForShardFiles checks if any shard files exist
func (s *BackendScheduler) checkForShardFiles() bool {
	for i := range work.ShardCount {
		if _, err := os.Stat(s.filenameForShard(uint8(i))); err == nil {
			return true // Found at least one shard file
		}
	}
	return false
}

func (s *BackendScheduler) filenameForShard(shardID uint8) string {
	return filepath.Join(s.cfg.LocalWorkPath, fmt.Sprintf("shard_%03d.json", shardID))
}

// checkForLegacyFile checks if the legacy work.json file exists
func (s *BackendScheduler) checkForLegacyFile(legacyPath string) bool {
	_, err := os.Stat(legacyPath)
	return err == nil
}

// migrateWorkCacheShardedToLegacy converts shard files back to single work.json
// This supports rolling back from sharded to original implementation
func (s *BackendScheduler) migrateWorkCacheShardedToLegacy(ctx context.Context) error {
	_, span := tracer.Start(ctx, "migrateShardedToLegacy")
	defer span.End()

	// Create temporary sharded work to load the shard files
	tempShardedWork := work.NewSharded(s.cfg.Work)

	// Load all existing shard files
	shardsLoaded := 0
	for i := range work.ShardCount {
		shardID := uint8(i)
		shardPath := s.filenameForShard(shardID)

		data, err := os.ReadFile(shardPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue // Skip empty shards
			}
			return fmt.Errorf("failed to read shard file %q: %w", shardPath, err)
		}

		err = tempShardedWork.UnmarshalShard(shardID, data)
		if err != nil {
			level.Error(log.Logger).Log("msg", "failed to unmarshal shard during migration", "shard_id", shardID, "error", err)
			continue
		}
		shardsLoaded++
	}

	level.Info(log.Logger).Log("msg", "loaded shard files for migration", "shards_loaded", shardsLoaded)

	// Get all jobs from the temporary sharded work
	allJobs := tempShardedWork.ListJobs()

	// Add all jobs to our non-sharded work instance
	for _, job := range allJobs {
		err := s.work.AddJobPreservingState(job)
		if err != nil {
			return fmt.Errorf("failed to add job during reverse migration: %w", err)
		}
	}

	// Marshal and save as legacy work.json
	data, err := s.work.Marshal()
	if err != nil {
		return fmt.Errorf("failed to marshal work for legacy format: %w", err)
	}

	legacyPath := filepath.Join(s.cfg.LocalWorkPath, backend.WorkFileName)

	err = os.MkdirAll(filepath.Dir(legacyPath), 0o700)
	if err != nil {
		return fmt.Errorf("failed to create directory for legacy file: %w", err)
	}

	err = os.WriteFile(legacyPath, data, 0o600)
	if err != nil {
		return fmt.Errorf("failed to write legacy work file: %w", err)
	}

	// Remove all shard files after migration
	for i := range work.ShardCount {
		shardPath := s.filenameForShard(uint8(i))

		if _, err := os.Stat(shardPath); err == nil {
			err = os.Remove(shardPath)
			if err != nil {
				return fmt.Errorf("failed to remove shard file after migration %q: %w", shardPath, err)
			}
		}
	}

	level.Info(log.Logger).Log("msg", "successfully migrated from sharded back to legacy format", "jobs_migrated", len(allJobs))

	err = s.flushWorkCacheToBackend(ctx)
	if err != nil {
		return fmt.Errorf("failed to flush migrated work cache to backend: %w", err)
	}

	// Remove the sharded object from the backend
	_ = s.writer.Delete(ctx, backend.ShardedWorkFileName, backend.KeyPath{}, nil)

	// Replay work on blocklist
	return s.replayWorkOnBlocklist(ctx)
}

// migrateWorkCacheLegacyToSharded migrates from old work.json to sharded format
func (s *BackendScheduler) migrateWorkCacheLegacyToSharded(ctx context.Context, legacyPath string) error {
	_, span := tracer.Start(ctx, "migrateLegacyWorkCache")
	defer span.End()

	// Read legacy file
	data, err := os.ReadFile(legacyPath)
	if err != nil {
		return fmt.Errorf("failed to read legacy work cache: %w", err)
	}

	// Create temporary original work instance for migration
	tempWork := work.New(s.cfg.Work)
	err = tempWork.Unmarshal(data)
	if err != nil {
		return fmt.Errorf("failed to unmarshal legacy work cache: %w", err)
	}

	newWork, err := work.MigrateToSharded(tempWork, s.cfg.Work)
	if err != nil {
		return fmt.Errorf("failed to migrate legacy work to sharded format: %w", err)
	}

	// Use the new sharded work instance
	s.work = newWork

	// Save in new sharded format
	err = s.flushWorkCacheOptimized(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to save migrated sharded work: %w", err)
	}

	// Remove legacy file after migration
	err = os.Remove(legacyPath)
	if err != nil {
		return fmt.Errorf("failed to remove legacy work cache after migration: %w", err)
	}

	err = s.flushWorkCacheToBackend(ctx)
	if err != nil {
		return fmt.Errorf("failed to flush migrated work cache to backend: %w", err)
	}

	level.Info(log.Logger).Log("msg", "successfully migrated legacy work cache to sharded format")

	// Remove the legacy object from the backend
	_ = s.writer.Delete(ctx, backend.WorkFileName, backend.KeyPath{}, nil)

	// Replay work on blocklist
	return s.replayWorkOnBlocklist(ctx)
}
