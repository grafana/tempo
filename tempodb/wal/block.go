package wal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/grafana/tempo/tempodb/backend"
)

type block struct {
	meta     *backend.BlockMeta
	filepath string
	readFile *os.File

	once sync.Once
}

func (b *block) fullFilename() string {
	if b.meta.Version == "v0" {
		return filepath.Join(b.filepath, fmt.Sprintf("%v:%v", b.meta.BlockID, b.meta.TenantID))
	} else {
		return filepath.Join(b.filepath, fmt.Sprintf("%v:%v:%v:%v", b.meta.BlockID, b.meta.TenantID, b.meta.Version, b.meta.Encoding))
	}
}

func (b *block) file() (*os.File, error) {
	var err error
	b.once.Do(func() {
		if b.readFile == nil {
			name := b.fullFilename()

			b.readFile, err = os.OpenFile(name, os.O_RDONLY, 0644)
		}
	})

	return b.readFile, err
}

func parseFilename(name string) (uuid.UUID, string, string, backend.Encoding, error) {
	splits := strings.Split(name, ":")

	if len(splits) != 2 && len(splits) != 4 {
		return uuid.UUID{}, "", "", backend.EncNone, fmt.Errorf("unable to parse %s. unexpected number of segments", name)
	}

	blockIDString := splits[0]
	tenantID := splits[1]

	version := "v0"
	encodingString := backend.EncNone.String()
	if len(splits) == 4 {
		version = splits[2]
		encodingString = splits[3]
	}

	blockID, err := uuid.Parse(blockIDString)
	if err != nil {
		return uuid.UUID{}, "", "", backend.EncNone, fmt.Errorf("unable to parse %s. error parsing uuid: %w", name, err)
	}

	encoding, err := backend.ParseEncoding(encodingString)
	if err != nil {
		return uuid.UUID{}, "", "", backend.EncNone, fmt.Errorf("unable to parse %s. error parsing encoding: %w", name, err)
	}

	if len(tenantID) == 0 || len(version) == 0 {
		return uuid.UUID{}, "", "", backend.EncNone, fmt.Errorf("unable to parse %s. missing fields", name)
	}

	return blockID, tenantID, version, encoding, nil
}
