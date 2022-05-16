package bits

import (
	"encoding/binary"
	"io"
)

type Reader struct {
	reader io.Reader
	length uint
	cache  uint64
	buffer [8]byte
}

func (r *Reader) Reset(rr io.Reader) {
	r.reader = rr
	r.length = 0
	r.cache = 0
}

func (r *Reader) ReadBit() (int, error) {
	bits, _, err := r.ReadBits(1)
	return int(bits), err
}

func (r *Reader) ReadBits(count uint) (uint64, uint, error) {
	bits, nbits := uint64(0), uint(0)

	for count > 0 {
		if r.length == 0 {
			byteCount := ByteCount(count)
			if byteCount > 8 {
				byteCount = 8
			}
			n, err := r.reader.Read(r.buffer[:byteCount])
			if err != nil && n == 0 {
				if err == io.EOF && nbits != 0 {
					err = io.ErrUnexpectedEOF
				}
				return bits, nbits, err
			}
			b := [8]byte{}
			copy(b[:], r.buffer[:n])
			r.length = 8 * uint(n)
			r.cache = binary.LittleEndian.Uint64(b[:])
		}

		n := count
		if n > r.length {
			n = r.length
		}

		bits |= (r.cache & ((1 << n) - 1)) << nbits
		nbits += n
		count -= n
		r.length -= n
		r.cache >>= n
	}

	return bits, nbits, nil
}

type Writer struct {
	writer io.Writer
	length uint
	cache  uint64
	buffer []byte
}

func (w *Writer) Buffered() int {
	return len(w.buffer)
}

func (w *Writer) Reset(ww io.Writer) {
	w.writer = ww
	w.length = 0
	w.buffer = w.buffer[:0]
}

func (w *Writer) Flush() error {
	w.flush()
	_, err := w.writer.Write(w.buffer)
	w.buffer = w.buffer[:0]
	return err
}

func (w *Writer) flush() {
	b := [8]byte{}
	binary.LittleEndian.PutUint64(b[:], w.cache)
	w.buffer = append(w.buffer, b[:ByteCount(w.length)]...)
	w.length = 0
	w.cache = 0
}

func (w *Writer) WriteBit(bit int) {
	w.WriteBits(uint64(bit), 1)
}

func (w *Writer) WriteBits(bits uint64, count uint) {
	for {
		w.cache |= (bits & ((1 << count) - 1)) << w.length
		n := 64 - w.length
		if n >= count {
			w.length += count
			break
		}
		w.length += n
		bits >>= n
		count -= n
		w.flush()
	}
}
