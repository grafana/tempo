package wal

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
	v2 "github.com/grafana/tempo/tempodb/encoding/v2"
)

const reasonOutsideIngestionSlack = "outside_ingestion_time_slack"

var (
	metricWarnings = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "warnings_total",
		Help:      "The total number of warnings per tenant with reason.",
	}, []string{"tenant", "reason"})
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
	Encoding          backend.Encoding `yaml:"encoding"`
	SearchEncoding    backend.Encoding `yaml:"search_encoding"`
	IngestionSlack    time.Duration    `yaml:"ingestion_time_range_slack"`
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
func (w *WAL) RescanBlocks(fn common.RangeFunc, additionalStartSlack time.Duration, log log.Logger) ([]common.AppendBlock, error) {
	files, err := os.ReadDir(w.c.Filepath)
	if err != nil {
		return nil, err
	}

	// todo: rescan blocks will need to detect if this is a vParquet or v2 wal file and choose the appropriate encoding
	v, err := encoding.FromVersion(v2.VersionString)
	if err != nil {
		return nil, fmt.Errorf("from version v2 failed %w", err)
	}

	blocks := make([]common.AppendBlock, 0, len(files))
	for _, f := range files {
		if f.IsDir() {
			continue
		}

		start := time.Now()
		fileInfo, err := f.Info()
		if err != nil {
			return nil, err
		}

		level.Info(log).Log("msg", "beginning replay", "file", f.Name(), "size", fileInfo.Size())
		b, warning, err := v.OpenAppendBlock(f.Name(), w.c.Filepath, w.c.IngestionSlack, additionalStartSlack, fn)

		remove := false
		if err != nil {
			// wal replay failed, clear and warn
			level.Warn(log).Log("msg", "failed to replay block. removing.", "file", f.Name(), "err", err)
			remove = true
		}

		if b != nil && b.Length() == 0 {
			level.Warn(log).Log("msg", "empty wal file. ignoring.", "file", f.Name(), "err", err)
			remove = true
		}

		if warning != nil {
			level.Warn(log).Log("msg", "received warning while replaying block. partial replay likely.", "file", f.Name(), "warning", warning, "length", b.DataLength())
		}

		if remove {
			err = os.Remove(filepath.Join(w.c.Filepath, f.Name()))
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

func (w *WAL) NewBlock(id uuid.UUID, tenantID string, dataEncoding string) (common.AppendBlock, error) {
	// todo: take version string and use here
	v, err := encoding.FromVersion(v2.VersionString)
	if err != nil {
		return nil, fmt.Errorf("from version v2 failed %w", err)
	}
	return v.CreateAppendBlock(id, tenantID, w.c.Filepath, w.c.Encoding, dataEncoding, w.c.IngestionSlack)
}

func (w *WAL) NewFile(blockid uuid.UUID, tenantid string, dir string) (*os.File, backend.Encoding, error) {
	// search WAL pinned to v2 for now
	walFileVersion := "v2"

	p := filepath.Join(w.c.Filepath, dir)
	err := os.MkdirAll(p, os.ModePerm)
	if err != nil {
		return nil, backend.EncNone, err
	}

	// blockID, tenantID, version, encoding (compression), dataEncoding
	filename := fmt.Sprintf("%v:%v:%v:%v:%v", blockid, tenantid, walFileVersion, w.c.SearchEncoding, "")
	file, err := os.OpenFile(filepath.Join(p, filename), os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, backend.EncNone, err
	}

	return file, w.c.SearchEncoding, nil
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
