// SPDX-License-Identifier: AGPL-3.0-only
// Adapted from github.com/grafana/mimir/pkg/storage/ingest/offset_file.go

package ingest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"

	"github.com/grafana/tempo/v3/pkg/util/atomicfs"
)

// offsetFileVersion is the version of the offset file format for future evolution.
const offsetFileVersion = 1

// offsetFileData is the JSON structure persisted to disk.
type offsetFileData struct {
	Version     int   `json:"version"`
	PartitionID int32 `json:"partition_id"`
	Offset      int64 `json:"offset"`
}

// OffsetFile persists the last committed Kafka offset for a single partition to a local
// file, so that the offset survives a Kafka consumer group being garbage collected (which
// happens when the group has no members - see grafana/tempo-squad#1389).
//
// The file is expected to live next to the local data the offset describes (e.g. a WAL
// directory), so that if that local data is lost, the offset is naturally reset along with
// it rather than silently pointing at data that's gone.
type OffsetFile struct {
	filePath    string
	partitionID int32
	logger      log.Logger
	mu          sync.Mutex
}

// NewOffsetFile creates a new OffsetFile. filePath must be non-empty.
func NewOffsetFile(filePath string, partitionID int32, logger log.Logger) *OffsetFile {
	return &OffsetFile{
		filePath:    filePath,
		partitionID: partitionID,
		logger:      logger,
	}
}

// Read reads the last committed offset from the file.
// Returns the offset and true if the file exists, contains valid data, and the partition ID matches.
// Returns 0 and false if the file doesn't exist, contains invalid data, or the partition ID does not match.
func (f *OffsetFile) Read() (int64, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()

	data, err := os.ReadFile(f.filePath)
	if os.IsNotExist(err) {
		return 0, false
	}
	if err != nil {
		level.Error(f.logger).Log("msg", "failed to read offset file", "file", f.filePath, "err", err)
		return 0, false
	}

	var parsed offsetFileData
	if err := json.Unmarshal(data, &parsed); err != nil {
		level.Error(f.logger).Log("msg", "failed to parse offset file", "file", f.filePath, "err", err)
		return 0, false
	}
	if parsed.Version != offsetFileVersion {
		level.Error(f.logger).Log("msg", "offset file has unknown version", "file", f.filePath, "version", parsed.Version)
		return 0, false
	}
	if parsed.PartitionID != f.partitionID {
		level.Error(f.logger).Log("msg", "offset file partition ID does not match expected partition, wrong volume may be mounted", "file", f.filePath, "expected_partition", f.partitionID, "file_partition", parsed.PartitionID)
		return 0, false
	}

	level.Info(f.logger).Log("msg", "read last committed offset from file", "file", f.filePath, "offset", parsed.Offset)
	return parsed.Offset, true
}

// Write writes the offset to the file atomically.
func (f *OffsetFile) Write(offset int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	dir := filepath.Dir(f.filePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	data := offsetFileData{
		Version:     offsetFileVersion,
		PartitionID: f.partitionID,
		Offset:      offset,
	}
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal offset file data: %w", err)
	}

	if err := atomicfs.CreateFile(f.filePath, bytes.NewReader(jsonBytes)); err != nil {
		return fmt.Errorf("failed to write offset file: %w", err)
	}

	level.Debug(f.logger).Log("msg", "wrote offset to file", "file", f.filePath, "offset", offset)
	return nil
}
