package search

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	cortex_util "github.com/cortexproject/cortex/pkg/util/log"
	"github.com/go-kit/log/level"

	"github.com/grafana/tempo/pkg/tempofb"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/wal"
)

// RescanBlocks scans through the search directory in the WAL folder and replays files
// todo: copied from wal.RescanBlocks(), see if we can reduce duplication?
func RescanBlocks(walPath string) ([]*StreamingSearchBlock, error) {
	searchFilepath := filepath.Join(walPath, "search")
	files, err := ioutil.ReadDir(searchFilepath)
	if err != nil {
		// this might happen if search is not enabled, dont err here
		level.Warn(cortex_util.Logger).Log("msg", "failed to open search wal directory", "err", err)
		return nil, nil
	}

	blocks := make([]*StreamingSearchBlock, 0, len(files))
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		start := time.Now()
		level.Info(cortex_util.Logger).Log("msg", "beginning replay", "file", f.Name(), "size", f.Size())

		// pass the path to search subdirectory and filename
		// here f.Name() does not have full path
		b, warning, err := newStreamingSearchBlockFromWALReplay(searchFilepath, f.Name())

		remove := false
		if err != nil {
			// wal replay failed, clear and warn
			level.Warn(cortex_util.Logger).Log("msg", "failed to replay block. removing.", "file", f.Name(), "err", err)
			remove = true
		}

		if b != nil && b.appender.Length() == 0 {
			level.Warn(cortex_util.Logger).Log("msg", "empty wal file. ignoring.", "file", f.Name(), "err", err)
			remove = true
		}

		if warning != nil {
			level.Warn(cortex_util.Logger).Log("msg", "received warning while replaying block. partial replay likely.", "file", f.Name(), "warning", warning, "records", b.appender.Length())
		}

		if remove {
			err = os.Remove(filepath.Join(searchFilepath, f.Name()))
			if err != nil {
				return nil, err
			}
			continue
		}

		level.Info(cortex_util.Logger).Log("msg", "replay complete", "file", f.Name(), "duration", time.Since(start))

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

	blockID, _, version, enc, _, err := wal.ParseFilename(filename)
	if err != nil {
		return nil, nil, err
	}

	v, err := encoding.FromVersion(version)
	if err != nil {
		return nil, nil, err
	}

	blockHeader := tempofb.NewSearchBlockHeaderMutable()
	records, warning, err := wal.ReplayWALAndGetRecords(f, v, enc, func(bytes []byte) error {
		entry := tempofb.SearchEntryFromBytes(bytes)
		blockHeader.AddEntry(entry)
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return &StreamingSearchBlock{
		blockID:  blockID,
		file:     f,
		appender: encoding.NewRecordAppender(records),
		header:   blockHeader,
		v:        v,
		enc:      enc,
	}, warning, nil
}
