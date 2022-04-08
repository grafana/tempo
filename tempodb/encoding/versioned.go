package encoding

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
	v2 "github.com/grafana/tempo/tempodb/encoding/v2"
)

// VersionedEncoding has a whole bunch of versioned functionality.  This is
//  currently quite sloppy and could easily be tightened up to just a few methods
//  but it is what it is for now!
type VersionedEncoding interface {
	Version() string

	OpenBlock(meta *backend.BlockMeta, r backend.Reader) (common.BackendBlock, error)

	NewCompactor() common.Compactor

	// CreateBlock with the trace contents of the iterator. The new block will have the same ID
	CreateBlock(ctx context.Context, cfg *common.BlockConfig, tenantID string, blockID uuid.UUID, encoding backend.Encoding, dataEncoding string, estimatedTotalObjects int, i common.TraceIterator, to backend.Writer) (*backend.BlockMeta, error)
	CopyBlock(ctx context.Context, meta *backend.BlockMeta, from backend.Reader, to backend.Writer) error
}

// FromVersion returns a versioned encoding for the provided string
func FromVersion(v string) (VersionedEncoding, error) {
	switch v {
	case "v2":
		return v2.Encoding{}, nil
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
	}
}

// These helpers choose the right encoding for the given block.

func OpenBlock(meta *backend.BlockMeta, r backend.Reader) (common.BackendBlock, error) {
	v, err := FromVersion(meta.Version)
	if err != nil {
		return nil, err
	}
	return v.OpenBlock(meta, r)
}

func CopyBlock(ctx context.Context, meta *backend.BlockMeta, from backend.Reader, to backend.Writer) error {
	v, err := FromVersion(meta.Version)
	if err != nil {
		return err
	}
	return v.CopyBlock(ctx, meta, from, to)
}
