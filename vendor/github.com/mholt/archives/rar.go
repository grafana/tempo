package archives

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path"
	"strings"
	"time"

	"github.com/nwaples/rardecode/v2"
)

func init() {
	RegisterFormat(Rar{})
}

type rarReader interface {
	Next() (*rardecode.FileHeader, error)
	io.Reader
	io.WriterTo
}

type Rar struct {
	// If true, errors encountered during reading or writing
	// a file within an archive will be logged and the
	// operation will continue on remaining files.
	ContinueOnError bool

	// Password to open archives.
	Password string

	// Name for a multi-volume archive. When Name is specified,
	// the named file is extracted (rather than any io.Reader that
	// may be passed to Extract). If the archive is a multi-volume
	// archive, this name will also be used by the decoder to derive
	// the filename of the next volume in the volume set.
	Name string

	// FS is an fs.FS exposing the files of the archive. Unless Name is
	// also specified, this does nothing. When Name is also specified,
	// FS defines the fs.FS that from which the archive will be opened,
	// and in the case of a multi-volume archive, from where each subsequent
	// volume of the volume set will be loaded.
	//
	// Typically this should be a DirFS pointing at the directory containing
	// the volumes of the archive.
	FS fs.FS
}

func (Rar) Extension() string { return ".rar" }
func (Rar) MediaType() string { return "application/vnd.rar" }

func (r Rar) Match(_ context.Context, filename string, stream io.Reader) (MatchResult, error) {
	var mr MatchResult

	// match filename
	if strings.Contains(strings.ToLower(filename), r.Extension()) {
		mr.ByName = true
	}

	// match file header (there are two versions; allocate buffer for larger one)
	buf, err := readAtMost(stream, len(rarHeaderV5_0))
	if err != nil {
		return mr, err
	}

	matchedV1_5 := len(buf) >= len(rarHeaderV1_5) &&
		bytes.Equal(rarHeaderV1_5, buf[:len(rarHeaderV1_5)])
	matchedV5_0 := len(buf) >= len(rarHeaderV5_0) &&
		bytes.Equal(rarHeaderV5_0, buf[:len(rarHeaderV5_0)])

	mr.ByStream = matchedV1_5 || matchedV5_0

	return mr, nil
}

// Archive is not implemented for RAR because it is patent-encumbered.

func (r Rar) Extract(ctx context.Context, sourceArchive io.Reader, handleFile FileHandler) error {
	var options []rardecode.Option
	if r.Password != "" {
		options = append(options, rardecode.Password(r.Password))
	}

	if r.FS != nil {
		options = append(options, rardecode.FileSystem(r.FS))
	}

	var (
		rr  rarReader
		err error
	)

	// If a name has been provided, then the sourceArchive stream is ignored
	// and the archive is opened directly via the filesystem (or provided FS).
	if r.Name != "" {
		var or *rardecode.ReadCloser
		if or, err = rardecode.OpenReader(r.Name, options...); err == nil {
			rr = or
			defer or.Close()
		}
	} else {
		rr, err = rardecode.NewReader(sourceArchive, options...)
	}
	if err != nil {
		return err
	}

	// important to initialize to non-nil, empty value due to how fileIsIncluded works
	skipDirs := skipList{}

	for {
		if err := ctx.Err(); err != nil {
			return err // honor context cancellation
		}

		hdr, err := rr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			if r.ContinueOnError {
				log.Printf("[ERROR] Advancing to next file in rar archive: %v", err)
				continue
			}
			return err
		}
		if fileIsIncluded(skipDirs, hdr.Name) {
			continue
		}

		info := rarFileInfo{hdr}
		file := FileInfo{
			FileInfo:      info,
			Header:        hdr,
			NameInArchive: hdr.Name,
			Open: func() (fs.File, error) {
				return fileInArchive{io.NopCloser(rr), info}, nil
			},
		}

		err = handleFile(ctx, file)
		if errors.Is(err, fs.SkipAll) {
			break
		} else if errors.Is(err, fs.SkipDir) && file.IsDir() {
			skipDirs.add(hdr.Name)
		} else if err != nil {
			return fmt.Errorf("handling file: %s: %w", hdr.Name, err)
		}
	}

	return nil
}

// rarFileInfo satisfies the fs.FileInfo interface for RAR entries.
type rarFileInfo struct {
	fh *rardecode.FileHeader
}

func (rfi rarFileInfo) Name() string       { return path.Base(rfi.fh.Name) }
func (rfi rarFileInfo) Size() int64        { return rfi.fh.UnPackedSize }
func (rfi rarFileInfo) Mode() os.FileMode  { return rfi.fh.Mode() }
func (rfi rarFileInfo) ModTime() time.Time { return rfi.fh.ModificationTime }
func (rfi rarFileInfo) IsDir() bool        { return rfi.fh.IsDir }
func (rfi rarFileInfo) Sys() any           { return nil }

var (
	rarHeaderV1_5 = []byte("Rar!\x1a\x07\x00")     // v1.5
	rarHeaderV5_0 = []byte("Rar!\x1a\x07\x01\x00") // v5.0
)

// Interface guard
var _ Extractor = Rar{}
