package search

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-kit/log/level"
	"github.com/grafana/tempo/pkg/tempofb"
	"github.com/grafana/tempo/pkg/util/log"
	v2 "github.com/grafana/tempo/tempodb/encoding/v2"
)

// RescanBlocks scans through the search directory in the WAL folder and replays files
// todo: copied from wal.RescanBlocks(), see if we can reduce duplication?
func RescanBlocks(walPath string) ([]*StreamingSearchBlock, error) {
	searchFilepath := filepath.Join(walPath, "search")
	files, err := os.ReadDir(searchFilepath)
	if err != nil {
		// this might happen if search is not enabled, dont err here
		level.Warn(log.Logger).Log("msg", "failed to open search wal directory", "err", err)
		return nil, nil
	}

	blocks := make([]*StreamingSearchBlock, 0, len(files))
	for _, f := range files {
		info, err := f.Info()

		if err != nil {
			return nil, fmt.Errorf("failed to file info %s, %w", f.Name(), err)
		}

		if info.IsDir() {
			continue
		}
		start := time.Now()
		level.Info(log.Logger).Log("msg", "beginning replay", "file", info.Name(), "size", info.Size())

		// pass the path to search subdirectory and filename
		// here info.Name() does not have full path
		b, warning, err := newStreamingSearchBlockFromWALReplay(searchFilepath, info.Name())

		remove := false
		if err != nil {
			// wal replay failed, clear and warn
			level.Warn(log.Logger).Log("msg", "failed to replay block. removing.", "file", info.Name(), "err", err)
			remove = true
		}

		if b != nil && b.appender.Length() == 0 {
			level.Warn(log.Logger).Log("msg", "empty wal file. ignoring.", "file", info.Name(), "err", err)
			remove = true
		}

		if warning != nil {
			level.Warn(log.Logger).Log("msg", "received warning while replaying block. partial replay likely.", "file", info.Name(), "warning", warning, "records", b.appender.Length())
		}

		if remove {
			err = os.Remove(filepath.Join(searchFilepath, info.Name()))
			if err != nil {
				return nil, err
			}
			continue
		}

		level.Info(log.Logger).Log("msg", "replay complete", "file", info.Name(), "duration", time.Since(start))

		blocks = append(blocks, b)
	}

	return blocks, nil
}

// newStreamingSearchBlockFromWALReplay creates a StreamingSearchBlock with in-memory records from a search WAL file
func newStreamingSearchBlockFromWALReplay(searchFilepath, filename string) (*StreamingSearchBlock, error, error) {
	f, err := os.OpenFile(filepath.Join(searchFilepath, filename), os.O_RDONLY, 0644)
	if err != nil {
		return nil, nil, err
	}

	blockID, _, _, enc, _, err := v2.ParseFilename(filename)
	if err != nil {
		return nil, nil, err
	}

	blockHeader := tempofb.NewSearchBlockHeaderMutable()
	records, warning, err := v2.ReplayWALAndGetRecords(f, enc, func(bytes []byte) error {
		entry := tempofb.NewSearchEntryFromBytes(bytes)
		blockHeader.AddEntry(entry)
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return &StreamingSearchBlock{
		blockID:  blockID,
		file:     f,
		appender: v2.NewRecordAppender(records),
		header:   blockHeader,
		enc:      enc,
	}, warning, nil
}
