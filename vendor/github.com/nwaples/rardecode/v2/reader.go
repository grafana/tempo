package rardecode

import (
	"bufio"
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"errors"
	"hash"
	"io"
	"math"
	"os"
	"time"
)

// FileHeader HostOS types
const (
	HostOSUnknown = 0
	HostOSMSDOS   = 1
	HostOSOS2     = 2
	HostOSWindows = 3
	HostOSUnix    = 4
	HostOSMacOS   = 5
	HostOSBeOS    = 6
)

const (
	maxPassword = int(128)
)

var (
	ErrShortFile        = errors.New("rardecode: decoded file too short")
	ErrInvalidFileBlock = errors.New("rardecode: invalid file block")
	ErrUnexpectedArcEnd = errors.New("rardecode: unexpected end of archive")
	ErrBadFileChecksum  = errors.New("rardecode: bad file checksum")
	ErrSolidOpen        = errors.New("rardecode: solid files don't support Open")
	ErrUnknownVersion   = errors.New("rardecode: unknown archive version")
)

// FileHeader represents a single file in a RAR archive.
type FileHeader struct {
	Name             string    // file name using '/' as the directory separator
	IsDir            bool      // is a directory
	Solid            bool      // is a solid file
	Encrypted        bool      // file contents are encrypted
	HeaderEncrypted  bool      // file header is encrypted
	HostOS           byte      // Host OS the archive was created on
	Attributes       int64     // Host OS specific file attributes
	PackedSize       int64     // packed file size (or first block if the file spans volumes)
	UnPackedSize     int64     // unpacked file size
	UnKnownSize      bool      // unpacked file size is not known
	ModificationTime time.Time // modification time (non-zero if set)
	CreationTime     time.Time // creation time (non-zero if set)
	AccessTime       time.Time // access time (non-zero if set)
	Version          int       // file version
}

// Mode returns an os.FileMode for the file, calculated from the Attributes field.
func (f *FileHeader) Mode() os.FileMode {
	var m os.FileMode

	if f.IsDir {
		m = os.ModeDir
	}
	if f.HostOS == HostOSWindows {
		if f.IsDir {
			m |= 0777
		} else if f.Attributes&1 > 0 {
			m |= 0444 // readonly
		} else {
			m |= 0666
		}
		return m
	}
	// assume unix perms for all remaining os types
	m |= os.FileMode(f.Attributes) & os.ModePerm

	// only check other bits on unix host created archives
	if f.HostOS != HostOSUnix {
		return m
	}

	if f.Attributes&0x200 != 0 {
		m |= os.ModeSticky
	}
	if f.Attributes&0x400 != 0 {
		m |= os.ModeSetgid
	}
	if f.Attributes&0x800 != 0 {
		m |= os.ModeSetuid
	}

	// Check for additional file types.
	if f.Attributes&0xF000 == 0xA000 {
		m |= os.ModeSymlink
	}
	return m
}

type byteReader interface {
	io.Reader
	bytes() ([]byte, error)
}

type bufByteReader struct {
	buf []byte
}

func (b *bufByteReader) Read(p []byte) (int, error) {
	if len(b.buf) == 0 {
		return 0, io.EOF
	}
	n := copy(p, b.buf)
	b.buf = b.buf[n:]
	return n, nil
}

func (b *bufByteReader) bytes() ([]byte, error) {
	if len(b.buf) == 0 {
		return nil, io.EOF
	}
	buf := b.buf
	b.buf = nil
	return buf, nil
}

func newBufByteReader(buf []byte) *bufByteReader {
	return &bufByteReader{buf: buf}
}

// packedFileReader provides sequential access to packed files in a RAR archive.
type packedFileReader struct {
	n int64 // bytes left in current data block
	v *volume
	r fileBlockReader
	h *fileBlockHeader // current file header
}

// init initializes a cloned packedFileReader
func (f *packedFileReader) init() error { return f.v.init() }

func (f *packedFileReader) clone() *packedFileReader {
	nr := &packedFileReader{n: f.n, h: f.h}
	nr.r = f.r.clone()
	nr.v = f.v.clone()
	return nr
}

func (f *packedFileReader) Close() error { return f.v.Close() }

// nextBlock reads the next file block in the current file at the current
// archive file position, or returns an error if there is a problem.
// It is invalid to call this when already at the last block in the current file.
func (f *packedFileReader) nextBlock() error {
	if f.h == nil {
		return io.EOF
	}
	// discard current block data
	if f.n > 0 {
		if err := f.v.discard(f.n); err != nil {
			return err
		}
		f.n = 0
	}
	if f.h.last {
		return io.EOF
	}
	h, err := f.r.next(f.v)
	if err != nil {
		if err == io.EOF {
			// archive ended, but file hasn't
			return ErrUnexpectedArcEnd
		}
		return err
	}
	if h.first || h.Name != f.h.Name {
		return ErrInvalidFileBlock
	}
	f.n = h.PackedSize
	f.h = h
	return nil
}

