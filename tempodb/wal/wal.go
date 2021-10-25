package wal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/google/uuid"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	versioned_encoding "github.com/grafana/tempo/tempodb/encoding"
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
func (w *WAL) RescanBlocks(log log.Logger) ([]*AppendBlock, error) {
	files, err := os.ReadDir(w.c.Filepath)
	if err != nil {
		return nil, err
	}

	blocks := make([]*AppendBlock, 0, len(files))
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
		b, warning, err := newAppendBlockFromFile(f.Name(), w.c.Filepath)

		remove := false
		if err != nil {
			// wal replay failed, clear and warn
			level.Warn(log).Log("msg", "failed to replay block. removing.", "file", f.Name(), "err", err)
			remove = true
		}

		if b != nil && b.appender.Length() == 0 {
			level.Warn(log).Log("msg", "empty wal file. ignoring.", "file", f.Name(), "err", err)
			remove = true
		}

		if warning != nil {
			level.Warn(log).Log("msg", "received warning while replaying block. partial replay likely.", "file", f.Name(), "warning", warning, "records", b.appender.Length())
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

func (w *WAL) NewBlock(id uuid.UUID, tenantID string, dataEncoding string) (*AppendBlock, error) {
	return newAppendBlock(id, tenantID, w.c.Filepath, w.c.Encoding, dataEncoding)
}

func (w *WAL) NewFile(blockid uuid.UUID, tenantid string, dir string) (*os.File, string, backend.Encoding, error) {
	// search WAL pinned to v2 for now
	walFileVersion := "v2"

	p := filepath.Join(w.c.Filepath, dir)
	err := os.MkdirAll(p, os.ModePerm)
	if err != nil {
		return nil, "", backend.EncNone, err
	}

	// blockID, tenantID, version, encoding (compression), dataEncoding
	filename := fmt.Sprintf("%v:%v:%v:%v:%v", blockid, tenantid, walFileVersion, w.c.SearchEncoding, "")
	file, err := os.OpenFile(filepath.Join(p, filename), os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, "", backend.EncNone, err
	}

	return file, walFileVersion, w.c.SearchEncoding, nil
}

// ParseFilename returns (blockID, tenant, version, encoding, dataEncoding, error).
// Example: "00000000-0000-0000-0000-000000000000:1:v2:snappy:v1"
func ParseFilename(filename string) (uuid.UUID, string, string, backend.Encoding, string, error) {
	splits := strings.Split(filename, ":")

	if len(splits) != 4 && len(splits) != 5 {
		return uuid.UUID{}, "", "", backend.EncNone, "", fmt.Errorf("unable to parse %s. unexpected number of segments", filename)
	}

	// first segment is blockID
	id, err := uuid.Parse(splits[0])
	if err != nil {
		return uuid.UUID{}, "", "", backend.EncNone, "", fmt.Errorf("unable to parse %s. error parsing uuid: %w", filename, err)
	}

	// second segment is tenant
	tenant := splits[1]
	if len(tenant) == 0 {
		return uuid.UUID{}, "", "", backend.EncNone, "", fmt.Errorf("unable to parse %s. missing fields", filename)
	}

	// third segment is version
	version := splits[2]
	_, err = versioned_encoding.FromVersion(version)
	if err != nil {
		return uuid.UUID{}, "", "", backend.EncNone, "", fmt.Errorf("unable to parse %s. error parsing version: %w", filename, err)
	}

	// fourth is encoding
	encodingString := splits[3]
	encoding, err := backend.ParseEncoding(encodingString)
	if err != nil {
		return uuid.UUID{}, "", "", backend.EncNone, "", fmt.Errorf("unable to parse %s. error parsing encoding: %w", filename, err)
	}

	// fifth is dataEncoding
	dataEncoding := ""
	if len(splits) == 5 {
		dataEncoding = splits[4]
	}

	return id, tenant, version, encoding, dataEncoding, nil
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
