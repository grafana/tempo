package backend

import (
	"context"
	"io"

	"github.com/grafana/tempo/pkg/util"
)

// ContextReader is an io.ReaderAt interface that passes context.  It is used to simplify access to backend objects
// and abstract away the name/meta and other details so that the data can be accessed directly and simply
type ContextReader interface {
	ReadAt(ctx context.Context, p []byte, off int64) (int, error)
	ReadAll(ctx context.Context) ([]byte, error)

	// Return an io.Reader representing the underlying. May not be supported by all implementations
	Reader() (io.Reader, error)
}

// backendReader is a shim that allows a backend.Reader to be used as a ContextReader
type backendReader struct {
	meta *BlockMeta
	name string
	r    Reader
}

// NewContextReader creates a ReaderAt for the given BlockMeta
func NewContextReader(meta *BlockMeta, name string, r Reader) ContextReader {
	return &backendReader{
		meta: meta,
		name: name,
		r:    r,
	}
}

// ReadAt implements ContextReader
func (b *backendReader) ReadAt(ctx context.Context, p []byte, off int64) (int, error) {
	err := b.r.ReadRange(ctx, b.name, b.meta.BlockID, b.meta.TenantID, uint64(off), p, nil)
	return len(p), err
}

// ReadAll implements ContextReader
func (b *backendReader) ReadAll(ctx context.Context) ([]byte, error) {
	return b.r.Read(ctx, b.name, b.meta.BlockID, b.meta.TenantID, nil)
}

// Reader implements ContextReader
func (b *backendReader) Reader() (io.Reader, error) {
	return nil, util.ErrUnsupported
}

// AllReader is an interface that supports both io.Reader and io.ReaderAt methods
type AllReader interface {
	io.Reader
	io.ReaderAt
}

// allReader wraps an AllReader and implements backend.ContextReader
type allReader struct {
	r AllReader
}

// NewContextReaderWithAllReader wraps a normal ReaderAt and drops the context
func NewContextReaderWithAllReader(r AllReader) ContextReader {
	return &allReader{
		r: r,
	}
}

// ReadAt implements ContextReader
func (r *allReader) ReadAt(_ context.Context, p []byte, off int64) (int, error) {
	return r.r.ReadAt(p, off)
}

// ReadAll implements ContextReader
func (r *allReader) ReadAll(_ context.Context) ([]byte, error) {
	return io.ReadAll(r.r)
}

// Reader implements ContextReader
func (r *allReader) Reader() (io.Reader, error) {
	return r.r, nil
}
