// Package bcj2 implements the BCJ2 filter for x86 binaries.
package bcj2

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/bodgit/sevenzip/internal/util"
	"github.com/hashicorp/go-multierror"
)

type readCloser struct {
	main util.ReadCloser
	call io.ReadCloser
	jump io.ReadCloser

	rd     util.ReadCloser
	nrange uint
	code   uint

	sd [256 + 2]uint

	previous byte
	written  uint32

	buf *bytes.Buffer
}

const (
	numMoveBits               = 5
	numbitModelTotalBits      = 11
	bitModelTotal        uint = 1 << numbitModelTotalBits
	numTopBits                = 24
	topValue             uint = 1 << numTopBits
)

var (
	errAlreadyClosed   = errors.New("bcj2: already closed")
	errNeedFourReaders = errors.New("bcj2: need exactly four readers")
)

func isJcc(b0, b1 byte) bool {
	return b0 == 0x0f && (b1&0xf0) == 0x80
}

func isJ(b0, b1 byte) bool {
	return (b1&0xfe) == 0xe8 || isJcc(b0, b1)
}

func index(b0, b1 byte) int {
	switch b1 {
	case 0xe8:
		return int(b0)
	case 0xe9:
		return 256
	default:
		return 257
	}
}

// NewReader returns a new BCJ2 io.ReadCloser.
func NewReader(_ []byte, _ uint64, readers []io.ReadCloser) (io.ReadCloser, error) {
	if len(readers) != 4 {
		return nil, errNeedFourReaders
	}

	rc := &readCloser{
		main:   util.ByteReadCloser(readers[0]),
		call:   readers[1],
		jump:   readers[2],
		rd:     util.ByteReadCloser(readers[3]),
		nrange: 0xffffffff,
		buf:    new(bytes.Buffer),
	}
	rc.buf.Grow(1 << 16)

	b := make([]byte, 5)
	if _, err := io.ReadFull(rc.rd, b); err != nil {
		if !errors.Is(err, io.EOF) {
			err = fmt.Errorf("bcj2: error reading initial state: %w", err)
		}

		return nil, err
	}

	for _, x := range b {
		rc.code = (rc.code << 8) | uint(x)
	}

	for i := range rc.sd {
		rc.sd[i] = bitModelTotal >> 1
	}

	return rc, nil
}

func (rc *readCloser) Close() error {
	if rc.main == nil || rc.call == nil || rc.jump == nil || rc.rd == nil {
		return errAlreadyClosed
	}

	//nolint:lll
	if err := multierror.Append(rc.main.Close(), rc.call.Close(), rc.jump.Close(), rc.rd.Close()).ErrorOrNil(); err != nil {
		return fmt.Errorf("bcj2: error closing: %w", err)
	}

	rc.main, rc.call, rc.jump, rc.rd = nil, nil, nil, nil

	return nil
}

func (rc *readCloser) Read(p []byte) (int, error) {
	if rc.main == nil || rc.call == nil || rc.jump == nil || rc.rd == nil {
		return 0, errAlreadyClosed
	}

	if err := rc.read(); err != nil && !errors.Is(err, io.EOF) {
		return 0, err
	}

	n, err := rc.buf.Read(p)
	if err != nil && !errors.Is(err, io.EOF) {
		err = fmt.Errorf("bcj2: error reading: %w", err)
	}

	return n, err
}

func (rc *readCloser) update() error {
	if rc.nrange < topValue {
		b, err := rc.rd.ReadByte()
		if err != nil && !errors.Is(err, io.EOF) {
			return fmt.Errorf("bcj2: error reading byte: %w", err)
		}

		rc.code = (rc.code << 8) | uint(b)
		rc.nrange <<= 8
	}

	return nil
}

func (rc *readCloser) decode(i int) (bool, error) {
	newBound := (rc.nrange >> numbitModelTotalBits) * rc.sd[i]

	if rc.code < newBound {
		rc.nrange = newBound
		rc.sd[i] += (bitModelTotal - rc.sd[i]) >> numMoveBits

		if err := rc.update(); err != nil {
			return false, err
		}

		return false, nil
	}

	rc.nrange -= newBound
	rc.code -= newBound
	rc.sd[i] -= rc.sd[i] >> numMoveBits

	if err := rc.update(); err != nil {
		return false, err
	}

	return true, nil
}

//nolint:cyclop,funlen
func (rc *readCloser) read() error {
	var (
		b   byte
		err error
	)

	for {
		if b, err = rc.main.ReadByte(); err != nil {
			if !errors.Is(err, io.EOF) {
				err = fmt.Errorf("bcj2: error reading byte: %w", err)
			}

			return err
		}

		rc.written++
		_ = rc.buf.WriteByte(b)

		if isJ(rc.previous, b) {
			break
		}

		rc.previous = b

		if rc.buf.Len() == rc.buf.Cap() {
			return nil
		}
	}

	bit, err := rc.decode(index(rc.previous, b))
	if err != nil {
		return err
	}

	//nolint:nestif
	if bit {
		var r io.Reader
		if b == 0xe8 {
			r = rc.call
		} else {
			r = rc.jump
		}

		var dest uint32
		if err = binary.Read(r, binary.BigEndian, &dest); err != nil {
			if !errors.Is(err, io.EOF) {
				err = fmt.Errorf("bcj2: error reading uint32: %w", err)
			}

			return err
		}

		dest -= rc.written + 4
		_ = binary.Write(rc.buf, binary.LittleEndian, dest)

		rc.previous = byte(dest >> 24)
		rc.written += 4
	} else {
		rc.previous = b
	}

	return nil
}
