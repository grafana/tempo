package backend

import "context"

// ReaderAt is a shim that allows a backend.Reader to be used as an io.ReaderAt
type ReaderAt struct {
	meta *BlockMeta
	name string
	r    Reader
}

// NewReaderAt creates a ReaderAt for the given BlockMeta
func NewReaderAt(meta *BlockMeta, name string, r Reader) *ReaderAt {
	return &ReaderAt{
		meta: meta,
		name: name,
		r:    r,
	}
}

// ReadAt implements ReaderAt
func (b *ReaderAt) ReadAt(p []byte, off int64) (int, error) {
	// todo:  how to preserve context?  len(p) is cheating
	err := b.r.ReadRange(context.Background(), b.name, b.meta.BlockID, b.meta.TenantID, uint64(off), p)
	return len(p), err
}
