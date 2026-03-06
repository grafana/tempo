package sevenzip

import (
	"bufio"
	"errors"
	"fmt"
	"hash"
	"hash/crc32"
	"io"
	"io/fs"
	"path"
	"time"

	"github.com/bodgit/plumbing"
	"github.com/bodgit/sevenzip/internal/util"
)

var (
	errAlgorithm             = errors.New("sevenzip: unsupported compression algorithm")
	errInvalidWhence         = errors.New("invalid whence")
	errNegativeSeek          = errors.New("negative seek")
	errSeekBackwards         = errors.New("cannot seek backwards")
	errSeekEOF               = errors.New("cannot seek beyond EOF")
	errMultipleOutputStreams = errors.New("more than one output stream")
	errNoBoundStream         = errors.New("cannot find bound stream")
	errNoUnboundStream       = errors.New("expecting one unbound output stream")
)

// CryptoReadCloser adds a Password method to decompressors.
type CryptoReadCloser interface {
	Password(password string) error
}

type signatureHeader struct {
	Signature [6]byte
	Major     byte
	Minor     byte
	CRC       uint32
}

type startHeader struct {
	Offset uint64
	Size   uint64
	CRC    uint32
}

type packInfo struct {
	position uint64
	streams  uint64
	size     []uint64
	digest   []uint32
}

type coder struct {
	id         []byte
	in, out    uint64
	properties []byte
}

type bindPair struct {
	in, out uint64
}

type folder struct {
	in, out       uint64
	packedStreams uint64
	coder         []*coder
	bindPair      []*bindPair
	size          []uint64
	packed        []uint64
}

func (f *folder) findInBindPair(i uint64) *bindPair {
	for _, v := range f.bindPair {
		if v.in == i {
			return v
		}
	}

	return nil
}

func (f *folder) findOutBindPair(i uint64) *bindPair {
	for _, v := range f.bindPair {
		if v.out == i {
			return v
		}
	}

	return nil
}

func (f *folder) coderReader(readers []io.ReadCloser, coder uint64, password string) (io.ReadCloser, bool, error) {
	dcomp := decompressor(f.coder[coder].id)
	if dcomp == nil {
		return nil, false, errAlgorithm
	}

	cr, err := dcomp(f.coder[coder].properties, f.size[coder], readers)
	if err != nil {
		return nil, false, err
	}

	crc, ok := cr.(CryptoReadCloser)
	if ok {
		if err = crc.Password(password); err != nil {
			return nil, true, fmt.Errorf("sevenzip: error setting password: %w", err)
		}
	}

	return plumbing.LimitReadCloser(cr, int64(f.size[coder])), ok, nil //nolint:gosec
}

type folderReadCloser struct {
	io.ReadCloser
	h             hash.Hash
	wc            *plumbing.WriteCounter
	size          int64
	hasEncryption bool
}

func (rc *folderReadCloser) Checksum() []byte {
	return rc.h.Sum(nil)
}

func (rc *folderReadCloser) Seek(offset int64, whence int) (int64, error) {
	var newo int64

	switch whence {
	case io.SeekStart:
		newo = offset
	case io.SeekCurrent:
		newo = int64(rc.wc.Count()) + offset //nolint:gosec
	case io.SeekEnd:
		newo = rc.Size() + offset
	default:
		return 0, errInvalidWhence
	}

	if newo < 0 {
		return 0, errNegativeSeek
	}

	if uint64(newo) < rc.wc.Count() {
		return 0, errSeekBackwards
	}

	if newo > rc.Size() {
		return 0, errSeekEOF
	}

	if _, err := io.CopyN(io.Discard, rc, newo-int64(rc.wc.Count())); err != nil { //nolint:gosec
		return 0, fmt.Errorf("sevenzip: error seeking: %w", err)
	}

	return newo, nil
}

func (rc *folderReadCloser) Size() int64 {
	return rc.size
}

