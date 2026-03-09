// Package plumbing is a collection of assorted I/O helpers.
package plumbing

import "io"

type nopWriteCloser struct {
	io.Writer
}

func (nopWriteCloser) Close() error {
	return nil
}

// NopWriteCloser returns an io.WriteCloser with a no-op Close method
// wrapping the provided io.Writer w.
func NopWriteCloser(w io.Writer) io.WriteCloser {
	return nopWriteCloser{w}
}
