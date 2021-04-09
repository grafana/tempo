package wal

import (
	"os"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
)

type ReplayBlock struct {
	block
	encoding encoding.VersionedEncoding
}

func NewReplayBlock(filename string, path string) (*ReplayBlock, error) {
	blockID, tenantID, version, e, err := parseFilename(filename)
	if err != nil {
		return nil, err
	}

	v, err := encoding.FromVersion(version)
	if err != nil {
		return nil, err
	}

	return &ReplayBlock{
		block: block{
			meta:     backend.NewBlockMeta(tenantID, blockID, version, e),
			filepath: path,
		},
		encoding: v,
	}, nil
}

func (r *ReplayBlock) Iterator() (encoding.Iterator, error) {
	name := r.fullFilename()
	f, err := os.OpenFile(name, os.O_RDONLY, 0644)
	if err != nil {
		return nil, err
	}

	dataReader, err := r.encoding.NewDataReader(backend.NewContextReaderWithAllReader(f), r.meta.Encoding)
	if err != nil {
		return nil, err
	}

	return encoding.NewWALIterator(dataReader, r.encoding.NewObjectReaderWriter()), nil
}

func (r *ReplayBlock) TenantID() string {
	return r.meta.TenantID
}

func (r *ReplayBlock) BlockID() string {
	return r.meta.BlockID.String()
}

func (r *ReplayBlock) Clear() error {
	if r.readFile != nil {
		_ = r.readFile.Close()
	}

	name := r.fullFilename()
	return os.Remove(name)
}
