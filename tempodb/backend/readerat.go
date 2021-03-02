package backend

import (
	"context"
	"io"
)

// ReaderAtContext is an io.ReaderAt interface that passes context.  It is used to simplify access to backend objects
// and abstract away the name/meta and other details so that the data can be accessed directly and simply
type ReaderAtContext interface {
	ReadAt(ctx context.Context, p []byte, off int64) (int, error)
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
	err := b.r.ReadRange(context.Background(), b.name, b.meta.BlockID, b.meta.TenantID, uint64(off), p)
	return len(p), err
}

// readerAtContext wraps a file and implements backend.ReaderAtContext
type readerAtContext struct {
	ra io.ReaderAt
}

// NewReaderAtWithReaderAt wraps a normal ReaderAt and drops the context
func NewReaderAtWithReaderAt(ra io.ReaderAt) ReaderAtContext {
	return &readerAtContext{
		ra: ra,
	}
}

func (r *readerAtContext) ReadAt(ctx context.Context, p []byte, off int64) (int, error) {
	return r.ra.ReadAt(p, off)
}