// next advances to the next packed file in the RAR archive.
func (f *packedFileReader) next() (*fileBlockHeader, error) {
	// skip to last block in current file
	var err error
	for err == nil {
		err = f.nextBlock()
	}
	if err != io.EOF {
		return nil, err
	}
	f.h, err = f.r.next(f.v) // get next file block
	if err != nil {
		return nil, err
	}
	if !f.h.first {
		return nil, ErrInvalidFileBlock
	}
	f.n = f.h.PackedSize
	return f.h, nil
}

// Read reads the packed data for the current file into p.
func (f *packedFileReader) Read(p []byte) (int, error) {
	for f.n == 0 {
		if err := f.nextBlock(); err != nil {
			return 0, err
		}
	}
	if int64(len(p)) > f.n {
		p = p[0:f.n]
	}
	n, err := f.v.Read(p)
	f.n -= int64(n)
	if err == io.EOF && f.n > 0 {
		return n, io.ErrUnexpectedEOF
	}
	if n > 0 {
		return n, nil
	}
	return n, err
}

func (f *packedFileReader) bytes() ([]byte, error) {
	for f.n == 0 {
		if err := f.nextBlock(); err != nil {
			return nil, err
		}
	}
	n := int(min(f.n, math.MaxInt))
	if k := f.v.br.Buffered(); k > 0 {
		n = min(k, n)
	} else {
		b, err := f.v.peek(n)
		if err != nil && err != bufio.ErrBufferFull {
			return nil, err
		}
		n = len(b)
	}
	b, err := f.v.readSlice(n)
	f.n -= int64(len(b))
	return b, err
}

func newPackedFileReader(r io.Reader, opts []Option) (*packedFileReader, error) {
	v, err := newVolume(r, opts)
	if err != nil {
		return nil, err
	}
	fbr, err := newFileBlockReader(v)
	if err != nil {
		return nil, err
	}
	return &packedFileReader{r: fbr, v: v}, nil
}

func openPackedFileReader(name string, opts []Option) (*packedFileReader, error) {
	v, err := openVolume(name, opts)
	if err != nil {
		return nil, err
	}
	fbr, err := newFileBlockReader(v)
	if err != nil {
		return nil, err
	}
	return &packedFileReader{r: fbr, v: v}, nil
}

type limitedReader struct {
	r        byteReader
	n        int64 // bytes remaining
	shortErr error // error returned when r returns io.EOF with n > 0
}

func (l *limitedReader) Read(p []byte) (int, error) {
	if l.n <= 0 {
		return 0, io.EOF
	}
	if int64(len(p)) > l.n {
		p = p[0:l.n]
	}
	n, err := l.r.Read(p)
	l.n -= int64(n)
	if err == io.EOF && l.n > 0 {
		return n, l.shortErr
	}
	return n, err
}

func (l *limitedReader) bytes() ([]byte, error) {
	b, err := l.r.bytes()
	if n := len(b); int64(n) > l.n {
		b = b[:int(l.n)]
	}
	l.n -= int64(len(b))
	return b, err
}

type checksumReader struct {
	r    byteReader
	hash hash.Hash
	pr   *packedFileReader
}

func (cr *checksumReader) eofError() error {
	// calculate file checksum
	h := cr.pr.h
	sum := cr.hash.Sum(nil)
	if !h.first && h.genKeys != nil {
		if err := h.genKeys(); err != nil {
			return err
		}
	}
	if len(h.hashKey) > 0 {
		mac := hmac.New(sha256.New, h.hashKey)
		_, _ = mac.Write(sum) // ignore error, should always succeed
		sum = mac.Sum(sum[:0])
		if len(h.sum) == 4 {
			// CRC32
			for i, v := range sum[4:] {
				sum[i&3] ^= v
			}
			sum = sum[:4]
		}
	}
	if !bytes.Equal(sum, h.sum) {
		return ErrBadFileChecksum
	}
	return io.EOF
}

func (cr *checksumReader) Read(p []byte) (int, error) {
	n, err := cr.r.Read(p)
	if n > 0 {
		if n, err = cr.hash.Write(p[:n]); err != nil {
			return n, err
		}
	}
	if err != io.EOF {
		return n, err
	}
	return n, cr.eofError()
}

func (cr *checksumReader) bytes() ([]byte, error) {
	b, err := cr.r.bytes()
	if len(b) > 0 {
		if _, err = cr.hash.Write(b); err != nil {
			return b, err
		}
	}
	if err != io.EOF {
		return b, err
	}
	return b, cr.eofError()
}

