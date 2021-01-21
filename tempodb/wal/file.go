package wal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

// File represents a walfile on disk.  It is used for wal replay
type File struct {
	Filepath string
	TenantID string
	BlockID  uuid.UUID
}

func newFile(filename string, filepath string) (*File, error) {
	block, tenant, err := parseFilename(filename)
	if err != nil {
		return nil, err
	}

	return &File{
		Filepath: filepath,
		BlockID:  block,
		TenantID: tenant,
	}, nil
}

// Clear removes the wal file that this struct represents
func (f *File) Clear() error {
	name := fullFilename(f)
	return os.Remove(name)
}

func parseFilename(name string) (uuid.UUID, string, error) {
	i := strings.Index(name, ":")

	if i < 0 {
		return uuid.UUID{}, "", fmt.Errorf("unable to parse %s", name)
	}

	blockIDString := name[:i]
	tenantID := name[i+1:]

	blockID, err := uuid.Parse(blockIDString)
	if err != nil {
		return uuid.UUID{}, "", err
	}

	return blockID, tenantID, nil
}

func fullFilename(f *File) string {
	return filepath.Join(f.Filepath, f.BlockID.String()+":"+f.TenantID)
}
