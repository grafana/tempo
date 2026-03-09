package plumbing

import (
	"io"
)

type multiWriteCloser struct {
	writeClosers []io.WriteCloser
}

func (t *multiWriteCloser) Write(p []byte) (n int, err error) {
	for _, wc := range t.writeClosers {
		n, err = wc.Write(p)
		if err != nil {
			return
		}

		if n != len(p) {
			err = io.ErrShortWrite

			return
		}
	}

	return len(p), nil
}

func (t *multiWriteCloser) Close() (err error) {
	for _, wc := range t.writeClosers {
		err = wc.Close()
		if err != nil {
			return
		}
	}

	return
}

// MultiWriteCloser creates a writer that duplicates its writes to all the
// provided writers, similar to the Unix tee(1) command.
//
// Each write is written to each listed writer, one at a time.
// If a listed writer returns an error, that overall write operation
// stops and returns the error; it does not continue down the list.
func MultiWriteCloser(writeClosers ...io.WriteCloser) io.WriteCloser {
	allWriteClosers := make([]io.WriteCloser, 0, len(writeClosers))

	for _, wc := range writeClosers {
		if mwc, ok := wc.(*multiWriteCloser); ok {
			allWriteClosers = append(allWriteClosers, mwc.writeClosers...)
		} else {
			allWriteClosers = append(allWriteClosers, wc)
		}
	}

	return &multiWriteCloser{allWriteClosers}
}

type multiReadCloser struct {
	readClosers []io.ReadCloser
	i           int
}

func (mrc *multiReadCloser) Read(p []byte) (n int, err error) {
	for mrc.i < len(mrc.readClosers) {
		if len(mrc.readClosers) == 1 {
			if rc, ok := mrc.readClosers[0].(*multiReadCloser); ok {
				mrc.readClosers = rc.readClosers

				continue
			}
		}

		n, err = mrc.readClosers[mrc.i].Read(p)
		if err == io.EOF { //nolint:errorlint
			mrc.i++
		}

		if n > 0 || err != io.EOF { //nolint:errorlint
			if err == io.EOF && mrc.i < len(mrc.readClosers) { //nolint:errorlint
				err = nil
			}

			return
		}
	}

	return 0, io.EOF
}

func (mrc *multiReadCloser) Close() (err error) {
	for _, rc := range mrc.readClosers {
		err = rc.Close()
		if err != nil {
			return
		}
	}

	return
}

// MultiReadCloser returns an io.ReadCloser that's the logical concatenation
// of the provider input readers. They're read sequentially. Once all inputs
// have returned io.EOF, Read will return EOF. If any of the readers return
// a non-nil, non-EOF error, Read will return that error.
func MultiReadCloser(readClosers ...io.ReadCloser) io.ReadCloser {
	rc := make([]io.ReadCloser, len(readClosers))
	copy(rc, readClosers)

	return &multiReadCloser{rc, 0}
}
