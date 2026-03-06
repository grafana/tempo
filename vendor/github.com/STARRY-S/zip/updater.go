package zip

import (
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"path/filepath"
	"slices"
	"strings"
)

const bufferSize int64 = 1 << 20 // 1M

// AppendMode specifies the way to append new file to existing zip archive.
type AppendMode int

const (
	// ZIP_APPEND_OVERWRITE removes the existing file data and append the new
	// data to the end of the zip archive.
	APPEND_MODE_OVERWRITE AppendMode = iota

	// ZIP_APPEND_KEEP_ORIGINAL will keep the original file data and only
	// write the new file data at the end of the existing zip archive file.
	// This mode will keep multiple file with same name into one archive file.
	APPEND_MODE_KEEP_ORIGINAL
)

// sectionReaderWriter implements [io.Reader], [io.Writer], [io.Seeker],
// [io.ReaderAt], [io.WriterAt] interfaces based on [io.ReadWriteSeeker].
type sectionReaderWriter struct {
	rws io.ReadWriteSeeker
}

func newSectionReaderWriter(rws io.ReadWriteSeeker) *sectionReaderWriter {
	return &sectionReaderWriter{
		rws: rws,
	}
}

func (s *sectionReaderWriter) ReadAt(p []byte, offset int64) (int, error) {
	currOffset, err := s.rws.Seek(0, io.SeekCurrent)
	if err != nil {
		return 0, err
	}
	defer s.rws.Seek(currOffset, io.SeekStart)
	_, err = s.rws.Seek(offset, io.SeekStart)
	if err != nil {
		return 0, err
	}
	return s.rws.Read(p)
}

func (s *sectionReaderWriter) WriteAt(p []byte, offset int64) (n int, err error) {
	currOffset, err := s.rws.Seek(0, io.SeekCurrent)
	if err != nil {
		return 0, err
	}
	defer s.rws.Seek(currOffset, io.SeekStart)
	_, err = s.rws.Seek(offset, io.SeekStart)
	if err != nil {
		return 0, err
	}
	return s.rws.Write(p)
}

func (s *sectionReaderWriter) Seek(offset int64, whence int) (int64, error) {
	return s.rws.Seek(offset, whence)
}

func (s *sectionReaderWriter) Read(p []byte) (n int, err error) {
	return s.rws.Read(p)
}

func (s *sectionReaderWriter) Write(p []byte) (n int, err error) {
	return s.rws.Write(p)
}

func (s *sectionReaderWriter) offset() (int64, error) {
	return s.rws.Seek(0, io.SeekCurrent)
}

type Directory struct {
	FileHeader
	offset int64 // header offset
}

func (d *Directory) HeaderOffset() int64 {
	return d.offset
}

// Updater allows to modify & append files into an existing zip archive without
// decompress the whole file.
type Updater struct {
	rw          *sectionReaderWriter
	offset      int64
	dir         []*header
	last        *fileWriter
	closed      bool
	compressors map[uint16]Compressor
	comment     string

	// Some JAR files are zip files with a prefix that is a bash script.
	// The baseOffset field is the start of the zip file proper.
	baseOffset int64
	// dirOffset is the offset to write the directory record.
	// Note that the dirOffset may not equal to the last file data end offset.
	dirOffset int64
}

// NewUpdater returns a new Updater from [io.ReadWriteSeeker], which is
// assumed to have the given size in bytes.
func NewUpdater(rws io.ReadWriteSeeker) (*Updater, error) {
	size, err := rws.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, err
	}
	zu := &Updater{
		rw: newSectionReaderWriter(rws),
	}
	if err = zu.init(size); err != nil && err != ErrInsecurePath {
		return nil, err
	}
	return zu, nil
}

