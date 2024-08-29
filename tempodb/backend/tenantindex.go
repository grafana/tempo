package backend

import (
	"bytes"
	"encoding/json"
	"time"

	backend_v1 "github.com/grafana/tempo/tempodb/backend/v1"
	"github.com/klauspost/compress/gzip"
)

const (
	internalFilename = "index.json"
)

// TenantIndex holds a list of all metas and compacted metas for a given tenant
// it is probably stored in /<tenantid>/blockindex.json.gz as a gzipped json file
type TenantIndex struct {
	CreatedAt     time.Time             `json:"created_at"`
	Meta          []*BlockMeta          `json:"meta"`
	CompactedMeta []*CompactedBlockMeta `json:"compacted"`
}

func newTenantIndex(meta []*BlockMeta, compactedMeta []*CompactedBlockMeta) *TenantIndex {
	return &TenantIndex{
		CreatedAt:     time.Now(),
		Meta:          meta,
		CompactedMeta: compactedMeta,
	}
}

// marshal converts to json and compresses the bucketindex
func (b *TenantIndex) marshal() ([]byte, error) {
	buffer := &bytes.Buffer{}

	gzip := gzip.NewWriter(buffer)
	gzip.Name = internalFilename

	jsonBytes, err := json.Marshal(b)
	if err != nil {
		return nil, err
	}

	if _, err = gzip.Write(jsonBytes); err != nil {
		return nil, err
	}
	if err = gzip.Flush(); err != nil {
		return nil, err
	}
	if err = gzip.Close(); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

// unmarshal decompresses and unmarshals the results from json
func (b *TenantIndex) unmarshal(buffer []byte) error {
	gzipReader, err := gzip.NewReader(bytes.NewReader(buffer))
	if err != nil {
		return err
	}
	defer gzipReader.Close()

	d := json.NewDecoder(gzipReader)
	return d.Decode(b)
}

func (b *TenantIndex) proto() (*backend_v1.TenantIndex, error) {
	// TODO: we ned to match the same nilness on proto and fromProto.  If we dont
	// have a meta or a compacted meta...
	// I think we set nil.
	// Tests pass without this, but it seems like it should be there.

	tenantIndex := &backend_v1.TenantIndex{
		CreatedAt:     b.CreatedAt,
		Meta:          make([]*backend_v1.BlockMeta, 0, len(b.Meta)),
		CompactedMeta: make([]*backend_v1.CompactedBlockMeta, 0, len(b.CompactedMeta)),
	}

	var (
		err error
		mPb *backend_v1.BlockMeta
		cPb *backend_v1.CompactedBlockMeta
	)

	for _, m := range b.Meta {
		mPb, err = m.ToBackendV1Proto()
		if err != nil {
			return nil, err
		}

		tenantIndex.Meta = append(tenantIndex.Meta, mPb)
	}

	for _, m := range b.CompactedMeta {
		cPb, err = m.ToBackendV1Proto()
		if err != nil {
			return nil, err
		}

		tenantIndex.CompactedMeta = append(tenantIndex.CompactedMeta, cPb)
	}

	return tenantIndex, nil
}

func (b *TenantIndex) fromProto(pb *backend_v1.TenantIndex) error {
	b.CreatedAt = pb.CreatedAt
	var (
		meta          = make([]*BlockMeta, len(pb.Meta))
		compactedMeta = make([]*CompactedBlockMeta, len(pb.CompactedMeta))
	)

	var (
		err error
		m   *BlockMeta
		c   *CompactedBlockMeta
	)

	for i, mPb := range pb.Meta {
		m = &BlockMeta{}
		err = m.FromBackendV1Proto(mPb)
		if err != nil {
			return err
		}
		meta[i] = m
	}

	if len(meta) > 0 {
		b.Meta = meta
	}

	for i, cPb := range pb.CompactedMeta {
		c = &CompactedBlockMeta{}
		err = c.FromBackendV1Proto(cPb)
		if err != nil {
			return err
		}
		compactedMeta[i] = c
	}

	if len(compactedMeta) > 0 {
		b.CompactedMeta = compactedMeta
	}

	return nil
}
