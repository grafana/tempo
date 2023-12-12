package local

import (
	"context"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/opentracing/opentracing-go"

	"github.com/grafana/tempo/tempodb/backend"
)

type Backend struct {
	cfg *Config
}

var (
	_ backend.RawReader = (*Backend)(nil)
	_ backend.RawWriter = (*Backend)(nil)
	_ backend.Compactor = (*Backend)(nil)
)

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
func (rw *Backend) Write(ctx context.Context, name string, keypath backend.KeyPath, data io.Reader, _ int64, _ *backend.CacheInfo) error {
	if err := ctx.Err(); err != nil {
		return err
	}

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
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	span, _ := opentracing.StartSpanFromContext(ctx, "local.Append", opentracing.Tags{
		"len": len(buffer),
	})
	defer span.Finish()

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
	if err := ctx.Err(); err != nil {
		return err
	}

	if tracker == nil {
		return nil
	}

	var dst *os.File = tracker.(*os.File)
	return dst.Close()
}

func (rw *Backend) Delete(ctx context.Context, name string, keypath backend.KeyPath, _ *backend.CacheInfo) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	path := rw.rootPath(append(keypath, name))
	return os.RemoveAll(path)
}

// List implements backend.Reader
func (rw *Backend) List(ctx context.Context, keypath backend.KeyPath) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	path := rw.rootPath(keypath)
	folders, err := os.ReadDir(path)
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

// ListBlocks implements backend.Reader
func (rw *Backend) ListBlocks(_ context.Context, tenant string) (metas []uuid.UUID, compactedMetas []uuid.UUID, err error) {
	rootPath := rw.rootPath(backend.KeyPath{tenant})
	fff := os.DirFS(rootPath)
	err = fs.WalkDir(fff, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		tenantFilePath := filepath.Join(tenant, path)

		parts := strings.Split(tenantFilePath, "/")
		// i.e: <tenantID/<blockID>/meta
		if len(parts) != 3 {
			return nil
		}

		if parts[2] != backend.MetaName && parts[2] != backend.CompactedMetaName {
			return nil
		}

		id, err := uuid.Parse(parts[1])
		if err != nil {
			return err
		}

		switch parts[2] {
		case backend.MetaName:
			metas = append(metas, id)
		case backend.CompactedMetaName:
			compactedMetas = append(compactedMetas, id)
		}

		return nil
	})

	return
}

// Read implements backend.Reader
func (rw *Backend) Read(ctx context.Context, name string, keypath backend.KeyPath, _ *backend.CacheInfo) (io.ReadCloser, int64, error) {
	if err := ctx.Err(); err != nil {
		return nil, -1, err
	}

	filename := rw.objectFileName(keypath, name)

	f, err := os.OpenFile(filename, os.O_RDONLY, 0o644)
	if err != nil {
		return nil, -1, readError(err)
	}

	stat, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, -1, err
	}

	return f, stat.Size(), err
}

// ReadRange implements backend.Reader
func (rw *Backend) ReadRange(ctx context.Context, name string, keypath backend.KeyPath, offset uint64, buffer []byte, _ *backend.CacheInfo) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	span, _ := opentracing.StartSpanFromContext(ctx, "local.ReadRange", opentracing.Tags{
		"len":    len(buffer),
		"offset": offset,
	})
	defer span.Finish()

	filename := rw.objectFileName(keypath, name)

	f, err := os.OpenFile(filename, os.O_RDONLY, 0o644)
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

// Shutdown implements backend.Reader. It attempts to clear all tenants
// that do not have blocks.
func (rw *Backend) Shutdown() {
	ctx := context.Background()

	// Shutdown() doesn't return error so this is best effort
	tenants, err := rw.List(ctx, backend.KeyPath{})
	if err != nil {
		return
	}

	for _, tenant := range tenants {
		blocks, err := rw.List(ctx, backend.KeyPath{tenant})
		if err != nil {
			continue
		}

		if len(blocks) == 0 {
			_ = os.RemoveAll(rw.rootPath(backend.KeyPath{tenant}))
		}
	}
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
