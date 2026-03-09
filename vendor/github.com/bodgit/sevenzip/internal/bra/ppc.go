package bra

import (
	"encoding/binary"
	"io"
)

const ppcAlignment = 4

type ppc struct {
	ip uint32
}

func (c *ppc) Size() int { return ppcAlignment }

func (c *ppc) Convert(b []byte, encoding bool) int {
	if len(b) < c.Size() {
		return 0
	}

	var i int

	for i = 0; i < len(b) & ^(ppcAlignment-1); i += ppcAlignment {
		v := binary.BigEndian.Uint32(b[i:])

		if b[i+0]&0xfc == 0x48 && b[i+3]&3 == 1 {
			if encoding {
				v += c.ip
			} else {
				v -= c.ip
			}

			v &= 0x03ffffff
			v |= 0x48000000
		}

		c.ip += uint32(ppcAlignment)

		binary.BigEndian.PutUint32(b[i:], v)
	}

	return i
}

// NewPPCReader returns a new PPC io.ReadCloser.
func NewPPCReader(_ []byte, _ uint64, readers []io.ReadCloser) (io.ReadCloser, error) {
	return newReader(readers, new(ppc))
}
