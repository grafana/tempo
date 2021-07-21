package backend

import (
	"bytes"
	"encoding/json"

	"github.com/klauspost/compress/gzip"
)

const (
	internalFilename = "index.json"
)

// tenantindex holds a list of all metas and compacted metas for a given tenant
// it is probably stored in /<tenantid>/blockindex.json.gz as a gzipped json file
type tenantindex struct {
	Meta          []*BlockMeta          `json:"meta"` // jpe add creation time
	CompactedMeta []*CompactedBlockMeta `json:"compacted"`
}

// marshal converts to json and compresses the bucketindex
func (b *tenantindex) marshal() ([]byte, error) {
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
	defer gzip.Close()

	return buffer.Bytes(), nil
}

// unmarshal decompresses and unmarshals the results from json
func (b *tenantindex) unmarshal(buffer []byte) error {
	gzipReader, err := gzip.NewReader(bytes.NewReader(buffer))
	if err != nil {
		return err
	}
	defer gzipReader.Close()

	d := json.NewDecoder(gzipReader)
	if err = d.Decode(b); err != nil {
		return err
	}

	return nil
}
