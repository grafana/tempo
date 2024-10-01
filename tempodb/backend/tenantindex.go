package backend

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	proto "github.com/gogo/protobuf/proto"
	"github.com/klauspost/compress/gzip"
	"github.com/klauspost/compress/zstd"
)

const (
	internalFilename = "index.json"
)

var _ proto.Message = (*TenantIndex)(nil)

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
	buffer := &bytes.Buffer{}

	z, err := zstd.NewWriter(buffer)
	if err != nil {
		return nil, err
	}

	pbBytes, err := proto.Marshal(b)
	if err != nil {
		return nil, err
	}

	if _, err = z.Write(pbBytes); err != nil {
		return nil, err
	}
	if err = z.Flush(); err != nil {
		return nil, err
	}
	if err = z.Close(); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func (b *TenantIndex) unmarshalPb(buffer []byte) error {
	decoder, err := zstd.NewReader(nil, zstd.WithDecoderConcurrency(0))
	if err != nil {
		return fmt.Errorf("error creating zstd decoder: %w", err)
	}

	bb, err := decoder.DecodeAll(buffer, nil)
	if err != nil {
		return fmt.Errorf("error decoding zstd: %w", err)
	}

	return b.Unmarshal(bb)
}
