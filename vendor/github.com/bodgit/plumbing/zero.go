package plumbing

import "io"

type devZero struct {
	io.Reader
}

func (w *devZero) Write(p []byte) (int, error) {
	return len(p), nil
}

// DevZero returns an io.ReadWriter that behaves like /dev/zero such that Read
// calls return an unlimited stream of zero bytes and all Write calls succeed
// without doing anything.
func DevZero() io.ReadWriter {
	return &devZero{FillReader(0)}
}
