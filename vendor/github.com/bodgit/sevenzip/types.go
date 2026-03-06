package sevenzip

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/bits"
	"time"

	"github.com/bodgit/sevenzip/internal/util"
	"github.com/bodgit/windows"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

const (
	idEnd = iota
	idHeader
	idArchiveProperties
	idAdditionalStreamsInfo
	idMainStreamsInfo
	idFilesInfo
	idPackInfo
	idUnpackInfo
	idSubStreamsInfo
	idSize
	idCRC
	idFolder
	idCodersUnpackSize
	idNumUnpackStream
	idEmptyStream
	idEmptyFile
	idAnti //nolint:deadcode,varcheck
	idName
	idCTime
	idATime
	idMTime
	idWinAttributes
	idComment //nolint:deadcode,varcheck
	idEncodedHeader
	idStartPos
	idDummy
)

var (
	errIncompleteRead         = errors.New("sevenzip: incomplete read")
	errUnexpectedID           = errors.New("sevenzip: unexpected id")
	errMissingUnpackInfo      = errors.New("sevenzip: missing unpack info")
	errWrongNumberOfFilenames = errors.New("sevenzip: wrong number of filenames")
)

func readUint64(r io.ByteReader) (uint64, error) {
	b, err := r.ReadByte()
	if err != nil {
		return 0, fmt.Errorf("readUint64: ReadByte error: %w", err)
	}

	l := bits.LeadingZeros8(^b)

	var v uint64
	if l < 7 {
		v |= uint64(b&((1<<(8-l))-1)) << (8 * l)
	}

	for i := 0; i < l; i++ {
		b, err := r.ReadByte()
		if err != nil {
			return 0, fmt.Errorf("readUint64: ReadByte error: %w", err)
		}

		v |= uint64(b) << (8 * i)
	}

	return v, nil
}

func readBool(r io.ByteReader, count uint64) ([]bool, error) {
	defined := make([]bool, count)

	var b, mask byte
	for i := range defined {
		if mask == 0 {
			var err error

			b, err = r.ReadByte()
			if err != nil {
				return nil, fmt.Errorf("readBool: ReadByte error: %w", err)
			}

			mask = 0x80
		}

		defined[i] = (b & mask) != 0
		mask >>= 1
	}

	return defined, nil
}

func readOptionalBool(r io.ByteReader, count uint64) ([]bool, error) {
	all, err := r.ReadByte()
	if err != nil {
		return nil, fmt.Errorf("readOptionalBool: ReadByte error: %w", err)
	}

	if all == 0 {
		return readBool(r, count)
	}

	defined := make([]bool, count)
	for i := range defined {
		defined[i] = true
	}

	return defined, nil
}

func readSizes(r io.ByteReader, count uint64) ([]uint64, error) {
	sizes := make([]uint64, count)

	for i := uint64(0); i < count; i++ {
		size, err := readUint64(r)
		if err != nil {
			return nil, err
		}

		sizes[i] = size
	}

	return sizes, nil
}

func readCRC(r util.Reader, count uint64) ([]uint32, error) {
	defined, err := readOptionalBool(r, count)
	if err != nil {
		return nil, err
	}

	crcs := make([]uint32, count)

	for i := range defined {
		if defined[i] {
			if err := binary.Read(r, binary.LittleEndian, &crcs[i]); err != nil {
				return nil, fmt.Errorf("readCRC: Read error: %w", err)
			}
		}
	}

	return crcs, nil
}

//nolint:cyclop
func readPackInfo(r util.Reader) (*packInfo, error) {
	p := new(packInfo)

	var err error

	p.position, err = readUint64(r)
	if err != nil {
		return nil, err
	}

	p.streams, err = readUint64(r)
	if err != nil {
		return nil, err
	}

	id, err := r.ReadByte()
	if err != nil {
		return nil, fmt.Errorf("readPackInfo: ReadByte error: %w", err)
	}

	if id == idSize {
		if p.size, err = readSizes(r, p.streams); err != nil {
			return nil, err
		}

		id, err = r.ReadByte()
		if err != nil {
			return nil, fmt.Errorf("readPackInfo: ReadByte error: %w", err)
		}
	}

	if id == idCRC {
		if p.digest, err = readCRC(r, p.streams); err != nil {
			return nil, err
		}

		id, err = r.ReadByte()
		if err != nil {
			return nil, fmt.Errorf("readPackInfo: ReadByte error: %w", err)
		}
	}

	if id != idEnd {
		return nil, errUnexpectedID
	}

	return p, nil
}

