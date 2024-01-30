package inspect

import (
	"encoding/binary"
	"errors"
	"io"
	"os"

	"github.com/parquet-go/parquet-go"

	"github.com/stoewer/parquet-cli/pkg/output"
)

func NewFileInfo(file *os.File, pfile *parquet.File) (*FileInfo, error) {
	var info FileInfo

	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}

	info.Add("Name", file.Name())
	info.Add("Size", stat.Size())

	footer, err := footerSize(file)
	if err != nil {
		return nil, err
	}
	info.Add("Footer", footer)

	meta := pfile.Metadata()
	info.Add("Version", meta.Version)
	info.Add("Creator", meta.CreatedBy)
	info.Add("Rows", meta.NumRows)
	info.Add("RowGroups", len(meta.RowGroups))
	info.Add("Columns", len(pfile.ColumnIndexes()))

	return &info, nil
}

type FileInfo struct {
	elem map[string]any
	keys []string
	next int
}

type infoRow []any

func (i *FileInfo) Header() []any {
	return []any{"Key", "Value"}
}

func (i *FileInfo) NextRow() (output.TableRow, error) {
	if i.next >= len(i.keys) {
		return nil, io.EOF
	}

	key := i.keys[i.next]
	i.next++
	return infoRow{key, i.elem[key]}, nil
}

func (i *FileInfo) Add(k string, v any) {
	if i.elem == nil {
		i.elem = make(map[string]any)
	}
	i.elem[k] = v
	i.keys = append(i.keys, k)
}

func (i *FileInfo) Data() any {
	return i.elem
}

func (r infoRow) Cells() []any {
	return r
}

func footerSize(file *os.File) (uint32, error) {
	stat, err := file.Stat()
	if err != nil {
		return 0, err
	}

	buf := make([]byte, 8)
	n, err := file.ReadAt(buf, stat.Size()-8)
	if err != nil && !errors.Is(err, io.EOF) {
		return 0, err
	}
	if n < 4 {
		return 0, errors.New("not enough bytes read to determine footer size")
	}

	return binary.LittleEndian.Uint32(buf[0:4]), nil
}