func (u *Updater) init(size int64) error {
	end, baseOffset, err := readDirectoryEnd(u.rw, size)
	if err != nil {
		return err
	}
	u.baseOffset = baseOffset
	u.dirOffset = int64(end.directoryOffset)
	// Since the number of directory records is not validated, it is not
	// safe to preallocate r.File without first checking that the specified
	// number of files is reasonable, since a malformed archive may
	// indicate it contains up to 1 << 128 - 1 files. Since each file has a
	// header which will be _at least_ 30 bytes we can safely preallocate
	// if (data size / 30) >= end.directoryRecords.
	if end.directorySize < uint64(size) && (uint64(size)-end.directorySize)/30 >= end.directoryRecords {
		u.dir = make([]*header, 0, end.directoryRecords)
	}
	u.comment = end.comment
	if _, err = u.rw.Seek(u.baseOffset+int64(end.directoryOffset), io.SeekStart); err != nil {
		return err
	}

	// The count of files inside a zip is truncated to fit in a uint16.
	// Gloss over this by reading headers until we encounter
	// a bad one, and then only report an ErrFormat or UnexpectedEOF if
	// the file count modulo 65536 is incorrect.
	for {
		f := &File{zip: nil, zipr: u.rw}
		err = readDirectoryHeader(f, u.rw)
		if err == ErrFormat || err == io.ErrUnexpectedEOF {
			break
		}
		if err != nil {
			return err
		}
		f.headerOffset += u.baseOffset
		h := &header{
			FileHeader: &f.FileHeader,
			offset:     uint64(f.headerOffset),
		}
		u.dir = append(u.dir, h)
	}
	if uint16(len(u.dir)) != uint16(end.directoryRecords) { // only compare 16 bits here
		// Return the readDirectoryHeader error if we read
		// the wrong number of directory entries.
		return err
	}

	// Ensure the directory record is ordered by file header offset.
	slices.SortFunc(u.dir, sortDirectoryFunc)
	for _, d := range u.dir {
		if d.Name == "" {
			// Zip permits an empty file name field.
			continue
		}
		// The zip specification states that names must use forward slashes,
		// so consider any backslashes in the name insecure.
		if !filepath.IsLocal(d.Name) || strings.Contains(d.Name, "\\") {
			return ErrInsecurePath
		}
	}
	return nil
}

// Append adds a file to the zip file using the provided name.
// It returns a [Writer] to which the file contents should be written.
// The file contents will be compressed using the Deflate method.
// The name must be a relative path: it must not start with a drive
// letter (e.g. C:) or leading slash, and only forward slashes are
// allowed. To create a directory instead of a file, add a trailing
// slash to the name.
//
// If mode is set to [APPEND_MODE_OVERWRITE], and file name already exists
// in the zip archive, Append will delete the existing file data and write the
// new file data at the end of the zip file.
//
// If mode is set to [APPEND_MODE_KEEP_ORIGINAL], the existing data won't be
// deleted from the zip file and Append only write the file data with the same
// file name at the end of the zip file.
//
// The file's contents must be written to the io.Writer before the next
// call to [Append], [AppendHeader], or [Close].
func (u *Updater) Append(name string, mode AppendMode) (io.Writer, error) {
	h := &FileHeader{
		Name:   name,
		Method: Deflate,
	}
	return u.AppendHeader(h, mode)
}

func (u *Updater) prepare(fh *FileHeader) error {
	if u.last != nil && !u.last.closed {
		if err := u.last.close(); err != nil {
			return err
		}
		offset, err := u.rw.offset()
		if err != nil {
			return err
		}
		if u.dirOffset < offset {
			u.dirOffset = offset
		}
	}
	if len(u.dir) > 0 && u.dir[len(u.dir)-1].FileHeader == fh {
		// See https://golang.org/issue/11144 confusion.
		return errors.New("archive/zip: invalid duplicate FileHeader")
	}
	return nil
}

