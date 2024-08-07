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

func newTenantIndex(meta []*backend_v1.BlockMeta, compactedMeta []*backend_v1.CompactedBlockMeta) *backend_v1.TenantIndex {
	i := &backend_v1.TenantIndex{
		CreatedAt:     time.Now().Unix(),
		Meta:          meta,
		CompactedMeta: compactedMeta,
	}

	return i
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
	tenantIndex := &backend_v1.TenantIndex{
		CreatedAt:     b.CreatedAt.Unix(),
		Meta:          make([]*backend_v1.BlockMeta, len(b.Meta)),
		CompactedMeta: make([]*backend_v1.CompactedBlockMeta, len(b.CompactedMeta)),
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
