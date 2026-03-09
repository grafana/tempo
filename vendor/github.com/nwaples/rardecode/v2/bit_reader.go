package rardecode

import (
	"io"
	"math/bits"
)

type bitReader interface {
	readBits(n uint8) (int, error) // read n bits of data
	unreadBits(n uint8)            // revert the reading of the last n bits read
}

// rar5BitReader is a bitReader that reads bytes from a byteReader and stops with io.EOF after l bits.
type rar5BitReader struct {
	r byteReader
	v int    // cache of bits read from r
	l int    // number of bits (not cached) that can be read from r
	n uint8  // number of unread bits in v
	b []byte // bytes() output cache from r
}

func (r *rar5BitReader) unreadBits(n uint8) { r.n += n }

// ReadByte reads a byte from rar5BitReader's byteReader ignoring the bit cache v.
func (r *rar5BitReader) ReadByte() (byte, error) {
	if len(r.b) == 0 {
		var err error
		r.b, err = r.r.bytes()
		if err != nil {
			if err == io.EOF {
				err = ErrDecoderOutOfData
			}
			return 0, err
		}
	}
	c := r.b[0]
	r.b = r.b[1:]
	return c, nil
}

func (r *rar5BitReader) reset(br byteReader) {
	r.r = br
	r.b = nil
}

// setLimit sets the maximum bit count that can be read.
func (r *rar5BitReader) setLimit(n int) {
	r.l = n
	r.n = 0
}

// readBits returns n bits from the underlying byteReader.
// n must be less than integer size - 8.
func (r *rar5BitReader) readBits(n uint8) (int, error) {
	for n > r.n {
		if r.l == 0 {
			// reached bits limit
			return 0, io.EOF
		}
		if len(r.b) == 0 {
			var err error
			r.b, err = r.r.bytes()
			if err != nil {
				if err == io.EOF {
					// io.EOF before we reached bit limit
					err = ErrDecoderOutOfData
				}
				return 0, err
			}
		}
		// try to fit as many bits into r.v as possible
		for len(r.b) > 0 && r.n <= bits.UintSize-8 {
			r.v = r.v<<8 | int(r.b[0])
			r.b = r.b[1:]
			r.n += 8
			r.l -= 8
			if r.l <= 0 {
				if r.l < 0 {
					// overshot, discard the extra bits
					bits := uint8(-r.l)
					r.l = 0
					r.v >>= bits
					r.n -= bits
				}
				break
			}
		}
	}
	r.n -= n
	return (r.v >> r.n) & ((1 << n) - 1), nil
}

// replaceByteReader is a byteReader that returns b on the first call to bytes()
// and then replaces the byteReader at rp with r.
type replaceByteReader struct {
	rp *byteReader
	r  byteReader
	b  []byte
}

func (r *replaceByteReader) Read(p []byte) (int, error) { return 0, io.EOF }

func (r *replaceByteReader) bytes() ([]byte, error) {
	*r.rp = r.r
	return r.b, nil
}

// rarBitReader wraps an io.ByteReader to perform various bit and byte
// reading utility functions used in RAR file processing.
type rarBitReader struct {
	r byteReader
	v int
	n uint8
	b []byte
}

func (r *rarBitReader) reset(br byteReader) {
	r.r = br
	r.n = 0
	r.v = 0
	r.b = nil
}

// unshiftBytes moves any bytes in rarBitReader bit cache back into a byte slice
// and sets up byteReader's so that all bytes can now be read by ReadByte() without
// going through the bit cache.
func (r *rarBitReader) unshiftBytes() {
	// no cached bits
	if r.n == 0 {
		return
	}
	// create and read byte slice for cached bits
	b := make([]byte, r.n/8)
	for i := len(b) - 1; i >= 0; i-- {
		b[i] = byte(r.v)
		r.v >>= 8
	}
	r.n = 0
	// current bytes buffer empty, so store b and return
	if len(r.b) == 0 {
		r.b = b
		return
	}
	// Put current bytes buffer and byteReader in a replaceByteReader and
	// the unshifted bytes in the rarBitReader bytes buffer.
	// When the bytes buffer is consumed, rarBitReader will call bytes()
	// on replaceByteReader which will return the old bytes buffer and
	// replace itself with the old byteReader in rarBitReader.
	r.r = &replaceByteReader{rp: &r.r, r: r.r, b: r.b}
	r.b = b
}

// readBits returns n bits from the underlying byteReader.
// n must be less than integer size - 8.
func (r *rarBitReader) readBits(n uint8) (int, error) {
	for n > r.n {
		if len(r.b) == 0 {
			var err error
			r.b, err = r.r.bytes()
			if err != nil {
				return 0, err
			}
		}
		// try to fit as many bits into r.v as possible
		for len(r.b) > 0 && r.n <= bits.UintSize-8 {
			r.v = r.v<<8 | int(r.b[0])
			r.b = r.b[1:]
			r.n += 8
		}
	}
	r.n -= n
	return (r.v >> r.n) & ((1 << n) - 1), nil
}

func (r *rarBitReader) unreadBits(n uint8) {
	r.n += n
}

// alignByte aligns the current bit reading input to the next byte boundary.
func (r *rarBitReader) alignByte() {
	r.n -= r.n % 8
}

// readUint32 reads a RAR V3 encoded uint32
func (r *rarBitReader) readUint32() (uint32, error) {
	n, err := r.readBits(2)
	if err != nil {
		return 0, err
	}
	if n != 1 {
		if bits.UintSize == 32 {
			if n == 3 {
				// 32bit platforms may not be able to read 32 bits as r.v
				// will need up to 7 extra bits for overflow from reading a byte.
				// Split it into two reads.
				n, err = r.readBits(16)
				if err != nil {
					return 0, err
				}
				m := uint32(n) << 16
				n, err = r.readBits(16)
				return m | uint32(n), err
			}
		}
		n, err = r.readBits(4 << uint(n))
		return uint32(n), err
	}
	n, err = r.readBits(4)
	if err != nil {
		return 0, err
	}
	if n == 0 {
		n, err = r.readBits(8)
		n |= -1 << 8
		return uint32(n), err
	}
	nlow, err := r.readBits(4)
	n = n<<4 | nlow
	return uint32(n), err
}

// ReadByte() returns a byte directly from buf b or the io.ByteReader r.
// Current bit offsets are ignored.
func (r *rarBitReader) ReadByte() (byte, error) {
	if len(r.b) == 0 {
		if r.r == nil {
			return 0, io.EOF
		}
		var err error
		r.b, err = r.r.bytes()
		if err != nil {
			return 0, err
		}
	}
	c := r.b[0]
	r.b = r.b[1:]
	return c, nil
}

// readFull reads len(p) bytes into p. If fewer bytes are read an error is returned.
func (r *rarBitReader) readFull(p []byte) error {
	if r.n == 0 && len(r.b) > 0 {
		n := copy(p, r.b)
		p = p[n:]
		r.b = r.b[n:]
	}
	for i := range p {
		n, err := r.readBits(8)
		if err != nil {
			return err
		}
		p[i] = byte(n)
	}
	return nil
}

func newRarBitReader(r byteReader) *rarBitReader {
	return &rarBitReader{r: r}
}
