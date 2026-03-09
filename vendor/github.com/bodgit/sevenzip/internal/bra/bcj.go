package bra

import (
	"encoding/binary"
	"io"
)

const bcjLookAhead = 4

type bcj struct {
	ip, state uint32
}

func (c *bcj) Size() int { return bcjLookAhead + 1 }

func test86MSByte(b byte) bool {
	return (b+1)&0xfe == 0
}

//nolint:cyclop,funlen,gocognit
func (c *bcj) Convert(b []byte, encoding bool) int {
	if len(b) < c.Size() {
		return 0
	}

	var (
		pos  uint32
		mask = c.state & 7
	)

	for {
		p := pos
		for ; int(p) < len(b)-bcjLookAhead; p++ {
			if b[p]&0xfe == 0xe8 {
				break
			}
		}

		d := p - pos
		pos = p

		if int(p) >= len(b)-bcjLookAhead {
			if d > 2 {
				c.state = 0
			} else {
				c.state = mask >> d
			}

			c.ip += pos

			return int(pos)
		}

		if d > 2 {
			mask = 0
		} else {
			mask >>= d
			if mask != 0 && (mask > 4 || mask == 3 || test86MSByte(b[p+(mask>>1)+1])) {
				mask = (mask >> 1) | 4
				pos++

				continue
			}
		}

		//nolint:nestif
		if test86MSByte(b[p+4]) {
			v := binary.LittleEndian.Uint32(b[p+1:])
			cur := c.ip + uint32(c.Size()) + pos //nolint:gosec
			pos += uint32(c.Size())              //nolint:gosec

			if encoding {
				v += cur
			} else {
				v -= cur
			}

			if mask != 0 {
				sh := mask & 6 << 2
				if test86MSByte(byte(v >> sh)) {
					v ^= (uint32(0x100) << sh) - 1
					if encoding {
						v += cur
					} else {
						v -= cur
					}
				}

				mask = 0
			}

			binary.LittleEndian.PutUint32(b[p+1:], v)
			b[p+4] = 0 - b[p+4]&1
		} else {
			mask = (mask >> 1) | 4
			pos++
		}
	}
}

// NewBCJReader returns a new BCJ io.ReadCloser.
func NewBCJReader(_ []byte, _ uint64, readers []io.ReadCloser) (io.ReadCloser, error) {
	return newReader(readers, new(bcj))
}
