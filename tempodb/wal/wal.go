package wal

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

const (
	completedDir = "completed"
	blocksDir    = "blocks"
)

type WAL struct {
	c *Config
	l *local.Backend
}

type Config struct {
	Filepath          string `yaml:"path"`
	CompletedFilepath string
	BlocksFilepath    string
	Encoding          backend.Encoding `yaml:"v2_encoding"`
	SearchEncoding    backend.Encoding `yaml:"search_encoding"`
	IngestionSlack    time.Duration    `yaml:"ingestion_time_range_slack"`
	Version           string           `yaml:"version,omitempty"`
}

func ValidateConfig(c *Config) error {
	if _, err := encoding.FromVersion(c.Version); err != nil {
		return fmt.Errorf("failed to validate block version %s: %w", c.Version, err)
	}

	return nil
}

func New(c *Config) (*WAL, error) {
	if c.Filepath == "" {
		return nil, fmt.Errorf("please provide a path for the WAL")
	}

	// make folder
	err := os.MkdirAll(c.Filepath, os.ModePerm)
	if err != nil {
		return nil, err
	}

	// The /completed/ folder is now obsolete and no new data is written,
	// but it needs to be cleared out one last time for any files left
	// from a previous version.
	if c.CompletedFilepath == "" {
		completedFilepath := filepath.Join(c.Filepath, completedDir)
		err = os.RemoveAll(completedFilepath)
		if err != nil {
			return nil, err
		}

		c.CompletedFilepath = completedFilepath
	}

	// Setup local backend in /blocks/
	p := filepath.Join(c.Filepath, blocksDir)
	err = os.MkdirAll(p, os.ModePerm)
	if err != nil {
		return nil, err
	}
	c.BlocksFilepath = p

	l, err := local.NewBackend(&local.Config{
		Path: p,
	})
	if err != nil {
		return nil, err
	}

	return &WAL{
		c: c,
		l: l,
	}, nil
}

// RescanBlocks returns a slice of append blocks from the wal folder
func (w *WAL) RescanBlocks(additionalStartSlack time.Duration, log log.Logger) ([]common.WALBlock, error) {
	files, err := os.ReadDir(w.c.Filepath)
	if err != nil {
		return nil, err
	}

	encodings := encoding.AllEncodings()
	blocks := make([]common.WALBlock, 0, len(files))
	for _, f := range files {
		// find owner
		var owner encoding.VersionedEncoding
		for _, e := range encodings {
			if e.OwnsWALBlock(f) {
				owner = e
				break
			}
		}

		if owner == nil {
			level.Warn(log).Log("msg", "unowned file entry ignored during wal replay", "file", f.Name(), "err", err)
			continue
		}

		start := time.Now()
		fileInfo, err := f.Info()
		if err != nil {
			return nil, err
		}

		level.Info(log).Log("msg", "beginning replay", "file", f.Name(), "size", fileInfo.Size())
		b, warning, err := owner.OpenWALBlock(f.Name(), w.c.Filepath, w.c.IngestionSlack, additionalStartSlack)

		remove := false
		if err != nil {
			// wal replay failed, clear and warn
			level.Warn(log).Log("msg", "failed to replay block. removing.", "file", f.Name(), "err", err)
			remove = true
		}

		if b != nil && b.DataLength() == 0 {
			level.Warn(log).Log("msg", "empty wal file. ignoring.", "file", f.Name(), "err", err)
			remove = true
		}

		if warning != nil {
			level.Warn(log).Log("msg", "received warning while replaying block. partial replay likely.", "file", f.Name(), "warning", warning, "length", b.DataLength())
		}

		if remove {
			err = os.RemoveAll(filepath.Join(w.c.Filepath, f.Name()))
			if err != nil {
				return nil, err
			}
			continue
		}

		level.Info(log).Log("msg", "replay complete", "file", f.Name(), "duration", time.Since(start))

		blocks = append(blocks, b)
	}

	return blocks, nil
}

func (w *WAL) NewBlock(id uuid.UUID, tenantID string, dataEncoding string) (common.WALBlock, error) {
	return w.newBlock(id, tenantID, dataEncoding, w.c.Version)
}

func (w *WAL) newBlock(id uuid.UUID, tenantID string, dataEncoding string, blockVersion string) (common.WALBlock, error) {
	v, err := encoding.FromVersion(blockVersion)
	if err != nil {
		return nil, err
	}
	return v.CreateWALBlock(id, tenantID, w.c.Filepath, w.c.Encoding, dataEncoding, w.c.IngestionSlack)
}

func (w *WAL) GetFilepath() string {
	return w.c.Filepath
}

func (w *WAL) ClearFolder(dir string) error {
	p := filepath.Join(w.c.Filepath, dir)
	return os.RemoveAll(p)
}

func (w *WAL) LocalBackend() *local.Backend {
	return w.l
}
