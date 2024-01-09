package backend

import (
	"context"
	"errors"
	"io"
)

type UpdateFn func(current io.ReadCloser) ([]byte, error)

type Version string

const (
	// VersionNew is a placeholder version for a new file
	VersionNew Version = "0"
)

var (
	ErrVersionDoesNotMatch = errors.New("version does not match")
	ErrVersionInvalid      = errors.New("version is not valid")
)

// VersionedReaderWriter is a collection of methods to read and write data from tempodb backends with
// versioning enabled.
type VersionedReaderWriter interface {
	RawReader

	// WriteVersioned data to an object, if the version does not match the request will fail with
	// ErrVersionDoesNotMatch. If the operation will create a new file, pass VersionNew.
	WriteVersioned(ctx context.Context, name string, keypath KeyPath, data io.Reader, version Version) (Version, error)

	// DeleteVersioned an object, if the version does not match the request will fail with
	// ErrVersionDoesNotMatch.
	DeleteVersioned(ctx context.Context, name string, keypath KeyPath, version Version) error

	// ReadVersioned data from an object and returns the current version.
	ReadVersioned(ctx context.Context, name string, keypath KeyPath) (io.ReadCloser, Version, error)
}

type FakeVersionedReaderWriter struct {
	RawReader
	RawWriter
}

var _ VersionedReaderWriter = (*FakeVersionedReaderWriter)(nil)

func NewFakeVersionedReaderWriter(r RawReader, w RawWriter) *FakeVersionedReaderWriter {
	return &FakeVersionedReaderWriter{r, w}
}

func (f *FakeVersionedReaderWriter) WriteVersioned(ctx context.Context, name string, keypath KeyPath, data io.Reader, _ Version) (Version, error) {
	err := f.Write(ctx, name, keypath, data, -1, nil)
	return VersionNew, err
}

func (f *FakeVersionedReaderWriter) ReadVersioned(ctx context.Context, name string, keypath KeyPath) (io.ReadCloser, Version, error) {
	readCloser, _, err := f.Read(ctx, name, keypath, nil)
	return readCloser, VersionNew, err
}

func (f *FakeVersionedReaderWriter) DeleteVersioned(ctx context.Context, name string, keypath KeyPath, _ Version) error {
	return f.Delete(ctx, name, keypath, nil)
}