//nolint:cyclop
func readCoder(r util.Reader) (*coder, error) {
	c := new(coder)

	v, err := r.ReadByte()
	if err != nil {
		return nil, fmt.Errorf("readCoder: ReadByte error: %w", err)
	}

	c.id = make([]byte, v&0xf)
	if n, err := r.Read(c.id); err != nil || n != int(v&0xf) {
		if err != nil {
			return nil, fmt.Errorf("readCoder: Read error: %w", err)
		}

		return nil, errIncompleteRead
	}

	if v&0x10 != 0 {
		c.in, err = readUint64(r)
		if err != nil {
			return nil, err
		}

		c.out, err = readUint64(r)
		if err != nil {
			return nil, err
		}
	} else {
		c.in, c.out = 1, 1
	}

	if v&0x20 != 0 {
		size, err := readUint64(r)
		if err != nil {
			return nil, err
		}

		c.properties = make([]byte, size)
		if n, err := r.Read(c.properties); err != nil || uint64(n) != size { //nolint:gosec
			if err != nil {
				return nil, fmt.Errorf("readCoder: Read error: %w", err)
			}

			return nil, errIncompleteRead
		}
	}

	return c, nil
}

//nolint:cyclop
func readFolder(r util.Reader) (*folder, error) {
	f := new(folder)

	coders, err := readUint64(r)
	if err != nil {
		return nil, err
	}

	f.coder = make([]*coder, coders)

	for i := uint64(0); i < coders; i++ {
		if f.coder[i], err = readCoder(r); err != nil {
			return nil, err
		}

		f.in += f.coder[i].in
		f.out += f.coder[i].out
	}

	bindPairs := f.out - 1

	f.bindPair = make([]*bindPair, bindPairs)

	for i := uint64(0); i < bindPairs; i++ {
		in, err := readUint64(r)
		if err != nil {
			return nil, err
		}

		out, err := readUint64(r)
		if err != nil {
			return nil, err
		}

		f.bindPair[i] = &bindPair{
			in:  in,
			out: out,
		}
	}

	f.packedStreams = f.in - bindPairs

	if f.packedStreams == 1 {
		f.packed = []uint64{}
		for i := uint64(0); i < f.in; i++ {
			if f.findInBindPair(i) == nil {
				f.packed = append(f.packed, i)
			}
		}
	} else {
		f.packed = make([]uint64, f.packedStreams)
		for i := uint64(0); i < f.packedStreams; i++ {
			if f.packed[i], err = readUint64(r); err != nil {
				return nil, err
			}
		}
	}

	return f, nil
}

//nolint:cyclop,funlen,gocognit
func readUnpackInfo(r util.Reader) (*unpackInfo, error) {
	u := new(unpackInfo)

	if id, err := r.ReadByte(); err != nil || id != idFolder {
		if err != nil {
			return nil, fmt.Errorf("readUnpackInfo: ReadByte error: %w", err)
		}

		return nil, errUnexpectedID
	}

	folders, err := readUint64(r)
	if err != nil {
		return nil, err
	}

	external, err := r.ReadByte()
	if err != nil {
		return nil, fmt.Errorf("readUnpackInfo: ReadByte error: %w", err)
	}

	if external > 0 {
		_, err := readUint64(r)
		if err != nil {
			return nil, err
		}
		// TODO Apparently we seek to this read offset and read the
		// folder information from there. Not clear if the offset is
		// absolute for the whole file, or relative to some known
		// position in the file. Cowardly waiting for an example
		return nil, errors.New("sevenzip: TODO readUnpackInfo external") //nolint:goerr113
	}

	u.folder = make([]*folder, folders)

	for i := uint64(0); i < folders; i++ {
		if u.folder[i], err = readFolder(r); err != nil {
			return nil, err
		}
	}

	if id, err := r.ReadByte(); err != nil || id != idCodersUnpackSize {
		if err != nil {
			return nil, fmt.Errorf("readUnpackInfo: ReadByte error: %w", err)
		}

		return nil, errUnexpectedID
	}

	for _, f := range u.folder {
		total := uint64(0)
		for _, c := range f.coder {
			total += c.out
		}

		f.size = make([]uint64, total)
		for i := range f.size {
			if f.size[i], err = readUint64(r); err != nil {
				return nil, err
			}
		}
	}

	id, err := r.ReadByte()
	if err != nil {
		return nil, fmt.Errorf("readUnpackInfo: ReadByte error: %w", err)
	}

	if id == idCRC {
		if u.digest, err = readCRC(r, folders); err != nil {
			return nil, err
		}

		id, err = r.ReadByte()
		if err != nil {
			return nil, fmt.Errorf("readUnpackInfo: ReadByte error: %w", err)
		}
	}

	if id != idEnd {
		return nil, errUnexpectedID
	}

	return u, nil
}

