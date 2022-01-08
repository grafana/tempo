package wal

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

const maxDataEncodingLength = 32

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

func newAppendBlock(id uuid.UUID, tenantID string, filepath string, e backend.Encoding, dataEncoding string) (*AppendBlock, error) {
	if strings.ContainsRune(dataEncoding, ':') ||
		len([]rune(dataEncoding)) > maxDataEncodingLength {
		return nil, fmt.Errorf("dataEncoding %s is invalid", dataEncoding)
	}

	v, err := encoding.FromVersion("v2") // let's pin wal files instead of tracking latest for safety
	if err != nil {
		return nil, err
	}

	h := &AppendBlock{
		encoding: v,
		meta:     backend.NewBlockMeta(tenantID, id, v.Version(), e, dataEncoding),
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
// be completed. It can return a warning or a fatal error
func newAppendBlockFromFile(filename string, path string) (*AppendBlock, error, error) {
	var warning error
	blockID, tenantID, version, e, dataEncoding, err := ParseFilename(filename)
	if err != nil {
		return nil, nil, err
	}

	v, err := encoding.FromVersion(version)
	if err != nil {
		return nil, nil, err
	}

	b := &AppendBlock{
		meta:     backend.NewBlockMeta(tenantID, blockID, version, e, dataEncoding),
		filepath: path,
		encoding: v,
	}

	// replay file to extract records
	f, err := b.file()
	if err != nil {
		return nil, nil, err
	}

	records, warning, err := ReplayWALAndGetRecords(f, v, e, func(bytes []byte) error {
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	b.appender = encoding.NewRecordAppender(records)
	b.meta.TotalObjects = b.appender.Length()

	return b, warning, nil
}

func (a *AppendBlock) Append(id common.ID, b []byte) error {
	err := a.appender.Append(id, b)
	if err != nil {
		return err
	}
	a.meta.ObjectAdded(id)
	return nil
}

func (a *AppendBlock) BlockID() uuid.UUID {
	return a.meta.BlockID
}

func (a *AppendBlock) DataLength() uint64 {
	return a.appender.DataLength()
}

func (a *AppendBlock) Meta() *backend.BlockMeta {
	return a.meta
}

func (a *AppendBlock) Iterator(combiner common.ObjectCombiner) (encoding.Iterator, error) {
	if a.appendFile != nil {
		err := a.appendFile.Close()
		if err != nil {
			return nil, err
		}
		a.appendFile = nil
	}

	records := a.appender.Records()
	readFile, err := a.file()
	if err != nil {
		return nil, err
	}

	dataReader, err := a.encoding.NewDataReader(backend.NewContextReaderWithAllReader(readFile), a.meta.Encoding)
	if err != nil {
		return nil, err
	}

	iterator := encoding.NewRecordIterator(records, dataReader, a.encoding.NewObjectReaderWriter())
	iterator, err = encoding.NewDedupingIterator(iterator, combiner, a.meta.DataEncoding)
	if err != nil {
		return nil, err
	}

	return iterator, nil
}

func (a *AppendBlock) Find(id common.ID, combiner common.ObjectCombiner) ([]byte, error) {
	records := a.appender.RecordsForID(id)
	file, err := a.file()
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, nil
	}

	dataReader, err := a.encoding.NewDataReader(backend.NewContextReaderWithAllReader(file), a.meta.Encoding)
	if err != nil {
		return nil, err
	}
	defer dataReader.Close()
	finder := encoding.NewPagedFinder(common.Records(records), dataReader, combiner, a.encoding.NewObjectReaderWriter(), a.meta.DataEncoding)

	return finder.Find(context.Background(), id)
}

func (a *AppendBlock) Clear() error {
	if a.readFile != nil {
		_ = a.readFile.Close()
		a.readFile = nil
	}

	if a.appendFile != nil {
		_ = a.appendFile.Close()
		a.appendFile = nil
	}

	// ignore error, it's important to remove the file above all else
	_ = a.appender.Complete()

	name := a.fullFilename()
	return os.Remove(name)
}

func (a *AppendBlock) fullFilename() string {
	if a.meta.Version == "v0" {
		return filepath.Join(a.filepath, fmt.Sprintf("%v:%v", a.meta.BlockID, a.meta.TenantID))
	}

	var filename string
	if a.meta.DataEncoding == "" {
		filename = fmt.Sprintf("%v:%v:%v:%v", a.meta.BlockID, a.meta.TenantID, a.meta.Version, a.meta.Encoding)
	} else {
		filename = fmt.Sprintf("%v:%v:%v:%v:%v", a.meta.BlockID, a.meta.TenantID, a.meta.Version, a.meta.Encoding, a.meta.DataEncoding)
	}

	return filepath.Join(a.filepath, filename)
}

func (a *AppendBlock) file() (*os.File, error) {
	var err error
	a.once.Do(func() {
		if a.readFile == nil {
			name := a.fullFilename()

			a.readFile, err = os.OpenFile(name, os.O_RDONLY, 0644)
		}
	})

	return a.readFile, err
}
