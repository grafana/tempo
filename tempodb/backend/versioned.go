package backend

import (
	"bytes"
	"context"
	"io"
)

type UpdateFn func(current io.ReadCloser) ([]byte, error)

type VersionedReaderWriter interface {
	RawReader
	RawWriter

	// Update applies updateFn to the given object. If supported by the backend, the operation will
	// fail if another process modified the file.
	// If the file does not exist yet, updateFn is called with nil
	Update(ctx context.Context, name string, keypath KeyPath, updateFn UpdateFn) error
}

type uncheckedVersionedReaderWriter struct {
	RawReader
	RawWriter
}

func NewUncheckedVersionedReaderWriter(r RawReader, w RawWriter) VersionedReaderWriter {
	return &uncheckedVersionedReaderWriter{r, w}
}

func (p *uncheckedVersionedReaderWriter) Update(ctx context.Context, name string, keypath KeyPath, updateFn UpdateFn) error {
	rBytes, _, err := p.Read(ctx, name, keypath, false)
	if err != nil && err != ErrDoesNotExist {
		return err
	}

	wBytes, err := updateFn(rBytes)
	if err != nil {
		return err
	}

	err = p.Write(ctx, name, keypath, bytes.NewReader(wBytes), int64(len(wBytes)), false)
	if err != nil {
		return err
	}
	return nil
}
