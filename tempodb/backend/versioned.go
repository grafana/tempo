package backend

import (
	"context"
	"io"

	"github.com/pkg/errors"
)

type UpdateFn func(current io.ReadCloser) ([]byte, error)

type Version string

const (
	// VersionNew is a placeholder version for a new file
	VersionNew Version = "0"
)

var (
	ErrVersionDoesNotMatch = errors.New("version does not match")
)

// VersionedReaderWriter is a collection of methods to read and write data from tempodb backends with
// versioning enabled.
type VersionedReaderWriter interface {
	RawReader
	RawWriter

	// WriteVersioned data to an object, if the version does not match the request will fail with
	// ErrVersionDoesNotMatch. If the operation will create a new file, specify VersionNew.
	WriteVersioned(ctx context.Context, name string, keypath KeyPath, data io.Reader, version Version) (Version, error)

	// ReadVersioned data from an object and returns the current version.
	ReadVersioned(ctx context.Context, name string, keypath KeyPath) (io.ReadCloser, Version, error)

	// TODO
	// DeleteVersioned
}

type fakeVersionedReaderWriter struct {
	RawReader
	RawWriter
}

var _ VersionedReaderWriter = (*fakeVersionedReaderWriter)(nil)

func NewFakeVersionedReaderWriter(r RawReader, w RawWriter) VersionedReaderWriter {
	return fakeVersionedReaderWriter{r, w}
}

func (f fakeVersionedReaderWriter) WriteVersioned(ctx context.Context, name string, keypath KeyPath, data io.Reader, version Version) (Version, error) {
	err := f.Write(ctx, name, keypath, data, -1, false)
	return VersionNew, err
}

func (f fakeVersionedReaderWriter) ReadVersioned(ctx context.Context, name string, keypath KeyPath) (io.ReadCloser, Version, error) {
	readCloser, _, err := f.Read(ctx, name, keypath, false)
	return readCloser, VersionNew, err
}