// AppendHeader adds a file to the zip archive using the provided [FileHeader]
// for the file metadata to the specific offset.
// Writer takes ownership of fh and may mutate its fields.
// The caller must not modify fh after calling CreateHeader.
//
// If the file name of the [FileHeader] already exists in the zip file,
// AppendHeader will remove the existing file data and the new file data will
// write at the end of the archive file.
//
// It should be noted that the size of the newly appended file size should be
// larger than the size of the replaced file. Especially when using the Deflate
// compression method, the compressed data size should be larger than the
// original file data size.
func (u *Updater) AppendHeader(fh *FileHeader, mode AppendMode) (io.Writer, error) {
	if err := u.prepare(fh); err != nil {
		return nil, err
	}

	var err error
	var offset int64 = -1
	var existingDirIndex int = -1
	if mode == APPEND_MODE_OVERWRITE {
		for i, d := range u.dir {
			if d.Name == fh.Name {
				offset = int64(d.offset)
				existingDirIndex = i
				break
			}
		}
	}
	if offset < 0 {
		offset = u.dirOffset
	}
	if existingDirIndex >= 0 {
		if offset, err = u.removeFile(existingDirIndex); err != nil {
			return nil, err
		}
	}

	// Seek the file offset.
	if _, err := u.rw.Seek(offset, io.SeekStart); err != nil {
		return nil, err
	}
	u.offset = offset

	// The ZIP format has a sad state of affairs regarding character encoding.
	// Officially, the name and comment fields are supposed to be encoded
	// in CP-437 (which is mostly compatible with ASCII), unless the UTF-8
	// flag bit is set. However, there are several problems:
	//
	//	* Many ZIP readers still do not support UTF-8.
	//	* If the UTF-8 flag is cleared, several readers simply interpret the
	//	name and comment fields as whatever the local system encoding is.
	//
	// In order to avoid breaking readers without UTF-8 support,
	// we avoid setting the UTF-8 flag if the strings are CP-437 compatible.
	// However, if the strings require multibyte UTF-8 encoding and is a
	// valid UTF-8 string, then we set the UTF-8 bit.
	//
	// For the case, where the user explicitly wants to specify the encoding
	// as UTF-8, they will need to set the flag bit themselves.
	utf8Valid1, utf8Require1 := detectUTF8(fh.Name)
	utf8Valid2, utf8Require2 := detectUTF8(fh.Comment)
	switch {
	case fh.NonUTF8:
		fh.Flags &^= 0x800
	case (utf8Require1 || utf8Require2) && (utf8Valid1 && utf8Valid2):
		fh.Flags |= 0x800
	}

	fh.CreatorVersion = fh.CreatorVersion&0xff00 | zipVersion20 // preserve compatibility byte
	fh.ReaderVersion = zipVersion20

	// If Modified is set, this takes precedence over MS-DOS timestamp fields.
	if !fh.Modified.IsZero() {
		// Contrary to the FileHeader.SetModTime method, we intentionally
		// do not convert to UTC, because we assume the user intends to encode
		// the date using the specified timezone. A user may want this control
		// because many legacy ZIP readers interpret the timestamp according
		// to the local timezone.
		//
		// The timezone is only non-UTC if a user directly sets the Modified
		// field directly themselves. All other approaches sets UTC.
		fh.ModifiedDate, fh.ModifiedTime = timeToMsDosTime(fh.Modified)

		// Use "extended timestamp" format since this is what Info-ZIP uses.
		// Nearly every major ZIP implementation uses a different format,
		// but at least most seem to be able to understand the other formats.
		//
		// This format happens to be identical for both local and central header
		// if modification time is the only timestamp being encoded.
		var mbuf [9]byte // 2*SizeOf(uint16) + SizeOf(uint8) + SizeOf(uint32)
		mt := uint32(fh.Modified.Unix())
		eb := writeBuf(mbuf[:])
		eb.uint16(extTimeExtraID)
		eb.uint16(5)  // Size: SizeOf(uint8) + SizeOf(uint32)
		eb.uint8(1)   // Flags: ModTime
		eb.uint32(mt) // ModTime
		fh.Extra = append(fh.Extra, mbuf[:]...)
	}

	var (
		ow io.Writer
		fw *fileWriter
	)
	h := &header{
		FileHeader: fh,
		offset:     uint64(u.offset),
	}
	if strings.HasSuffix(fh.Name, "/") {
		// Set the compression method to Store to ensure data length is truly zero,
		// which the writeHeader method always encodes for the size fields.
		// This is necessary as most compression formats have non-zero lengths
		// even when compressing an empty string.
		fh.Method = Store
		fh.Flags &^= 0x8 // we will not write a data descriptor

		// Explicitly clear sizes as they have no meaning for directories.
		fh.CompressedSize = 0
		fh.CompressedSize64 = 0
		fh.UncompressedSize = 0
		fh.UncompressedSize64 = 0

		ow = dirWriter{}
	} else {
		fh.Flags |= 0x8 // we will write a data descriptor

		fw = &fileWriter{
			zipw:      u.rw,
			compCount: &countWriter{w: u.rw},
			crc32:     crc32.NewIEEE(),
		}
		comp := u.compressor(fh.Method)
		if comp == nil {
			return nil, ErrAlgorithm
		}
		var err error
		fw.comp, err = comp(fw.compCount)
		if err != nil {
			return nil, err
		}
		fw.rawCount = &countWriter{w: fw.comp}
		fw.header = h
		ow = fw
	}
	u.dir = append(u.dir, h)
	// No need to re-sort u.dir here since the new created header is write
	// to the end of the files.
	if err := writeHeader(u.rw, h); err != nil {
		return nil, err
	}
	// If we're creating a directory, fw is nil.
	u.last = fw
	offset, err = u.rw.offset()
	if err != nil {
		return nil, err
	}
	if u.dirOffset < offset {
		u.dirOffset = offset
	}

	return ow, nil
}

