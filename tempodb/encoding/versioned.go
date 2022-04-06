package encoding

import (
	"context"
	"fmt"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
	v2 "github.com/grafana/tempo/tempodb/encoding/v2"
)

// VersionedEncoding has a whole bunch of versioned functionality.  This is
//  currently quite sloppy and could easily be tightened up to just a few methods
//  but it is what it is for now!
type VersionedEncoding interface {
	Version() string

	OpenBackendBlock(meta *backend.BlockMeta, r backend.Reader) (common.BackendBlock, error)

	NewCompactor() common.Compactor

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

func OpenBackendBlock(meta *backend.BlockMeta, r backend.Reader) (common.BackendBlock, error) {
	v, err := FromVersion(meta.Version)
	if err != nil {
		return nil, err
	}
	return v.OpenBackendBlock(meta, r)
}

func CopyBlock(ctx context.Context, meta *backend.BlockMeta, from backend.Reader, to backend.Writer) error {
	v, err := FromVersion(meta.Version)
	if err != nil {
		return err
	}
	return v.CopyBlock(ctx, meta, from, to)
}
