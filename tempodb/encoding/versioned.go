package encoding

import (
	"context"
	"fmt"

	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
	v2 "github.com/grafana/tempo/tempodb/encoding/v2"
	"github.com/grafana/tempo/tempodb/encoding/vparquet"
)

// VersionedEncoding has a whole bunch of versioned functionality.  This is
//  currently quite sloppy and could easily be tightened up to just a few methods
//  but it is what it is for now!
type VersionedEncoding interface {
	Version() string

	// OpenBlock for reading
	OpenBlock(meta *backend.BlockMeta, r backend.Reader) (common.BackendBlock, error)

	// NewCompactor creates a Compactor that can be used to combine blocks of this
	// encoding. It is expected to use internal details for efficiency.
	NewCompactor() common.Compactor

	// CreateBlock with the given attributes and trace contents.
	// BlockMeta is used as a container for many options. Required fields:
	// * BlockID
	// * TenantID
	// * Encoding
	// * DataEncoding
	// * StartTime
	// * EndTime
	// * TotalObjects
	CreateBlock(ctx context.Context, cfg *common.BlockConfig, meta *backend.BlockMeta, i common.Iterator, dec model.ObjectDecoder, r backend.Reader, to backend.Writer) (*backend.BlockMeta, error)

	// CopyBlock from one backend to another.
	CopyBlock(ctx context.Context, meta *backend.BlockMeta, from backend.Reader, to backend.Writer) error
}

// FromVersion returns a versioned encoding for the provided string
func FromVersion(v string) (VersionedEncoding, error) {
	switch v {
	case "v2":
		return v2.Encoding{}, nil
	case vparquet.VersionString:
		return vparquet.Encoding{}, nil
	}

	return nil, fmt.Errorf("%s is not a valid block version", v)
}

// LatestEncoding is used by Compactor and Complete block
func LatestEncoding() VersionedEncoding {
	return v2.Encoding{}
}

// allEncodings returns all encodings
func allEncodings() []VersionedEncoding {
	return []VersionedEncoding{
		v2.Encoding{},
		vparquet.Encoding{},
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
