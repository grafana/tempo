// SPDX-FileCopyrightText: 2024 Shun Sakai
//
// SPDX-License-Identifier: Apache-2.0 OR MIT

package lzip

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"io"

	"github.com/ulikunitz/xz/lzma"
)

// Writer is an [io.WriteCloser] that can be written to retrieve a lzip-format
// compressed file from data.
type Writer struct {
	w           io.Writer
	compressor  *lzma.Writer
	buf         bytes.Buffer
	header      *header
	wroteHeader bool
	trailer
	closed bool
}

// WriterOptions configures [Writer].
type WriterOptions struct {
	// DictSize sets the dictionary size.
	DictSize uint32
}

func newWriterOptions() *WriterOptions {
	opt := &WriterOptions{DefaultDictSize}

	return opt
}

// Verify checks if [WriterOptions] is valid.
func (o *WriterOptions) Verify() error {
	switch dictSize := o.DictSize; {
	case dictSize < MinDictSize:
		return &DictSizeTooSmallError{dictSize}
	case dictSize > MaxDictSize:
		return &DictSizeTooLargeError{dictSize}
	}

	return nil
}

// NewWriter creates a new [Writer] writing the given writer.
//
// This uses the default parameters.
func NewWriter(w io.Writer) *Writer {
	opt := newWriterOptions()

	z, err := NewWriterOptions(w, opt)
	if err != nil {
		panic(err)
	}

	return z
}

// NewWriterOptions creates a new [Writer] writing the given writer.
//
// This uses the given [WriterOptions].
func NewWriterOptions(w io.Writer, opt *WriterOptions) (*Writer, error) {
	if err := opt.Verify(); err != nil {
		return nil, err
	}

	z := &Writer{w: w}

	compressor, err := lzma.WriterConfig{DictCap: int(opt.DictSize)}.NewWriter(&z.buf)
	if err != nil {
		return nil, err
	}

	z.compressor = compressor

	header := newHeader(opt.DictSize)
	z.header = header

	return z, nil
}

// Write compresses the given uncompressed data.
func (z *Writer) Write(p []byte) (int, error) {
	if !z.wroteHeader {
		z.wroteHeader = true

		var header [headerSize]byte

		copy(header[:magicSize], z.header.magic[:])
		header[4] = byte(z.header.version)
		header[5] = z.header.dictSize

		if _, err := z.w.Write(header[:]); err != nil {
			return 0, err
		}
	}

	n, err := z.compressor.Write(p)
	if err != nil {
		return n, err
	}

	z.trailer.crc = crc32.Update(z.trailer.crc, crc32.IEEETable, p)
	z.trailer.dataSize += uint64(len(p))

	return n, nil
}

// Close closes the [Writer] and writing the lzip trailer. It does not close
// the underlying [io.Writer].
func (z *Writer) Close() error {
	if z.closed {
		return nil
	}

	z.closed = true

	if err := z.compressor.Close(); err != nil {
		return err
	}

	cb := z.buf.Bytes()[lzma.HeaderLen:]
	if _, err := z.w.Write(cb); err != nil {
		return err
	}

	var trailer [trailerSize]byte

	binary.LittleEndian.PutUint32(trailer[:4], z.trailer.crc)
	binary.LittleEndian.PutUint64(trailer[4:12], z.trailer.dataSize)
	binary.LittleEndian.PutUint64(trailer[12:], headerSize+uint64(len(cb))+trailerSize)

	if memberSize := binary.LittleEndian.Uint64(trailer[12:]); memberSize > MaxMemberSize {
		return &MemberSizeTooLargeError{memberSize}
	}

	_, err := z.w.Write(trailer[:])

	return err
}