// removeFile removes file in zip by rewinding data and directory record.
func (u *Updater) removeFile(dirIndex int) (int64, error) {
	// start is the file header offset.
	var start = int64(u.dir[dirIndex].offset)
	// end is the next file header offset or directory offset.
	var end int64
	if dirIndex == len(u.dir)-1 {
		end = u.dirOffset
	} else {
		end = int64(u.dir[dirIndex+1].offset)
	}
	// size is the file header and compressed data size.
	var size = end - start

	// Allocate buffer to rewind file data.
	var buffer = make([]byte, bufferSize)
	var rp int64 = end   // read point
	var wp int64 = start // write point
	// Rewind data in buffer size block.
	for rp < u.dirOffset-bufferSize {
		n, err := u.rw.ReadAt(buffer, rp)
		if err != nil {
			return 0, fmt.Errorf("zip: rewind data: ReadAt: %w", err)
		}
		_, err = u.rw.WriteAt(buffer[:n], wp)
		if err != nil {
			return 0, fmt.Errorf("zip: rewind data: WriteAt: %w", err)
		}
		rp += int64(n)
		wp += int64(n)
	}
	// Rewind remaining data that smaller than the buffer size block.
	if rp < u.dirOffset {
		n, err := u.rw.ReadAt(buffer[:u.dirOffset-rp], rp)
		if err != nil {
			return 0, fmt.Errorf("zip: rewind data: ReadAt: %w", err)
		}
		_, err = u.rw.WriteAt(buffer[:n], wp)
		if err != nil {
			return 0, fmt.Errorf("zip: rewind data: ReadAt: %w", err)
		}
		rp += int64(n)
		wp += int64(n)
		// assert: rewind data before directory record
		if rp != u.dirOffset {
			return 0, errors.New("zip: rewind data: read data before directory failed")
		}
	}
	// Remove deleted file directory record.
	u.dir = append(u.dir[:dirIndex], u.dir[dirIndex+1:len(u.dir)]...)
	// Update the file header offset in directory record.
	for i := dirIndex; i < len(u.dir); i++ {
		u.dir[i].offset -= uint64(size)
		u.dir[i].Extra = nil // Will re-generate zip64 extra data when calling
	}
	return wp, nil
}

func (u *Updater) compressor(method uint16) Compressor {
	comp := u.compressors[method]
	if comp == nil {
		comp = compressor(method)
	}
	return comp
}

func (u *Updater) SetComment(comment string) error {
	if len(comment) > uint16max {
		return errors.New("zip: Writer.Comment too long")
	}
	u.comment = comment
	return nil
}

func (u *Updater) GetComment() string {
	return u.comment
}

