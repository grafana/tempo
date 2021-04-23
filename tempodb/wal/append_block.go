package wal

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

// AppendBlock is a block that is actively used to append new objects to.  It stores all data in the appendFile
// in the order it was received and an in memory sorted index.
type AppendBlock struct {
	meta     *backend.BlockMeta
	encoding encoding.VersionedEncoding

	appendFile *os.File
	appender   encoding.Appender

	filepath string
	readFile *os.File
	once     sync.Once
}

func newAppendBlock(id uuid.UUID, tenantID string, filepath string, e backend.Encoding) (*AppendBlock, error) {
	v, err := encoding.FromVersion("v2") // let's pin wal files instead of tracking latest for safety
	if err != nil {
		return nil, err
	}

	h := &AppendBlock{
		encoding: v,
		meta:     backend.NewBlockMeta(tenantID, id, v.Version(), e),
		filepath: filepath,
	}

	name := h.fullFilename()

	f, err := os.OpenFile(name, os.O_APPEND|os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return nil, err
	}
	h.appendFile = f

	dataWriter, err := h.encoding.NewDataWriter(f, e)
	if err != nil {
		return nil, err
	}

	h.appender = encoding.NewAppender(dataWriter)

	return h, nil
}

// newAppendBlockFromFile returns an AppendBlock that can not be appended to, but can
// be completed. It is meant for wal replay
func newAppendBlockFromFile(filename string, path string) (*AppendBlock, error) { // jpe test
	blockID, tenantID, version, e, err := parseFilename(filename)
	if err != nil {
		return nil, err
	}

	v, err := encoding.FromVersion(version)
	if err != nil {
		return nil, err
	}

	b := &AppendBlock{
		meta:     backend.NewBlockMeta(tenantID, blockID, version, e),
		filepath: path,
		encoding: v,
	}

	// replay file to extract records
	f, err := b.file()
	if err != nil {
		return nil, err
	}

	dataReader, err := b.encoding.NewDataReader(backend.NewContextReaderWithAllReader(f), b.meta.Encoding)
	if err != nil {
		return nil, err
	}
	defer dataReader.Close()

	var buffer []byte
	var records []*common.Record
	objectReader := b.encoding.NewObjectReaderWriter()
	currentOffset := uint64(0)
	for {
		buffer, pageLen, err := dataReader.NextPage(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		reader := bytes.NewReader(buffer)
		id, _, err := objectReader.UnmarshalObjectFromReader(reader)
		if err != nil {
			return nil, err
		}
		// wal should only ever have one object per page, test that here
		_, _, err = objectReader.UnmarshalObjectFromReader(reader)
		if err != io.EOF {
			return nil, fmt.Errorf("wal pages should only have one object: %w", err)
		}

		// make a copy so we don't hold onto the iterator buffer
		recordID := append([]byte(nil), id...)
		records = append(records, &common.Record{
			ID:     recordID,
			Start:  currentOffset,
			Length: pageLen,
		})
		currentOffset += uint64(pageLen)
	}

	common.SortRecords(records)

	b.appender = encoding.NewRecordAppender(records)

	return b, nil
}

func (h *AppendBlock) Write(id common.ID, b []byte) error {
	err := h.appender.Append(id, b)
	if err != nil {
		return err
	}
	h.meta.ObjectAdded(id)
	return nil
}

func (h *AppendBlock) BlockID() uuid.UUID {
	return h.meta.BlockID
}

func (h *AppendBlock) DataLength() uint64 {
	return h.appender.DataLength()
}

func (h *AppendBlock) Meta() *backend.BlockMeta {
	return h.meta
}

func (h *AppendBlock) GetIterator(combiner common.ObjectCombiner) (encoding.Iterator, error) {
	if h.appendFile != nil {
		err := h.appendFile.Close()
		if err != nil {
			return nil, err
		}
		h.appendFile = nil
	}

	records := h.appender.Records()
	readFile, err := h.file()
	if err != nil {
		return nil, err
	}

	dataReader, err := h.encoding.NewDataReader(backend.NewContextReaderWithAllReader(readFile), h.meta.Encoding)
	if err != nil {
		return nil, err
	}

	iterator := encoding.NewRecordIterator(records, dataReader, h.encoding.NewObjectReaderWriter())
	iterator, err = encoding.NewDedupingIterator(iterator, combiner)
	if err != nil {
		return nil, err
	}

	return iterator, nil
}

func (h *AppendBlock) Find(id common.ID, combiner common.ObjectCombiner) ([]byte, error) {
	records := h.appender.Records()
	file, err := h.file()
	if err != nil {
		return nil, err
	}

	dataReader, err := h.encoding.NewDataReader(backend.NewContextReaderWithAllReader(file), h.meta.Encoding)
	if err != nil {
		return nil, err
	}
	defer dataReader.Close()
	finder := encoding.NewPagedFinder(common.Records(records), dataReader, combiner, h.encoding.NewObjectReaderWriter())

	return finder.Find(context.Background(), id)
}

func (h *AppendBlock) Clear() error {
	if h.readFile != nil {
		_ = h.readFile.Close()
		h.readFile = nil
	}

	if h.appendFile != nil {
		_ = h.appendFile.Close()
		h.appendFile = nil
	}

	name := h.fullFilename()
	return os.Remove(name)
}

func (b *AppendBlock) fullFilename() string {
	if b.meta.Version == "v0" {
		return filepath.Join(b.filepath, fmt.Sprintf("%v:%v", b.meta.BlockID, b.meta.TenantID))
	}

	return filepath.Join(b.filepath, fmt.Sprintf("%v:%v:%v:%v", b.meta.BlockID, b.meta.TenantID, b.meta.Version, b.meta.Encoding))
}

func (b *AppendBlock) file() (*os.File, error) {
	var err error
	b.once.Do(func() {
		if b.readFile == nil {
			name := b.fullFilename()

			b.readFile, err = os.OpenFile(name, os.O_RDONLY, 0644)
		}
	})

	return b.readFile, err
}

func parseFilename(name string) (uuid.UUID, string, string, backend.Encoding, error) {
	splits := strings.Split(name, ":")

	if len(splits) != 2 && len(splits) != 4 {
		return uuid.UUID{}, "", "", backend.EncNone, fmt.Errorf("unable to parse %s. unexpected number of segments", name)
	}

	blockIDString := splits[0]
	tenantID := splits[1]

	version := "v0"
	encodingString := backend.EncNone.String()
	if len(splits) == 4 {
		version = splits[2]
		encodingString = splits[3]
	}

	blockID, err := uuid.Parse(blockIDString)
	if err != nil {
		return uuid.UUID{}, "", "", backend.EncNone, fmt.Errorf("unable to parse %s. error parsing uuid: %w", name, err)
	}

	encoding, err := backend.ParseEncoding(encodingString)
	if err != nil {
		return uuid.UUID{}, "", "", backend.EncNone, fmt.Errorf("unable to parse %s. error parsing encoding: %w", name, err)
	}

	if len(tenantID) == 0 || len(version) == 0 {
		return uuid.UUID{}, "", "", backend.EncNone, fmt.Errorf("unable to parse %s. missing fields", name)
	}

	return blockID, tenantID, version, encoding, nil
}