//nolint:cyclop,funlen
func readSubStreamsInfo(r util.Reader, folder []*folder) (*subStreamsInfo, error) {
	s := new(subStreamsInfo)

	id, err := r.ReadByte()
	if err != nil {
		return nil, fmt.Errorf("readSubStreamsInfo: ReadByte error: %w", err)
	}

	s.streams = make([]uint64, len(folder))
	if id == idNumUnpackStream {
		for i := range s.streams {
			if s.streams[i], err = readUint64(r); err != nil {
				return nil, err
			}
		}

		id, err = r.ReadByte()
		if err != nil {
			return nil, fmt.Errorf("readSubStreamsInfo: ReadByte error: %w", err)
		}
	} else {
		for i := range s.streams {
			s.streams[i] = 1
		}
	}

	// Count the files in each stream
	files := uint64(0)
	for _, v := range s.streams {
		files += v
	}

	if id == idSize {
		s.size = make([]uint64, files)
		k := 0

		for i := range s.streams {
			total := uint64(0)

			for j := uint64(1); j < s.streams[i]; j++ {
				if s.size[k], err = readUint64(r); err != nil {
					return nil, err
				}

				total += s.size[k]
				k++
			}

			s.size[k] = folder[i].unpackSize() - total
			k++
		}

		id, err = r.ReadByte()
		if err != nil {
			return nil, fmt.Errorf("readSubStreamsInfo: ReadByte error: %w", err)
		}
	}

	if id == idCRC {
		if s.digest, err = readCRC(r, files); err != nil {
			return nil, err
		}

		id, err = r.ReadByte()
		if err != nil {
			return nil, fmt.Errorf("readSubStreamsInfo: ReadByte error: %w", err)
		}
	}

	if id != idEnd {
		return nil, errUnexpectedID
	}

	return s, nil
}

//nolint:cyclop
func readStreamsInfo(r util.Reader) (*streamsInfo, error) {
	s := new(streamsInfo)

	id, err := r.ReadByte()
	if err != nil {
		return nil, fmt.Errorf("readStreamsInfo: ReadByte error: %w", err)
	}

	if id == idPackInfo {
		if s.packInfo, err = readPackInfo(r); err != nil {
			return nil, err
		}

		id, err = r.ReadByte()
		if err != nil {
			return nil, fmt.Errorf("readStreamsInfo: ReadByte error: %w", err)
		}
	}

	if id == idUnpackInfo {
		if s.unpackInfo, err = readUnpackInfo(r); err != nil {
			return nil, err
		}

		id, err = r.ReadByte()
		if err != nil {
			return nil, fmt.Errorf("readStreamsInfo: ReadByte error: %w", err)
		}
	}

	if id == idSubStreamsInfo {
		if s.unpackInfo == nil {
			return nil, errMissingUnpackInfo
		}

		if s.subStreamsInfo, err = readSubStreamsInfo(r, s.unpackInfo.folder); err != nil {
			return nil, err
		}

		id, err = r.ReadByte()
		if err != nil {
			return nil, fmt.Errorf("readStreamsInfo: ReadByte error: %w", err)
		}
	}

	if id != idEnd {
		return nil, errUnexpectedID
	}

	return s, nil
}

