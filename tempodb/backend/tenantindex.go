package backend

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	proto "github.com/gogo/protobuf/proto"
	"github.com/klauspost/compress/gzip"
)

const (
	internalFilename = "index.json"
)

var (
	_    proto.Message = (*TenantIndex)(nil)
	Zstd               = &ZstdCodec{}
)

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

func (b *TenantIndex) marshalPb() ([]byte, error) {
	pbBytes, err := proto.Marshal(b)
	if err != nil {
		return nil, err
	}

	buffer := []byte{}
	return Zstd.Encode(pbBytes, buffer)
}

func (b *TenantIndex) unmarshalPb(buffer []byte) error {
	bb, err := Zstd.Decode(buffer)
	if err != nil {
		return fmt.Errorf("error decoding zstd: %w", err)
	}

	return b.Unmarshal(bb)
}
