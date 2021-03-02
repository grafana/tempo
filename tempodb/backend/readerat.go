package backend

import (
	"context"
	"io"
	"io/ioutil"
)

// ReaderAtContext is an io.ReaderAt interface that passes context.  It is used to simplify access to backend objects
// and abstract away the name/meta and other details so that the data can be accessed directly and simply
type ReaderAtContext interface { // jpe ContextReader
	ReadAt(ctx context.Context, p []byte, off int64) (int, error)
	ReadAll(ctx context.Context) ([]byte, error)
}

// readerAt is a shim that allows a backend.Reader to be used as an io.ReaderAt
type readerAt struct {
	meta *BlockMeta
	name string
	r    Reader
}

// NewReaderAt creates a ReaderAt for the given BlockMeta
func NewReaderAt(meta *BlockMeta, name string, r Reader) ReaderAtContext {
	return &readerAt{
		meta: meta,
		name: name,
		r:    r,
	}
}

// ReadAt implements ReaderAtContext
func (b *readerAt) ReadAt(ctx context.Context, p []byte, off int64) (int, error) {
	err := b.r.ReadRange(ctx, b.name, b.meta.BlockID, b.meta.TenantID, uint64(off), p)
	return len(p), err
}

// ReadAll implements ReaderAtContext
func (b *readerAt) ReadAll(ctx context.Context) ([]byte, error) {
	return b.r.Read(ctx, b.name, b.meta.BlockID, b.meta.TenantID)
}

// AllReader is an interface that supports both io.Reader and io.ReaderAt methods
type AllReader interface {
	io.Reader
	io.ReaderAt
}

// readerAtContext wraps a file and implements backend.ReaderAtContext
type readerAtContext struct {
	r      AllReader
	length int
}

// NewReaderAtWithReaderAt wraps a normal ReaderAt and drops the context
func NewReaderAtWithReaderAt(r AllReader) ReaderAtContext { // jpe change name (all names in this file)
	return &readerAtContext{
		r: r,
	}
}

// ReadAt implements ReaderAtContext
func (r *readerAtContext) ReadAt(ctx context.Context, p []byte, off int64) (int, error) {
	return r.r.ReadAt(p, off)
}

// ReadAll implements ReaderAtContext
func (r *readerAtContext) ReadAll(ctx context.Context) ([]byte, error) {
	return ioutil.ReadAll(r.r)
}
