package backendscheduler

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

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

	workPath := s.cfg.LocalWorkPath

	err := os.MkdirAll(workPath, 0o700)
	if err != nil {
		return fmt.Errorf("error creating directory %q: %w", workPath, err)
	}

	if len(affectedJobIDs) == 0 {
		// No specific jobs affected, do full flush
		return s.flushAllShards(ctx, shardedWork, workPath)
	}

	// Only flush affected shards
	shardData, err := shardedWork.MarshalAffectedShards(affectedJobIDs)
	if err != nil {
		return fmt.Errorf("failed to marshal affected shards: %w", err)
	}

	// Write only the affected shard files
	totalBytes := 0
	for shardID, data := range shardData {
		shardPath := filepath.Join(workPath, fmt.Sprintf("work_%03d.json", shardID))

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
func (s *BackendScheduler) flushAllShards(ctx context.Context, shardedWork work.ShardedWorkInterface, workPath string) error {
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

		shardPath := filepath.Join(workPath, fmt.Sprintf("shard_%03d.json", shardID))
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

	// Check what work implementation we're using
	useSharding := s.cfg.Work.Sharded

	// Check what files exist
	workPath := s.cfg.LocalWorkPath
	legacyPath := filepath.Join(workPath, backend.WorkFileName)
	shardsExist := s.checkForShardFiles(workPath)
	legacyExists := s.checkForLegacyFile(legacyPath)

	switch {
	case useSharding && shardsExist:
		// Sharding enabled, shard files exist - load shards
		return s.loadShardedWorkCache(ctx, s.work.(work.ShardedWorkInterface))

	case useSharding && legacyExists && !shardsExist:
		// Sharding enabled, only legacy file exists - migrate to shards
		level.Info(log.Logger).Log("msg", "migrating legacy work cache to sharded format")
		return s.migrateLegacyWorkCache(ctx, s.work.(work.ShardedWorkInterface), legacyPath)

	case !useSharding && legacyExists:
		// Sharding disabled, legacy file exists - load legacy
		return s.loadWorkCacheFallback(ctx)

	case !useSharding && shardsExist && !legacyExists:
		// Sharding disabled, only shard files exist - migrate back to legacy
		level.Info(log.Logger).Log("msg", "migrating sharded work cache back to legacy format")
		return s.migrateShardedToLegacy(ctx, workPath, legacyPath)

	case !useSharding && shardsExist && legacyExists:
		// Both exist with sharding disabled - prefer legacy (user explicitly disabled sharding)
		level.Warn(log.Logger).Log("msg", "both legacy and shard files exist with sharding disabled, using legacy file")
		return s.loadWorkCacheFallback(ctx)

	case useSharding && shardsExist && legacyExists:
		// Both exist with sharding enabled - prefer shards (migration already happened)
		level.Info(log.Logger).Log("msg", "both legacy and shard files exist with sharding enabled, using shard files")
		return s.loadShardedWorkCache(ctx, s.work.(work.ShardedWorkInterface))

	default:
		// No files exist - try backend, then start fresh
		return s.loadWorkCacheFromBackend(ctx)
	}
}

// checkForShardFiles checks if any shard files exist
func (s *BackendScheduler) checkForShardFiles(workPath string) bool {
	for i := 0; i < work.ShardCount; i++ {
		shardPath := filepath.Join(workPath, fmt.Sprintf("shard_%03d.json", i))
		if _, err := os.Stat(shardPath); err == nil {
			return true // Found at least one shard file
		}
	}
	return false
}

// checkForLegacyFile checks if the legacy work.json file exists
func (s *BackendScheduler) checkForLegacyFile(legacyPath string) bool {
	_, err := os.Stat(legacyPath)
	return err == nil
}

// migrateShardedToLegacy converts shard files back to single work.json
// This supports rolling back from sharded to original implementation
func (s *BackendScheduler) migrateShardedToLegacy(ctx context.Context, workPath, legacyPath string) error {
	_, span := tracer.Start(ctx, "migrateShardedToLegacy")
	defer span.End()

	// Create temporary sharded work to load the shard files
	tempShardedWork := work.NewSharded(s.cfg.Work)

	// Load all existing shard files
	shardsLoaded := 0
	for i := 0; i < work.ShardCount; i++ {
		shardID := uint8(i)
		shardPath := filepath.Join(workPath, fmt.Sprintf("shard_%03d.json", shardID))

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

	err = os.MkdirAll(filepath.Dir(legacyPath), 0o700)
	if err != nil {
		return fmt.Errorf("failed to create directory for legacy file: %w", err)
	}

	err = os.WriteFile(legacyPath, data, 0o600)
	if err != nil {
		return fmt.Errorf("failed to write legacy work file: %w", err)
	}

	// Archive shard files instead of deleting them (safety measure)
	archiveDir := filepath.Join(workPath, "shards_archived_"+time.Now().Format("20060102_150405"))
	err = os.MkdirAll(archiveDir, 0o700)
	if err != nil {
		level.Warn(log.Logger).Log("msg", "failed to create archive directory", "error", err)
	} else {
		// Move shard files to archive
		for i := 0; i < work.ShardCount; i++ {
			shardPath := filepath.Join(workPath, fmt.Sprintf("shard_%03d.json", i))
			if _, err := os.Stat(shardPath); err == nil {
				archivePath := filepath.Join(archiveDir, fmt.Sprintf("shard_%03d.json", i))
				if renameErr := os.Rename(shardPath, archivePath); renameErr != nil {
					level.Warn(log.Logger).Log("msg", "failed to archive shard file", "shard", i, "error", renameErr)
				}
			}
		}
		level.Info(log.Logger).Log("msg", "archived shard files", "archive_dir", archiveDir)
	}

	level.Info(log.Logger).Log("msg", "successfully migrated from sharded back to legacy format", "jobs_migrated", len(allJobs))

	// Replay work on blocklist
	s.replayWorkOnBlocklist(ctx)

	return nil
}

// loadShardedWorkCache loads work cache from shard files
func (s *BackendScheduler) loadShardedWorkCache(ctx context.Context, shardedWork work.ShardedWorkInterface) error {
	ctx, span := tracer.Start(ctx, "loadShardedWorkCache")
	defer span.End()

	workPath := s.cfg.LocalWorkPath

	// Check for legacy work.json file first
	legacyPath := filepath.Join(workPath, backend.WorkFileName)
	if _, err := os.Stat(legacyPath); err == nil {
		level.Info(log.Logger).Log("msg", "found legacy work cache, migrating to sharded format")
		return s.migrateLegacyWorkCache(ctx, shardedWork, legacyPath)
	}

	// Load shard files
	shardsLoaded := 0
	for i := range work.ShardCount {
		shardID := uint8(i)
		shardPath := filepath.Join(workPath, fmt.Sprintf("shard_%03d.json", shardID))

		data, err := os.ReadFile(shardPath)
		if err != nil {
			if os.IsNotExist(err) {
				// Shard file doesn't exist, skip (empty shard)
				continue
			}
			return fmt.Errorf("failed to read shard file %q: %w", shardPath, err)
		}

		err = shardedWork.UnmarshalShard(shardID, data)
		if err != nil {
			level.Error(log.Logger).Log("msg", "failed to unmarshal shard", "shard_id", shardID, "error", err)
			// Continue loading other shards
			continue
		}
		shardsLoaded++
	}

	level.Info(log.Logger).Log("msg", "loaded sharded work cache", "shards_loaded", shardsLoaded)

	// Replay work on blocklist
	s.replayWorkOnBlocklist(ctx)

	return nil
}

// migrateLegacyWorkCache migrates from old work.json to sharded format
func (s *BackendScheduler) migrateLegacyWorkCache(ctx context.Context, shardedWork work.ShardedWorkInterface, legacyPath string) error {
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

	// Migrate jobs to sharded work by copying them manually
	jobs := tempWork.ListJobs()
	for _, job := range jobs {
		err = shardedWork.AddJobPreservingState(job)
		if err != nil {
			return fmt.Errorf("failed to add job during migration: %w", err)
		}
	}

	// Save in new sharded format
	err = s.flushAllShards(ctx, shardedWork, s.cfg.LocalWorkPath)
	if err != nil {
		return fmt.Errorf("failed to save migrated sharded work: %w", err)
	}

	// Backup and remove legacy file
	backupPath := legacyPath + ".backup"
	err = os.Rename(legacyPath, backupPath)
	if err != nil {
		level.Warn(log.Logger).Log("msg", "failed to backup legacy work cache", "error", err)
	}

	level.Info(log.Logger).Log("msg", "successfully migrated legacy work cache to sharded format")

	// Replay work on blocklist
	s.replayWorkOnBlocklist(ctx)

	return nil
}

// loadWorkCacheFallback loads using the original approach
func (s *BackendScheduler) loadWorkCacheFallback(ctx context.Context) error {
	ctx, span := tracer.Start(ctx, "loadWorkCacheFallback")
	defer span.End()

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
	s.replayWorkOnBlocklist(ctx)

	return nil
}
