package plumbing

import "io"

type fillReader struct {
	b byte
}

func (r *fillReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = r.b
	}

	return len(p), nil
}

// FillReader returns an io.Reader such that Read calls return an unlimited
// stream of b bytes.
func FillReader(b byte) io.Reader {
	return &fillReader{b}
}
