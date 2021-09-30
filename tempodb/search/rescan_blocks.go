package search

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	cortex_util "github.com/cortexproject/cortex/pkg/util/log"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"

	"github.com/grafana/tempo/pkg/tempofb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/wal"
)

// RescanBlocks scans through the search directory in the WAL folder and replays files
// todo: copied from wal.RescanBlocks(), see if we can reduce duplication?
func RescanBlocks(walPath string) ([]*StreamingSearchBlock, error) {
	searchFilepath := filepath.Join(walPath, "search")
	files, err := ioutil.ReadDir(searchFilepath)
	if err != nil {
		return nil, err
	}

	blocks := make([]*StreamingSearchBlock, 0, len(files))
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		start := time.Now()
		level.Info(cortex_util.Logger).Log("msg", "beginning replay", "file", f.Name(), "size", f.Size())

		b, warning, err := newStreamingSearchBlockFromWALReplay(f.Name())

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
func newStreamingSearchBlockFromWALReplay(filename string) (*StreamingSearchBlock, error, error) {
	f, err := os.OpenFile(filename, os.O_RDONLY, 0644)
	if err != nil {
		return nil, nil, err
	}

	blockID, _, _, _, _, err := parseFilename(filename)
	if err != nil {
		return nil, nil, err
	}

	// version is pinned to v2 for now
	v, err := encoding.FromVersion("v2")
	if err != nil {
		return nil, nil, err
	}

	blockHeader := tempofb.NewSearchBlockHeaderMutable()
	records, warning, err := wal.ReplayWALAndGetRecords(f, v, backend.EncNone, func(bytes []byte) error {
		entry := tempofb.SearchEntryFromBytes(bytes)
		blockHeader.AddEntry(entry)
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return &StreamingSearchBlock{
		BlockID:  blockID,
		file:     f,
		appender: encoding.NewRecordAppender(records),
		header:   blockHeader,
		encoding: v,
	}, warning, nil
}

func parseFilename(filename string) (blockID uuid.UUID, tenantID string, version string, encoding string, dataEncoding string, err error) {
	splits := strings.Split(filename, ":")

	if len(splits) < 6 {
		return uuid.Nil, "", "", "", "", err
	}

	id, err := uuid.Parse(splits[0])
	if err != nil {
		return uuid.Nil, "", "", "", "", err
	}
	// todo: any other validation?
	return id, splits[1], splits[2], splits[3], splits[4], nil
}