func readTimes(r util.Reader, count uint64) ([]time.Time, error) {
	defined, err := readOptionalBool(r, count)
	if err != nil {
		return nil, err
	}

	external, err := r.ReadByte()
	if err != nil {
		return nil, fmt.Errorf("readTimes: ReadByte error: %w", err)
	}

	if external > 0 {
		_, err := readUint64(r)
		if err != nil {
			return nil, err
		}
		// TODO Apparently we seek to this read offset and read the
		// folder information from there. Not clear if the offset is
		// absolute for the whole file, or relative to some known
		// position in the file. Cowardly waiting for an example
		return nil, errors.New("sevenzip: TODO readTimes external") //nolint:goerr113
	}

	times := make([]time.Time, count)

	for i := range defined {
		if defined[i] {
			var ft windows.Filetime
			if err := binary.Read(r, binary.LittleEndian, &ft); err != nil {
				return nil, fmt.Errorf("readTimes: Read error: %w", err)
			}

			times[i] = time.Unix(0, ft.Nanoseconds()).UTC()
		}
	}

	return times, nil
}

func splitNull(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	if i := bytes.IndexRune(data, rune(0)); i >= 0 {
		return i + 1, data[0:i], nil
	}

	if atEOF {
		return len(data), data, nil
	}

	return
}

func readNames(r util.Reader, count, length uint64) ([]string, error) {
	external, err := r.ReadByte()
	if err != nil {
		return nil, fmt.Errorf("readNames: ReadByte error: %w", err)
	}

	if external > 0 {
		_, err := readUint64(r)
		if err != nil {
			return nil, err
		}
		// TODO Apparently we seek to this read offset and read the
		// folder information from there. Not clear if the offset is
		// absolute for the whole file, or relative to some known
		// position in the file. Cowardly waiting for an example
		return nil, errors.New("sevenzip: TODO readNames external") //nolint:goerr113
	}

	utf16le := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM)
	scanner := bufio.NewScanner(transform.NewReader(io.LimitReader(r, int64(length-1)), utf16le.NewDecoder())) //nolint:gosec,lll
	scanner.Split(splitNull)

	names, i := make([]string, 0, count), uint64(0)
	for scanner.Scan() {
		names = append(names, scanner.Text())
		i++
	}

	if err = scanner.Err(); err != nil {
		return nil, fmt.Errorf("readNames: Scan error: %w", err)
	}

	if i != count {
		return nil, errWrongNumberOfFilenames
	}

	return names, nil
}

func readAttributes(r util.Reader, count uint64) ([]uint32, error) {
	defined, err := readOptionalBool(r, count)
	if err != nil {
		return nil, err
	}

	external, err := r.ReadByte()
	if err != nil {
		return nil, fmt.Errorf("readAttributes: ReadByte error: %w", err)
	}

	if external > 0 {
		_, err := readUint64(r)
		if err != nil {
			return nil, err
		}
		// TODO Apparently we seek to this read offset and read the
		// folder information from there. Not clear if the offset is
		// absolute for the whole file, or relative to some known
		// position in the file. Cowardly waiting for an example
		return nil, errors.New("sevenzip: TODO readAttributes external") //nolint:goerr113
	}

	attributes := make([]uint32, count)

	for i := range defined {
		if defined[i] {
			if err := binary.Read(r, binary.LittleEndian, &attributes[i]); err != nil {
				return nil, fmt.Errorf("readAttributes: Read error: %w", err)
			}
		}
	}

	return attributes, nil
}

