package backend

import "context"

// BackendReaderAt is a shim that allows a backend.Reader to be used as an io.ReaderAt
type BackendReaderAt struct {
	meta *BlockMeta
	name string
	r    Reader
}

// NewBackendReaderAt creates a BackendReaderAt for the given BlockMeta
func NewBackendReaderAt(meta *BlockMeta, name string, r Reader) *BackendReaderAt {
	return &BackendReaderAt{
		meta: meta,
		name: name,
		r:    r,
	}
}

// ReadAt implements ReaderAt
func (b *BackendReaderAt) ReadAt(p []byte, off int64) (int, error) {
	// todo:  how to preserve context?  len(p) is cheating
	err := b.r.ReadRange(context.Background(), b.name, b.meta.BlockID, b.meta.TenantID, uint64(off), p)
	return len(p), err
}
