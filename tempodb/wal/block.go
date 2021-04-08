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
	return filepath.Join(b.filepath, fmt.Sprintf("%v:%v", b.meta.BlockID, b.meta.TenantID))
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

func parseFilename(name string) (uuid.UUID, string, error) {
	i := strings.Index(name, ":")

	if i < 0 {
		return uuid.UUID{}, "", fmt.Errorf("unable to parse %s. no colon", name)
	}

	blockIDString := name[:i]
	tenantID := name[i+1:]

	blockID, err := uuid.Parse(blockIDString)
	if err != nil {
		return uuid.UUID{}, "", err
	}

	if len(tenantID) == 0 {
		return uuid.UUID{}, "", fmt.Errorf("unable to parse %s. empty tenant", name)
	}

	return blockID, tenantID, nil
}