//nolint:cyclop,funlen,gocognit,gocyclo
func readFilesInfo(r util.Reader) (*filesInfo, error) {
	f := new(filesInfo)

	files, err := readUint64(r)
	if err != nil {
		return nil, err
	}

	f.file = make([]FileHeader, files)

	var emptyStreams uint64

	for {
		property, err := r.ReadByte()
		if err != nil {
			return nil, fmt.Errorf("readFilesInfo: ReadByte error: %w", err)
		}

		if property == idEnd {
			break
		}

		length, err := readUint64(r)
		if err != nil {
			return nil, err
		}

		switch property {
		case idEmptyStream:
			empty, err := readBool(r, files)
			if err != nil {
				return nil, err
			}

			for i := range f.file {
				f.file[i].isEmptyStream = empty[i]

				if empty[i] {
					emptyStreams++
				}
			}
		case idEmptyFile:
			empty, err := readBool(r, emptyStreams)
			if err != nil {
				return nil, err
			}

			j := 0

			for i := range f.file {
				if f.file[i].isEmptyStream {
					f.file[i].isEmptyFile = empty[j]
					j++
				}
			}
		case idCTime:
			times, err := readTimes(r, files)
			if err != nil {
				return nil, err
			}

			for i, t := range times {
				f.file[i].Created = t
			}
		case idATime:
			times, err := readTimes(r, files)
			if err != nil {
				return nil, err
			}

			for i, t := range times {
				f.file[i].Accessed = t
			}
		case idMTime:
			times, err := readTimes(r, files)
			if err != nil {
				return nil, err
			}

			for i, t := range times {
				f.file[i].Modified = t
			}
		case idName:
			names, err := readNames(r, files, length)
			if err != nil {
				return nil, err
			}

			for i, n := range names {
				f.file[i].Name = n
			}
		case idWinAttributes:
			attributes, err := readAttributes(r, files)
			if err != nil {
				return nil, err
			}

			for i, a := range attributes {
				f.file[i].Attributes = a
			}
		case idStartPos:
			return nil, errors.New("sevenzip: TODO idStartPos") //nolint:goerr113
		case idDummy:
			if _, err := io.CopyN(io.Discard, r, int64(length)); err != nil { //nolint:gosec
				return nil, fmt.Errorf("readFilesInfo: CopyN error: %w", err)
			}
		default:
			return nil, errUnexpectedID
		}
	}

	return f, nil
}

//nolint:cyclop,funlen
func readHeader(r util.Reader) (*header, error) {
	h := new(header)

	id, err := r.ReadByte()
	if err != nil {
		return nil, fmt.Errorf("readHeader: ReadByte error: %w", err)
	}

	if id == idArchiveProperties {
		return nil, errors.New("sevenzip: TODO idArchiveProperties") //nolint:goerr113,revive

		//nolint:govet
		id, err = r.ReadByte()
		if err != nil {
			return nil, fmt.Errorf("readHeader: ReadByte error: %w", err)
		}
	}

	if id == idAdditionalStreamsInfo {
		return nil, errors.New("sevenzip: TODO idAdditionalStreamsInfo") //nolint:goerr113,revive

		//nolint:govet
		id, err = r.ReadByte()
		if err != nil {
			return nil, fmt.Errorf("readHeader: ReadByte error: %w", err)
		}
	}

	if id == idMainStreamsInfo {
		if h.streamsInfo, err = readStreamsInfo(r); err != nil {
			return nil, err
		}

		id, err = r.ReadByte()
		if err != nil {
			return nil, fmt.Errorf("readHeader: ReadByte error: %w", err)
		}
	}

	if id == idFilesInfo {
		if h.filesInfo, err = readFilesInfo(r); err != nil {
			return nil, err
		}

		id, err = r.ReadByte()
		if err != nil {
			return nil, fmt.Errorf("readHeader: ReadByte error: %w", err)
		}
	}

	if id != idEnd {
		return nil, errUnexpectedID
	}

	if h.streamsInfo == nil || h.filesInfo == nil {
		return h, nil
	}

	j := 0

	for i := range h.filesInfo.file {
		if h.filesInfo.file[i].isEmptyStream {
			continue
		}

		if h.streamsInfo.subStreamsInfo != nil {
			h.filesInfo.file[i].CRC32 = h.streamsInfo.subStreamsInfo.digest[j]
		}

		_, h.filesInfo.file[i].UncompressedSize = h.streamsInfo.FileFolderAndSize(j)
		j++
	}

	return h, nil
}

func readEncodedHeader(r util.Reader) (*header, error) {
	if id, err := r.ReadByte(); err != nil || id != idHeader {
		if err != nil {
			return nil, fmt.Errorf("readEncodedHeader: ReadByte error: %w", err)
		}

		return nil, errUnexpectedID
	}

	header, err := readHeader(r)
	if err != nil {
		return nil, err
	}

	return header, nil
}
