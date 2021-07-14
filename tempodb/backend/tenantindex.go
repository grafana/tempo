package backend

import (
	"bytes"
	"encoding/json"
	"errors"

	"github.com/klauspost/compress/gzip"
)

const (
	internalFilename = "index.json"
)

// tenantindex holds a list of all metas and compacted metas for a given tenant
// it is probably stored in /<tenantid>/blockindex.json.gz as a gzipped json file
type tenantindex struct {
	Tenant        string                `json:"tenant"` // jpe -remove
	Meta          []*BlockMeta          `json:"meta"`
	CompactedMeta []*CompactedBlockMeta `json:"compacted"`
}

// marshal converts to json and compresses the bucketindex
func (b *tenantindex) marshal() ([]byte, error) {
	if b.Tenant == "" {
		return nil, errors.New("empty tenant found while marshalling index")
	}

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
		return err // jpe return "index corrupted"?
	}
	defer gzipReader.Close()

	d := json.NewDecoder(gzipReader)
	if err = d.Decode(b); err != nil {
		return err
	}

	return nil
}
