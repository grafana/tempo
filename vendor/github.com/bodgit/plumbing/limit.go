package plumbing

import "io"

// A LimitedReadCloser reads from R but limits the amount of
// data returned to just N bytes. Each call to Read
// updates N to reflect the new amount remaining.
// Read returns EOF when N <= 0 or when the underlying R returns EOF.
type LimitedReadCloser struct {
	R io.ReadCloser
	N int64
}

func (l *LimitedReadCloser) Read(p []byte) (n int, err error) {
	if l.N <= 0 {
		return 0, io.EOF
	}

	if int64(len(p)) > l.N {
		p = p[0:l.N]
	}

	n, err = l.R.Read(p)
	l.N -= int64(n)

	return
}

// Close closes the LimitedReadCloser, rendering it unusable for I/O.
func (l *LimitedReadCloser) Close() error {
	return l.R.Close()
}

// LimitReadCloser returns an io.ReadCloser that reads from r
// but stops with EOF after n bytes.
// The underlying implementation is a *LimitedReadCloser.
func LimitReadCloser(r io.ReadCloser, n int64) io.ReadCloser {
	return &LimitedReadCloser{r, n}
}
