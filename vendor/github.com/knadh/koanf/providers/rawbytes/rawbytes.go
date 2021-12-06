// Package rawbytes implements a koanf.Provider that takes a []byte slice
// and provides it to koanf to be parsed by a koanf.Parser.
package rawbytes

import (
	"errors"
)

// RawBytes implements a raw bytes provider.
type RawBytes struct {
	b []byte
}

// Provider returns a provider that takes a raw []byte slice to be parsed
// by a koanf.Parser parser. This should be a nested conf map, like the
// contents of a raw JSON config file.
func Provider(b []byte) *RawBytes {
	r := &RawBytes{b: make([]byte, len(b))}
	copy(r.b[:], b)
	return r
}

// ReadBytes is not supported by the env provider.
func (r *RawBytes) ReadBytes() ([]byte, error) {
	return r.b, nil
}

// Read returns the raw bytes for parsing.
func (r *RawBytes) Read() (map[string]interface{}, error) {
	return nil, errors.New("buf provider does not support this method")
}

// Watch is not supported.
func (r *RawBytes) Watch(cb func(event interface{}, err error)) error {
	return errors.New("rawbytes provider does not support this method")
}