// Reader provides sequential access to files in a RAR archive.
type Reader struct {
	r  byteReader        // reader for current unpacked file
	dr *decodeReader     // reader for decoding and filters if file is compressed
	pr *packedFileReader // reader for current raw file bytes
}

// Read reads from the current file in the RAR archive.
func (r *Reader) Read(p []byte) (int, error) {
	if r.r == nil {
		err := r.nextFile()
		if err != nil {
			return 0, err
		}
	}
	return r.r.Read(p)
}

// WriteTo implements io.WriterTo.
func (r *Reader) WriteTo(w io.Writer) (int64, error) {
	if r.r == nil {
		err := r.nextFile()
		if err != nil {
			return 0, err
		}
	}
	var n int64
	b, err := r.r.bytes()
	for err == nil {
		var nn int
		nn, err = w.Write(b)
		n += int64(nn)
		if err == nil {
			b, err = r.r.bytes()
		}
	}
	if err == io.EOF {
		err = nil
	}
	return n, err
}

// Next advances to the next file in the archive.
func (r *Reader) Next() (*FileHeader, error) {
	// check if file is a compressed file in a solid archive
	if h := r.pr.h; h != nil && h.decVer > 0 && h.arcSolid {
		var err error
		if r.r == nil {
			// setup full file reader
			err = r.nextFile()
		}
		// decode and discard bytes
		for err == nil {
			_, err = r.dr.bytes()
		}
		if err != io.EOF {
			return nil, err
		}
	}
	// get next packed file
	h, err := r.pr.next()
	if err != nil {
		return nil, err
	}
	// Clear the reader as it will be setup on the next Read() or WriteTo().
	r.r = nil
	return &h.FileHeader, nil
}

func (r *Reader) nextFile() error {
	h := r.pr.h
	if h == nil {
		return io.EOF
	}
	// start with packed file reader
	r.r = r.pr
	// check for encryption
	if h.genKeys != nil {
		r.r = newAesDecryptReader(r.pr, h) // decrypt
	}
	// check for compression
	if h.decVer > 0 {
		if r.dr == nil {
			r.dr = new(decodeReader)
		}
		err := r.dr.init(r.r, h.decVer, h.winSize, !h.Solid, h.UnPackedSize)
		if err != nil {
			return err
		}
		r.r = r.dr
	}
	if h.UnPackedSize >= 0 && !h.UnKnownSize {
		// Limit reading to UnPackedSize as there may be padding
		r.r = &limitedReader{r.r, h.UnPackedSize, ErrShortFile}
	}
	if h.hash != nil {
		r.r = &checksumReader{r.r, h.hash(), r.pr}
	}
	return nil
}

// NewReader creates a Reader reading from r.
// NewReader only supports single volume archives.
// Multi-volume archives must use OpenReader.
func NewReader(r io.Reader, opts ...Option) (*Reader, error) {
	pr, err := newPackedFileReader(r, opts)
	if err != nil {
		return nil, err
	}
	return &Reader{pr: pr}, nil
}

// ReadCloser is a Reader that allows closing of the rar archive.
type ReadCloser struct {
	Reader
}

// Close closes the rar file.
func (rc *ReadCloser) Close() error {
	return rc.pr.Close()
}

// OpenReader opens a RAR archive specified by the name and returns a ReadCloser.
func OpenReader(name string, opts ...Option) (*ReadCloser, error) {
	pr, err := openPackedFileReader(name, opts)
	if err != nil {
		return nil, err
	}
	return &ReadCloser{Reader{pr: pr}}, nil
}

// File represents a file in a RAR archive
type File struct {
	FileHeader
	pr *packedFileReader
}

// Open returns an io.ReadCloser that provides access to the File's contents.
// Open is not supported on Solid File's as their contents depend on the decoding
// of the preceding files in the archive. Use OpenReader and Next to access Solid file
// contents instead.
func (f *File) Open() (io.ReadCloser, error) {
	if f.Solid {
		return nil, ErrSolidOpen
	}
	r := new(ReadCloser)
	r.pr = f.pr.clone()
	return r, r.pr.init()
}

// List returns a list of File's in the RAR archive specified by name.
func List(name string, opts ...Option) ([]*File, error) {
	r, err := OpenReader(name, opts...)
	if err != nil {
		return nil, err
	}
	pr := r.pr
	defer pr.Close()

	var fl []*File
	for {
		// get next file
		h, err := pr.next()
		if err != nil {
			if err == io.EOF {
				return fl, nil
			}
			return nil, err
		}

		// save information for File
		f := new(File)
		f.FileHeader = h.FileHeader
		f.pr = pr.clone()
		fl = append(fl, f)
	}
}
