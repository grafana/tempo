package local

import (
	"context"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/grafana/tempo/tempodb/backend"
)

type Backend struct {
	cfg *Config
}

var _ backend.RawReader = (*Backend)(nil)
var _ backend.RawWriter = (*Backend)(nil)
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

func New(cfg *Config) (backend.RawReader, backend.RawWriter, backend.Compactor, error) {
	l, err := NewBackend(cfg)
	return l, l, l, err
}

// Write implements backend.Writer
func (rw *Backend) Write(ctx context.Context, name string, keypath backend.KeyPath, data io.Reader, _ int64) error {
	blockFolder := rw.rootPath(keypath)
	err := os.MkdirAll(blockFolder, os.ModePerm)
	if err != nil {
		return err
	}

	tracesFileName := rw.objectFileName(keypath, name)
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

// Append implements backend.Writer
func (rw *Backend) Append(ctx context.Context, name string, keypath backend.KeyPath, tracker backend.AppendTracker, buffer []byte) (backend.AppendTracker, error) {
	var dst *os.File
	if tracker == nil {
		blockFolder := rw.rootPath(keypath)
		err := os.MkdirAll(blockFolder, os.ModePerm)
		if err != nil {
			return nil, err
		}

		tracesFileName := rw.objectFileName(keypath, name)
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

// List implements backend.Reader
func (rw *Backend) List(ctx context.Context, keypath backend.KeyPath) ([]string, error) {
	path := rw.rootPath(keypath)
	folders, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}

	objects := make([]string, 0, len(folders))
	for _, f := range folders {
		if !f.IsDir() {
			continue
		}
		objects = append(objects, f.Name())
	}

	return objects, nil
}

// Read implements backend.Reader
func (rw *Backend) Read(ctx context.Context, name string, keypath backend.KeyPath) (io.ReadCloser, int64, error) {
	filename := rw.objectFileName(keypath, name)

	f, err := os.OpenFile(filename, os.O_RDONLY, 0644)
	if err != nil {
		return nil, -1, readError(err)
	}

	stat, err := f.Stat()
	if err != nil {
		return nil, -1, err
	}

	return f, stat.Size(), err
}

// ReadRange implements backend.Reader
func (rw *Backend) ReadRange(ctx context.Context, name string, keypath backend.KeyPath, offset uint64, buffer []byte) error {
	filename := rw.objectFileName(keypath, name)

	f, err := os.OpenFile(filename, os.O_RDONLY, 0644)
	if err != nil {
		return readError(err)
	}
	defer f.Close()

	_, err = f.ReadAt(buffer, int64(offset))
	if err != nil {
		return err
	}

	return nil
}

// Shutdown implements backend.Reader
func (rw *Backend) Shutdown() {

}

func (rw *Backend) objectFileName(keypath backend.KeyPath, name string) string {
	return filepath.Join(rw.rootPath(keypath), name)
}

func (rw *Backend) metaFileName(blockID uuid.UUID, tenantID string) string {
	return filepath.Join(rw.rootPath(backend.KeyPathForBlock(blockID, tenantID)), backend.MetaName)
}

func (rw *Backend) rootPath(keypath backend.KeyPath) string {
	return filepath.Join(rw.cfg.Path, filepath.Join(keypath...))
}

func readError(err error) error {
	if os.IsNotExist(err) {
		return backend.ErrDoesNotExist
	}

	return err
}