func newFolderReadCloser(rc io.ReadCloser, size int64, hasEncryption bool) *folderReadCloser {
	nrc := new(folderReadCloser)
	nrc.h = crc32.NewIEEE()
	nrc.wc = new(plumbing.WriteCounter)
	nrc.ReadCloser = plumbing.TeeReadCloser(rc, io.MultiWriter(nrc.h, nrc.wc))
	nrc.size = size
	nrc.hasEncryption = hasEncryption

	return nrc
}

func (f *folder) unpackSize() uint64 {
	if len(f.size) == 0 {
		return 0
	}

	for i := len(f.size) - 1; i >= 0; i-- {
		if f.findOutBindPair(uint64(i)) == nil {
			return f.size[i]
		}
	}

	return f.size[len(f.size)-1]
}

type unpackInfo struct {
	folder []*folder
	digest []uint32
}

type subStreamsInfo struct {
	streams []uint64
	size    []uint64
	digest  []uint32
}

type streamsInfo struct {
	packInfo       *packInfo
	unpackInfo     *unpackInfo
	subStreamsInfo *subStreamsInfo
}

func (si *streamsInfo) Folders() int {
	if si != nil && si.unpackInfo != nil {
		return len(si.unpackInfo.folder)
	}

	return 0
}

func (si *streamsInfo) FileFolderAndSize(file int) (int, uint64) {
	total := uint64(0)

	var (
		folder  int
		streams uint64 = 1
	)

	if si.subStreamsInfo != nil {
		for folder, streams = range si.subStreamsInfo.streams {
			total += streams
			if uint64(file) < total { //nolint:gosec
				break
			}
		}
	}

	if streams == 1 {
		return folder, si.unpackInfo.folder[folder].size[len(si.unpackInfo.folder[folder].coder)-1]
	}

	return folder, si.subStreamsInfo.size[file]
}

func (si *streamsInfo) folderOffset(folder int) int64 {
	offset := uint64(0)

	for i, k := 0, uint64(0); i < folder; i++ {
		for j := k; j < k+si.unpackInfo.folder[i].packedStreams; j++ {
			offset += si.packInfo.size[j]
		}

		k += si.unpackInfo.folder[i].packedStreams
	}

	return int64(si.packInfo.position + offset) //nolint:gosec
}

//nolint:cyclop,funlen,lll
func (si *streamsInfo) FolderReader(r io.ReaderAt, folder int, password string) (*folderReadCloser, uint32, bool, error) {
	f := si.unpackInfo.folder[folder]
	in := make([]io.ReadCloser, f.in)
	out := make([]io.ReadCloser, f.out)

	packedOffset := 0
	for i := 0; i < folder; i++ {
		packedOffset += len(si.unpackInfo.folder[i].packed)
	}

	offset := int64(0)

	for i, input := range f.packed {
		size := int64(si.packInfo.size[packedOffset+i]) //nolint:gosec
		in[input] = util.NopCloser(bufio.NewReader(io.NewSectionReader(r, si.folderOffset(folder)+offset, size)))
		offset += size
	}

	var (
		hasEncryption bool
		input, output uint64
	)

	for i, c := range f.coder {
		if c.out != 1 {
			return nil, 0, hasEncryption, errMultipleOutputStreams
		}

		for j := input; j < input+c.in; j++ {
			if in[j] != nil {
				continue
			}

			bp := f.findInBindPair(j)
			if bp == nil || out[bp.out] == nil {
				return nil, 0, hasEncryption, errNoBoundStream
			}

			in[j] = out[bp.out]
		}

		var (
			isEncrypted bool
			err         error
		)

		out[output], isEncrypted, err = f.coderReader(in[input:input+c.in], uint64(i), password) //nolint:gosec
		if err != nil {
			return nil, 0, hasEncryption, err
		}

		if isEncrypted {
			hasEncryption = true
		}

		input += c.in
		output += c.out
	}

	unbound := make([]uint64, 0, f.out)

	for i := uint64(0); i < f.out; i++ {
		if bp := f.findOutBindPair(i); bp == nil {
			unbound = append(unbound, i)
		}
	}

	if len(unbound) != 1 || out[unbound[0]] == nil {
		return nil, 0, hasEncryption, errNoUnboundStream
	}

	fr := newFolderReadCloser(out[unbound[0]], int64(f.unpackSize()), hasEncryption) //nolint:gosec

	if si.unpackInfo.digest != nil {
		return fr, si.unpackInfo.digest[folder], hasEncryption, nil
	}

	return fr, 0, hasEncryption, nil
}