func (u *Updater) Close() error {
	if u.last != nil && !u.last.closed {
		if err := u.last.close(); err != nil {
			return err
		}
		u.last = nil
	}
	if u.closed {
		return errors.New("zip: updater closed twice")
	}
	u.closed = true

	// write central directory
	start, err := u.rw.offset()
	if err != nil {
		return err
	}
	if start < u.dirOffset {
		// Make data to `\0` between the last file and the diretory record.
		// NOTE: this step is not mandatory but will make the file data clean.
		var buffSize int64
		var buffer []byte
		size := u.dirOffset - start
		if u.dirOffset-start > bufferSize {
			buffer = make([]byte, bufferSize)
			buffSize = bufferSize
		} else {
			buffer = make([]byte, size)
			buffSize = size
		}
		var wp = start
		_, err = u.rw.Seek(wp, io.SeekStart)
		if err != nil {
			return err
		}
		// Write `\0` in block size.
		for wp < u.dirOffset-buffSize {
			n, err := u.rw.Write(buffer)
			if err != nil {
				return err
			}
			wp += int64(n)
		}
		if wp < u.dirOffset {
			if _, err := u.rw.Write(buffer[:u.dirOffset-wp]); err != nil {
				return err
			}
		}
		start = u.dirOffset
	}
	for _, h := range u.dir {
		var buf []byte = make([]byte, directoryHeaderLen)
		b := writeBuf(buf)
		b.uint32(uint32(directoryHeaderSignature))
		b.uint16(h.CreatorVersion)
		b.uint16(h.ReaderVersion)
		b.uint16(h.Flags)
		b.uint16(h.Method)
		b.uint16(h.ModifiedTime)
		b.uint16(h.ModifiedDate)
		b.uint32(h.CRC32)
		if h.isZip64() || h.offset >= uint32max {
			// the file needs a zip64 header. store maxint in both
			// 32 bit size fields (and offset later) to signal that the
			// zip64 extra header should be used.
			b.uint32(uint32max) // compressed size
			b.uint32(uint32max) // uncompressed size

			// append a zip64 extra block to Extra
			var buf [28]byte // 2x uint16 + 3x uint64
			eb := writeBuf(buf[:])
			eb.uint16(zip64ExtraID)
			eb.uint16(24) // size = 3x uint64
			eb.uint64(h.UncompressedSize64)
			eb.uint64(h.CompressedSize64)
			eb.uint64(uint64(h.offset))
			h.Extra = append(h.Extra, buf[:]...)
		} else {
			b.uint32(h.CompressedSize)
			b.uint32(h.UncompressedSize)
		}

		b.uint16(uint16(len(h.Name)))
		b.uint16(uint16(len(h.Extra)))
		b.uint16(uint16(len(h.Comment)))
		b = b[4:] // skip disk number start and internal file attr (2x uint16)
		b.uint32(h.ExternalAttrs)
		if h.offset > uint32max {
			b.uint32(uint32max)
		} else {
			b.uint32(uint32(h.offset))
		}
		if _, err := u.rw.Write(buf); err != nil {
			return err
		}
		if _, err := io.WriteString(u.rw, h.Name); err != nil {
			return err
		}
		if _, err := u.rw.Write(h.Extra); err != nil {
			return err
		}
		if _, err := io.WriteString(u.rw, h.Comment); err != nil {
			return err
		}
	}
	end, err := u.rw.offset()
	if err != nil {
		return err
	}

	records := uint64(len(u.dir))
	size := uint64(end - start)
	offset := uint64(start)

	if records >= uint16max || size >= uint32max || offset >= uint32max {
		var buf [directory64EndLen + directory64LocLen]byte
		b := writeBuf(buf[:])

		// zip64 end of central directory record
		b.uint32(directory64EndSignature)
		b.uint64(directory64EndLen - 12) // length minus signature (uint32) and length fields (uint64)
		b.uint16(zipVersion45)           // version made by
		b.uint16(zipVersion45)           // version needed to extract
		b.uint32(0)                      // number of this disk
		b.uint32(0)                      // number of the disk with the start of the central directory
		b.uint64(records)                // total number of entries in the central directory on this disk
		b.uint64(records)                // total number of entries in the central directory
		b.uint64(size)                   // size of the central directory
		b.uint64(offset)                 // offset of start of central directory with respect to the starting disk number

		// zip64 end of central directory locator
		b.uint32(directory64LocSignature)
		b.uint32(0)           // number of the disk with the start of the zip64 end of central directory
		b.uint64(uint64(end)) // relative offset of the zip64 end of central directory record
		b.uint32(1)           // total number of disks

		if _, err := u.rw.Write(buf[:]); err != nil {
			return err
		}

		// store max values in the regular end record to signal
		// that the zip64 values should be used instead
		records = uint16max
		size = uint32max
		offset = uint32max
	}

	// write end record
	var buf [directoryEndLen]byte
	b := writeBuf(buf[:])
	b.uint32(uint32(directoryEndSignature))
	b = b[4:]                        // skip over disk number and first disk number (2x uint16)
	b.uint16(uint16(records))        // number of entries this disk
	b.uint16(uint16(records))        // number of entries total
	b.uint32(uint32(size))           // size of directory
	b.uint32(uint32(offset))         // start of directory
	b.uint16(uint16(len(u.comment))) // byte size of EOCD comment
	if _, err := u.rw.Write(buf[:]); err != nil {
		return err
	}
	if _, err := io.WriteString(u.rw, u.comment); err != nil {
		return err
	}

	return nil
}

func sortDirectoryFunc(a, b *header) int {
	switch {
	case a.offset > b.offset:
		return 1
	case a.offset < b.offset:
		return -1
	}
	return 0
}
