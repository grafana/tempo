package bra

import (
	"encoding/binary"
	"io"
)

const sparcAlignment = 4

type sparc struct {
	ip uint32
}

func (c *sparc) Size() int { return sparcAlignment }

func (c *sparc) Convert(b []byte, encoding bool) int {
	if len(b) < c.Size() {
		return 0
	}

	var i int

	for i = 0; i < len(b) & ^(sparcAlignment-1); i += sparcAlignment {
		v := binary.BigEndian.Uint32(b[i:])

		if (b[i+0] == 0x40 && b[i+1]&0xc0 == 0) || (b[i+0] == 0x7f && b[i+1] >= 0xc0) {
			v <<= 2

			if encoding {
				v += c.ip
			} else {
				v -= c.ip
			}

			v &= 0x01ffffff
			v -= uint32(1) << 24
			v ^= 0xff000000
			v >>= 2
			v |= 0x40000000
		}

		c.ip += uint32(sparcAlignment)

		binary.BigEndian.PutUint32(b[i:], v)
	}

	return i
}

// NewSPARCReader returns a new SPARC io.ReadCloser.
func NewSPARCReader(_ []byte, _ uint64, readers []io.ReadCloser) (io.ReadCloser, error) {
	return newReader(readers, new(sparc))
}
