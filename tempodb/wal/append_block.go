package wal

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
	v2 "github.com/grafana/tempo/tempodb/encoding/v2"
)

const maxDataEncodingLength = 32

// AppendBlock is a block that is actively used to append new objects to.  It stores all data in the appendFile
// in the order it was received and an in memory sorted index.
type AppendBlock struct {
	meta           *backend.BlockMeta
	encoding       encoding.VersionedEncoding
	ingestionSlack time.Duration

	appendFile *os.File
	appender   v2.Appender

	filepath string
	readFile *os.File
	once     sync.Once
}

func newAppendBlock(id uuid.UUID, tenantID string, filepath string, e backend.Encoding, dataEncoding string, ingestionSlack time.Duration) (*AppendBlock, error) {
	if strings.ContainsRune(dataEncoding, ':') ||
		len([]rune(dataEncoding)) > maxDataEncodingLength {
		return nil, fmt.Errorf("dataEncoding %s is invalid", dataEncoding)
	}

	v, err := encoding.FromVersion("v2") // let's pin wal files instead of tracking latest for safety
	if err != nil {
		return nil, err
	}

	h := &AppendBlock{
		encoding:       v,
		meta:           backend.NewBlockMeta(tenantID, id, v.Version(), e, dataEncoding),
		filepath:       filepath,
		ingestionSlack: ingestionSlack,
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

	h.appender = v2.NewAppender(dataWriter)

	return h, nil
}

// newAppendBlockFromFile returns an AppendBlock that can not be appended to, but can
// be completed. It can return a warning or a fatal error
func newAppendBlockFromFile(filename string, path string, ingestionSlack time.Duration, additionalStartSlack time.Duration, fn RangeFunc) (*AppendBlock, error, error) {
	var warning error
	blockID, tenantID, version, e, dataEncoding, err := ParseFilename(filename)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing wal filename: %w", err)
	}

	v, err := encoding.FromVersion(version)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing version: %w", err)
	}

	b := &AppendBlock{
		meta:           backend.NewBlockMeta(tenantID, blockID, version, e, dataEncoding),
		filepath:       path,
		encoding:       v,
		ingestionSlack: ingestionSlack,
	}

	// replay file to extract records
	f, err := b.file()
	if err != nil {
		return nil, nil, fmt.Errorf("accessing file: %w", err)
	}

	blockStart := uint32(math.MaxUint32)
	blockEnd := uint32(0)

	records, warning, err := ReplayWALAndGetRecords(f, v, e, func(bytes []byte) error {
		start, end, err := fn(bytes, dataEncoding)
		if err != nil {
			return err
		}
		start, end = b.adjustTimeRangeForSlack(start, end, additionalStartSlack)
		if start < blockStart {
			blockStart = start
		}
		if end > blockEnd {
			blockEnd = end
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	b.appender = v2.NewRecordAppender(records)
	b.meta.TotalObjects = b.appender.Length()
	b.meta.StartTime = time.Unix(int64(blockStart), 0)
	b.meta.EndTime = time.Unix(int64(blockEnd), 0)

	return b, warning, nil
}

// Append adds an id and object to this wal block. start/end should indicate the time range
// associated with the past object. They are unix epoch seconds.
func (a *AppendBlock) Append(id common.ID, b []byte, start, end uint32) error {
	err := a.appender.Append(id, b)
	if err != nil {
		return err
	}
	start, end = a.adjustTimeRangeForSlack(start, end, 0)
	a.meta.ObjectAdded(id, start, end)
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

func (a *AppendBlock) Iterator(combiner model.ObjectCombiner) (v2.Iterator, error) {
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

	iterator := v2.NewRecordIterator(records, dataReader, a.encoding.NewObjectReaderWriter())
	iterator, err = v2.NewDedupingIterator(iterator, combiner, a.meta.DataEncoding)
	if err != nil {
		return nil, err
	}

	return iterator, nil
}

func (a *AppendBlock) Find(id common.ID, combiner model.ObjectCombiner) ([]byte, error) {
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
	finder := v2.NewPagedFinder(common.Records(records), dataReader, combiner, a.encoding.NewObjectReaderWriter(), a.meta.DataEncoding)

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

func (a *AppendBlock) adjustTimeRangeForSlack(start uint32, end uint32, additionalStartSlack time.Duration) (uint32, uint32) {
	now := time.Now()
	startOfRange := uint32(now.Add(-a.ingestionSlack).Add(-additionalStartSlack).Unix())
	endOfRange := uint32(now.Add(a.ingestionSlack).Unix())

	warn := false
	if start < startOfRange {
		warn = true
		start = uint32(now.Unix())
	}
	if end > endOfRange {
		warn = true
		end = uint32(now.Unix())
	}

	if warn {
		metricWarnings.WithLabelValues(a.meta.TenantID, reasonOutsideIngestionSlack).Inc()
	}

	return start, end
}
