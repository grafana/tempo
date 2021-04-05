package local

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/grafana/tempo/tempodb/backend"
)

type Backend struct {
	cfg *Config
}

var _ backend.Reader = (*Backend)(nil)
var _ backend.Writer = (*Backend)(nil)
var _ backend.Compactor = (*Backend)(nil)

func NewBackend(cfg *Config) (*Backend, error) {
	err := os.MkdirAll(cfg.Path, os.ModePerm)
	if err != nil {
		return nil, err
	}

	l := &Backend{
		cfg: cfg,
	}

	return l, nil
}

func New(cfg *Config) (backend.Reader, backend.Writer, backend.Compactor, error) {
	l, err := NewBackend(cfg)
	return l, l, l, err
}

// Write implements backend.Writer
func (rw *Backend) Write(ctx context.Context, name string, blockID uuid.UUID, tenantID string, buffer []byte) error {
	return rw.WriteReader(ctx, name, blockID, tenantID, bytes.NewBuffer(buffer), 0)
}

// WriteReader implements backend.Writer
func (rw *Backend) WriteReader(ctx context.Context, name string, blockID uuid.UUID, tenantID string, data io.Reader, _ int64) error {
	blockFolder := rw.rootPath(blockID, tenantID)
	err := os.MkdirAll(blockFolder, os.ModePerm)
	if err != nil {
		return err
	}

	tracesFileName := rw.objectFileName(blockID, tenantID, name)
	dst, err := os.Create(tracesFileName)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, data)
	if err != nil {
		return err
	}
	return err
}

// WriteBlockMeta implements backend.Writer
func (rw *Backend) WriteBlockMeta(ctx context.Context, meta *backend.BlockMeta) error {
	blockID := meta.BlockID
	tenantID := meta.TenantID

	blockFolder := rw.rootPath(blockID, tenantID)
	err := os.MkdirAll(blockFolder, os.ModePerm)
	if err != nil {
		return err
	}

	bMeta, err := json.Marshal(meta)
	if err != nil {
		return err
	}

	metaFileName := rw.metaFileName(blockID, tenantID)
	err = ioutil.WriteFile(metaFileName, bMeta, 0644)
	if err != nil {
		return err
	}

	return nil
}

// Append implements backend.Writer
func (rw *Backend) Append(ctx context.Context, name string, blockID uuid.UUID, tenantID string, tracker backend.AppendTracker, buffer []byte) (backend.AppendTracker, error) {
	var dst *os.File
	if tracker == nil {
		blockFolder := rw.rootPath(blockID, tenantID)
		err := os.MkdirAll(blockFolder, os.ModePerm)
		if err != nil {
			return nil, err
		}

		tracesFileName := rw.objectFileName(blockID, tenantID, name)
		dst, err = os.Create(tracesFileName)
		if err != nil {
			return nil, err
		}
	} else {
		dst = tracker.(*os.File)
	}

	_, err := dst.Write(buffer)
	if err != nil {
		return nil, err
	}

	return dst, nil
}

// CloseAppend implements backend.Writer
func (rw *Backend) CloseAppend(ctx context.Context, tracker backend.AppendTracker) error {
	if tracker == nil {
		return nil
	}

	var dst *os.File = tracker.(*os.File)
	return dst.Close()
}

// Tenants implements backend.Reader
func (rw *Backend) Tenants(ctx context.Context) ([]string, error) {
	folders, err := ioutil.ReadDir(rw.cfg.Path)
	if err != nil {
		return nil, err
	}

	tenants := make([]string, 0, len(folders))
	for _, f := range folders {
		if !f.IsDir() {
			continue
		}
		tenants = append(tenants, f.Name())
	}

	return tenants, nil
}

// Blocks implements backend.Reader
func (rw *Backend) Blocks(ctx context.Context, tenantID string) ([]uuid.UUID, error) {
	var warning error
	path := path.Join(rw.cfg.Path, tenantID)
	folders, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}

	blocks := make([]uuid.UUID, 0, len(folders))
	for _, f := range folders {
		if !f.IsDir() {
			continue
		}
		blockID, err := uuid.Parse(f.Name())
		if err != nil {
			warning = err
			continue
		}
		blocks = append(blocks, blockID)
	}

	return blocks, warning
}

// BlockMeta implements backend.Reader
func (rw *Backend) BlockMeta(ctx context.Context, blockID uuid.UUID, tenantID string) (*backend.BlockMeta, error) {
	filename := rw.metaFileName(blockID, tenantID)
	bytes, err := ioutil.ReadFile(filename)
	if os.IsNotExist(err) {
		return nil, backend.ErrMetaDoesNotExist
	}
	if err != nil {
		return nil, err
	}

	out := &backend.BlockMeta{}
	err = json.Unmarshal(bytes, out)
	if err != nil {
		return nil, err
	}

	return out, nil
}

// Read implements backend.Reader
func (rw *Backend) Read(ctx context.Context, name string, blockID uuid.UUID, tenantID string) ([]byte, error) {
	filename := rw.objectFileName(blockID, tenantID, name)
	return ioutil.ReadFile(filename)
}

// ReadRange implements backend.Reader
func (rw *Backend) ReadRange(ctx context.Context, name string, blockID uuid.UUID, tenantID string, offset uint64, buffer []byte) error {
	filename := rw.objectFileName(blockID, tenantID, name)

	f, err := os.OpenFile(filename, os.O_RDONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.ReadAt(buffer, int64(offset))
	if err != nil {
		return err
	}

	return nil
}

func (rw *Backend) ReadReader(ctx context.Context, name string, blockID uuid.UUID, tenantID string) (io.ReadCloser, int64, error) {
	filename := rw.objectFileName(blockID, tenantID, name)

	f, err := os.OpenFile(filename, os.O_RDONLY, 0644)
	if err != nil {
		return nil, -1, err
	}

	stat, err := f.Stat()
	if err != nil {
		return nil, -1, err
	}

	return f, stat.Size(), err
}

// Shutdown implements backend.Reader
func (rw *Backend) Shutdown() {

}

func (rw *Backend) objectFileName(blockID uuid.UUID, tenantID string, name string) string {
	return filepath.Join(rw.rootPath(blockID, tenantID), name)
}

func (rw *Backend) metaFileName(blockID uuid.UUID, tenantID string) string {
	return filepath.Join(rw.rootPath(blockID, tenantID), "meta.json")
}

func (rw *Backend) rootPath(blockID uuid.UUID, tenantID string) string {
	return filepath.Join(rw.cfg.Path, tenantID, blockID.String())
}
