package plumbing

import "io"

type teeReaderAt struct {
	r io.ReaderAt
	w io.Writer
}

func (t *teeReaderAt) ReadAt(p []byte, off int64) (n int, err error) {
	n, err = t.r.ReadAt(p, off)
	if n > 0 {
		if n, err := t.w.Write(p[:n]); err != nil {
			return n, err
		}
	}

	return
}

// TeeReaderAt returns an io.ReaderAt that writes to w what it reads from r.
// All reads from r performed through it are matched with corresponding writes
// to w.  There is no internal buffering - the write must complete before the
// read completes. Any error encountered while writing is reported as a read
// error.
func TeeReaderAt(r io.ReaderAt, w io.Writer) io.ReaderAt {
	return &teeReaderAt{r, w}
}

type teeReadCloser struct {
	r io.ReadCloser
	w io.Writer
}

func (t *teeReadCloser) Read(p []byte) (n int, err error) {
	n, err = t.r.Read(p)
	if n > 0 {
		if n, err := t.w.Write(p[:n]); err != nil {
			return n, err
		}
	}

	return
}

func (t *teeReadCloser) Close() error {
	return t.r.Close()
}

// TeeReadCloser returns an io.ReadCloser that writes to w what it reads from
// r. All reads from r performed through it are matched with corresponding
// writes to w. There is no internal buffering - the write must complete
// before the read completes. Any error encountered while writing is reported
// as a read error.
func TeeReadCloser(r io.ReadCloser, w io.Writer) io.ReadCloser {
	return &teeReadCloser{r, w}
}