type filesInfo struct {
	file []FileHeader
}

type header struct {
	streamsInfo *streamsInfo
	filesInfo   *filesInfo
}

// FileHeader describes a file within a 7-zip file.
type FileHeader struct {
	Name             string
	Created          time.Time
	Accessed         time.Time
	Modified         time.Time
	Attributes       uint32
	CRC32            uint32
	UncompressedSize uint64

	// Stream is an opaque identifier representing the compressed stream
	// that contains the file. Any File with the same value can be assumed
	// to be stored within the same stream.
	Stream int

	isEmptyStream bool
	isEmptyFile   bool
}

// FileInfo returns an fs.FileInfo for the FileHeader.
func (h *FileHeader) FileInfo() fs.FileInfo {
	return headerFileInfo{h}
}

type headerFileInfo struct {
	fh *FileHeader
}

func (fi headerFileInfo) Name() string       { return path.Base(fi.fh.Name) }
func (fi headerFileInfo) Size() int64        { return int64(fi.fh.UncompressedSize) } //nolint:gosec
func (fi headerFileInfo) IsDir() bool        { return fi.Mode().IsDir() }
func (fi headerFileInfo) ModTime() time.Time { return fi.fh.Modified.UTC() }
func (fi headerFileInfo) Mode() fs.FileMode  { return fi.fh.Mode() }
func (fi headerFileInfo) Type() fs.FileMode  { return fi.fh.Mode().Type() }
func (fi headerFileInfo) Sys() interface{}   { return fi.fh }

func (fi headerFileInfo) Info() (fs.FileInfo, error) { return fi, nil }

const (
	// Unix constants. The specification doesn't mention them,
	// but these seem to be the values agreed on by tools.
	sIFMT   = 0xf000
	sIFSOCK = 0xc000
	sIFLNK  = 0xa000
	sIFREG  = 0x8000
	sIFBLK  = 0x6000
	sIFDIR  = 0x4000
	sIFCHR  = 0x2000
	sIFIFO  = 0x1000
	sISUID  = 0x800
	sISGID  = 0x400
	sISVTX  = 0x200

	msdosDir      = 0x10
	msdosReadOnly = 0x01
)

// Mode returns the permission and mode bits for the FileHeader.
func (h *FileHeader) Mode() (mode fs.FileMode) {
	// Prefer the POSIX attributes if they're present
	if h.Attributes&0xf0000000 != 0 {
		mode = unixModeToFileMode(h.Attributes >> 16)
	} else {
		mode = msdosModeToFileMode(h.Attributes)
	}

	return
}

func msdosModeToFileMode(m uint32) (mode fs.FileMode) {
	if m&msdosDir != 0 {
		mode = fs.ModeDir | 0o777
	} else {
		mode = 0o666
	}

	if m&msdosReadOnly != 0 {
		mode &^= 0o222
	}

	return mode
}

//nolint:cyclop
func unixModeToFileMode(m uint32) fs.FileMode {
	mode := fs.FileMode(m & 0o777)

	switch m & sIFMT {
	case sIFBLK:
		mode |= fs.ModeDevice
	case sIFCHR:
		mode |= fs.ModeDevice | fs.ModeCharDevice
	case sIFDIR:
		mode |= fs.ModeDir
	case sIFIFO:
		mode |= fs.ModeNamedPipe
	case sIFLNK:
		mode |= fs.ModeSymlink
	case sIFREG:
		// nothing to do
	case sIFSOCK:
		mode |= fs.ModeSocket
	}

	if m&sISGID != 0 {
		mode |= fs.ModeSetgid
	}

	if m&sISUID != 0 {
		mode |= fs.ModeSetuid
	}

	if m&sISVTX != 0 {
		mode |= fs.ModeSticky
	}

	return mode
}
