package encoding

import (
	"context"
	"fmt"
	"io/fs"
	"time"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
	v2 "github.com/grafana/tempo/tempodb/encoding/v2"
	"github.com/grafana/tempo/tempodb/encoding/vparquet"
	"github.com/grafana/tempo/tempodb/encoding/vparquet2"
	"github.com/grafana/tempo/tempodb/encoding/vparquet3"
)

// VersionedEncoding represents a backend block version, and the methods to
// read/write them.
type VersionedEncoding interface {
	Version() string

	// OpenBlock for reading
	OpenBlock(meta *backend.BlockMeta, r backend.Reader) (common.BackendBlock, error)

	// NewCompactor creates a Compactor that can be used to combine blocks of this
	// encoding. It is expected to use internal details for efficiency.
	NewCompactor(common.CompactionOptions) common.Compactor

	// CreateBlock with the given attributes and trace contents.
	// BlockMeta is used as a container for many options. Required fields:
	// * BlockID
	// * TenantID
	// * Encoding
	// * DataEncoding
	// * StartTime
	// * EndTime
	// * TotalObjects
	CreateBlock(ctx context.Context, cfg *common.BlockConfig, meta *backend.BlockMeta, i common.Iterator, r backend.Reader, to backend.Writer) (*backend.BlockMeta, error)

	// CopyBlock from one backend to another.
	CopyBlock(ctx context.Context, meta *backend.BlockMeta, from backend.Reader, to backend.Writer) error

	// MigrateBlock from one backend and tenant to another.
	MigrateBlock(ctx context.Context, fromMeta, toMeta *backend.BlockMeta, from backend.Reader, to backend.Writer) error

	// OpenWALBlock opens an existing appendable block for the WAL
	OpenWALBlock(filename, path string, ingestionSlack, additionalStartSlack time.Duration) (common.WALBlock, error, error)

	// CreateWALBlock creates a new appendable block for the WAL
	// BlockMeta is used as a container for many options. Required fields:
	// * BlockID
	// * TenantID
	// * Encoding (v2)
	// * DataEncoding
	// * DedicatedColumns (vParquet3)
	// * ReplicationFactor (Optional)
	CreateWALBlock(meta *backend.BlockMeta, filepath string, ingestionSlack time.Duration) (common.WALBlock, error)

	// OwnsWALBlock indicates if this encoding owns the WAL block
	OwnsWALBlock(entry fs.DirEntry) bool
}

// FromVersion returns a versioned encoding for the provided string
func FromVersion(v string) (VersionedEncoding, error) {
	switch v {
	case v2.VersionString:
		return v2.Encoding{}, nil
	case vparquet.VersionString:
		return vparquet.Encoding{}, nil
	case vparquet2.VersionString:
		return vparquet2.Encoding{}, nil
	case vparquet3.VersionString:
		return vparquet3.Encoding{}, nil
	default:
		return nil, fmt.Errorf("%s is not a valid block version", v)
	}
}

// DefaultEncoding for newly written blocks.
func DefaultEncoding() VersionedEncoding {
	return vparquet3.Encoding{}
}

// LatestEncoding returns the most recent encoding.
func LatestEncoding() VersionedEncoding {
	return vparquet3.Encoding{}
}

// AllEncodings returns all encodings
func AllEncodings() []VersionedEncoding {
	return []VersionedEncoding{
		v2.Encoding{},
		vparquet2.Encoding{},
		vparquet3.Encoding{},
	}
}

// OpenBlock for reading in the backend. It automatically chooes the encoding for the given block.
func OpenBlock(meta *backend.BlockMeta, r backend.Reader) (common.BackendBlock, error) {
	v, err := FromVersion(meta.Version)
	if err != nil {
		return nil, err
	}
	return v.OpenBlock(meta, r)
}

// CopyBlock from one backend to another. It automatically chooses the encoding for the given block.
func CopyBlock(ctx context.Context, meta *backend.BlockMeta, from backend.Reader, to backend.Writer) error {
	v, err := FromVersion(meta.Version)
	if err != nil {
		return err
	}
	return v.CopyBlock(ctx, meta, from, to)
}
