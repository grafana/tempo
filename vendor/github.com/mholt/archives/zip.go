package archives

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"path"
	"strings"

	szip "github.com/STARRY-S/zip"
	"golang.org/x/text/encoding"

	"github.com/dsnet/compress/bzip2"
	"github.com/klauspost/compress/zip"
	"github.com/klauspost/compress/zstd"
	"github.com/ulikunitz/xz"
)

func init() {
	RegisterFormat(Zip{})

	// TODO: What about custom flate levels too
	zip.RegisterCompressor(ZipMethodBzip2, func(out io.Writer) (io.WriteCloser, error) {
		return bzip2.NewWriter(out, &bzip2.WriterConfig{ /*TODO: Level: z.CompressionLevel*/ })
	})
	zip.RegisterCompressor(ZipMethodZstd, func(out io.Writer) (io.WriteCloser, error) {
		return zstd.NewWriter(out)
	})
	zip.RegisterCompressor(ZipMethodXz, func(out io.Writer) (io.WriteCloser, error) {
		return xz.NewWriter(out)
	})

	zip.RegisterDecompressor(ZipMethodBzip2, func(r io.Reader) io.ReadCloser {
		bz2r, err := bzip2.NewReader(r, nil)
		if err != nil {
			return nil
		}
		return bz2r
	})
	zip.RegisterDecompressor(ZipMethodZstd, func(r io.Reader) io.ReadCloser {
		zr, err := zstd.NewReader(r)
		if err != nil {
			return nil
		}
		return zr.IOReadCloser()
	})
	zip.RegisterDecompressor(ZipMethodXz, func(r io.Reader) io.ReadCloser {
		xr, err := xz.NewReader(r)
		if err != nil {
			return nil
		}
		return io.NopCloser(xr)
	})
}

type Zip struct {
	// Only compress files which are not already in a
	// compressed format (determined simply by examining
	// file extension).
	SelectiveCompression bool

	// The method or algorithm for compressing stored files.
	Compression uint16

	// If true, errors encountered during reading or writing
	// a file within an archive will be logged and the
	// operation will continue on remaining files.
	ContinueOnError bool

	// For files in zip archives that do not have UTF-8
	// encoded filenames and comments, specify the character
	// encoding here.
	TextEncoding encoding.Encoding
}

func (Zip) Extension() string { return ".zip" }
func (Zip) MediaType() string { return "application/zip" }

func (z Zip) Match(_ context.Context, filename string, stream io.Reader) (MatchResult, error) {
	var mr MatchResult

	// match filename
	if strings.Contains(strings.ToLower(filename), z.Extension()) {
		mr.ByName = true
	}

	// match file header
	for _, hdr := range zipHeaders {
		buf, err := readAtMost(stream, len(hdr))
		if err != nil {
			return mr, err
		}
		if bytes.Equal(buf, hdr) {
			mr.ByStream = true
			break
		}
	}

	return mr, nil
}

func (z Zip) Archive(ctx context.Context, output io.Writer, files []FileInfo) error {
	zw := zip.NewWriter(output)
	defer zw.Close()

	for i, file := range files {
		if err := z.archiveOneFile(ctx, zw, i, file); err != nil {
			return err
		}
	}

	return nil
}

func (z Zip) ArchiveAsync(ctx context.Context, output io.Writer, jobs <-chan ArchiveAsyncJob) error {
	zw := zip.NewWriter(output)
	defer zw.Close()

	var i int
	for job := range jobs {
		job.Result <- z.archiveOneFile(ctx, zw, i, job.File)
		i++
	}

	return nil
}

func (z Zip) archiveOneFile(ctx context.Context, zw *zip.Writer, idx int, file FileInfo) error {
	if err := ctx.Err(); err != nil {
		return err // honor context cancellation
	}

	hdr, err := zip.FileInfoHeader(file)
	if err != nil {
		return fmt.Errorf("getting info for file %d: %s: %w", idx, file.Name(), err)
	}
	hdr.Name = file.NameInArchive // complete path, since FileInfoHeader() only has base name
	if hdr.Name == "" {
		hdr.Name = file.Name() // assume base name of file I guess
	}

	// customize header based on file properties
	if file.IsDir() {
		if !strings.HasSuffix(hdr.Name, "/") {
			hdr.Name += "/" // required
		}
		hdr.Method = zip.Store
	} else if z.SelectiveCompression {
		// only enable compression on compressable files
		ext := strings.ToLower(path.Ext(hdr.Name))
		if _, ok := compressedFormats[ext]; ok {
			hdr.Method = zip.Store
		} else {
			hdr.Method = z.Compression
		}
	} else {
		hdr.Method = z.Compression
	}

	w, err := zw.CreateHeader(hdr)
	if err != nil {
		return fmt.Errorf("creating header for file %d: %s: %w", idx, file.Name(), err)
	}

	// directories have no file body
	if file.IsDir() {
		return nil
	}
	if err := openAndCopyFile(file, w); err != nil {
		return fmt.Errorf("writing file %d: %s: %w", idx, file.Name(), err)
	}

	return nil
}

// Extract extracts files from z, implementing the Extractor interface. Uniquely, however,
// sourceArchive must be an io.ReaderAt and io.Seeker, which are oddly disjoint interfaces
// from io.Reader which is what the method signature requires. We chose this signature for
// the interface because we figure you can Read() from anything you can ReadAt() or Seek()
// with. Due to the nature of the zip archive format, if sourceArchive is not an io.Seeker
// and io.ReaderAt, an error is returned.
func (z Zip) Extract(ctx context.Context, sourceArchive io.Reader, handleFile FileHandler) error {
	sra, ok := sourceArchive.(seekReaderAt)
	if !ok {
		return fmt.Errorf("input type must be an io.ReaderAt and io.Seeker because of zip format constraints")
	}

	size, err := streamSizeBySeeking(sra)
	if err != nil {
		return fmt.Errorf("determining stream size: %w", err)
	}

	zr, err := zip.NewReader(sra, size)
	if err != nil {
		return err
	}

	// important to initialize to non-nil, empty value due to how fileIsIncluded works
	skipDirs := skipList{}

	for i, f := range zr.File {
		if err := ctx.Err(); err != nil {
			return err // honor context cancellation
		}

		// ensure filename and comment are UTF-8 encoded (issue #147 and PR #305)
		z.decodeText(&f.FileHeader)

		if fileIsIncluded(skipDirs, f.Name) {
			continue
		}

		info := f.FileInfo()
		file := FileInfo{
			FileInfo:      info,
			Header:        f.FileHeader,
			NameInArchive: f.Name,
			Open: func() (fs.File, error) {
				openedFile, err := f.Open()
				if err != nil {
					return nil, err
				}
				return fileInArchive{openedFile, info}, nil
			},
		}

		err := handleFile(ctx, file)
		if errors.Is(err, fs.SkipAll) {
			break
		} else if errors.Is(err, fs.SkipDir) && file.IsDir() {
			skipDirs.add(f.Name)
		} else if err != nil {
			if z.ContinueOnError {
				log.Printf("[ERROR] %s: %v", f.Name, err)
				continue
			}
			return fmt.Errorf("handling file %d: %s: %w", i, f.Name, err)
		}
	}

	return nil
}

// decodeText decodes the name and comment fields from hdr into UTF-8.
// It is a no-op if the text is already UTF-8 encoded or if z.TextEncoding
// is not specified.
func (z Zip) decodeText(hdr *zip.FileHeader) {
	if hdr.NonUTF8 && z.TextEncoding != nil {
		dec := z.TextEncoding.NewDecoder()
		filename, err := dec.String(hdr.Name)
		if err == nil {
			hdr.Name = filename
		}
		if hdr.Comment != "" {
			comment, err := dec.String(hdr.Comment)
			if err == nil {
				hdr.Comment = comment
			}
		}
	}
}

// Insert appends the listed files into the provided Zip archive stream.
// If the filename already exists in the archive, it will be replaced.
func (z Zip) Insert(ctx context.Context, into io.ReadWriteSeeker, files []FileInfo) error {
	// following very simple example at https://github.com/STARRY-S/zip?tab=readme-ov-file#usage
	zu, err := szip.NewUpdater(into)
	if err != nil {
		return err
	}
	defer zu.Close()

	for idx, file := range files {
		if err := ctx.Err(); err != nil {
			return err // honor context cancellation
		}

		hdr, err := szip.FileInfoHeader(file)
		if err != nil {
			return fmt.Errorf("getting info for file %d: %s: %w", idx, file.NameInArchive, err)
		}
		hdr.Name = file.NameInArchive // complete path, since FileInfoHeader() only has base name
		if hdr.Name == "" {
			hdr.Name = file.Name() // assume base name of file I guess
		}

		// customize header based on file properties
		if file.IsDir() {
			if !strings.HasSuffix(hdr.Name, "/") {
				hdr.Name += "/" // required
			}
			hdr.Method = zip.Store
		} else if z.SelectiveCompression {
			// only enable compression on compressable files
			ext := strings.ToLower(path.Ext(hdr.Name))
			if _, ok := compressedFormats[ext]; ok {
				hdr.Method = zip.Store
			} else {
				hdr.Method = z.Compression
			}
		}

		w, err := zu.Append(hdr.Name, szip.APPEND_MODE_OVERWRITE)
		if err != nil {
			return fmt.Errorf("inserting file header: %d: %s: %w", idx, file.Name(), err)
		}

		// directories have no file body
		if file.IsDir() {
			return nil
		}
		if err := openAndCopyFile(file, w); err != nil {
			if z.ContinueOnError && ctx.Err() == nil {
				log.Printf("[ERROR] appending file %d into archive: %s: %v", idx, file.Name(), err)
				continue
			}
			return fmt.Errorf("copying inserted file %d: %s: %w", idx, file.Name(), err)
		}
	}

	return nil
}

type seekReaderAt interface {
	io.ReaderAt
	io.Seeker
}

// Additional compression methods not offered by archive/zip.
// See https://pkware.cachefly.net/webdocs/casestudies/APPNOTE.TXT section 4.4.5.
const (
	ZipMethodBzip2 = 12
	// TODO: LZMA: Disabled - because 7z isn't able to unpack ZIP+LZMA ZIP+LZMA2 archives made this way - and vice versa.
	// ZipMethodLzma     = 14
	ZipMethodZstd = 93
	ZipMethodXz   = 95
)

// compressedFormats is a (non-exhaustive) set of lowercased
// file extensions for formats that are typically already
// compressed. Compressing files that are already compressed
// is inefficient, so use this set of extensions to avoid that.
var compressedFormats = map[string]struct{}{
	".7z":   {},
	".avi":  {},
	".br":   {},
	".bz2":  {},
	".cab":  {},
	".docx": {},
	".gif":  {},
	".gz":   {},
	".jar":  {},
	".jpeg": {},
	".jpg":  {},
	".lz":   {},
	".lz4":  {},
	".lzma": {},
	".m4v":  {},
	".mov":  {},
	".mp3":  {},
	".mp4":  {},
	".mpeg": {},
	".mpg":  {},
	".png":  {},
	".pptx": {},
	".rar":  {},
	".sz":   {},
	".tbz2": {},
	".tgz":  {},
	".tsz":  {},
	".txz":  {},
	".xlsx": {},
	".xz":   {},
	".zip":  {},
	".zipx": {},
}

var zipHeaders = [][]byte{
	[]byte("PK\x03\x04"), // normal
	[]byte("PK\x05\x06"), // empty
}

// Interface guards
var (
	_ Archiver      = Zip{}
	_ ArchiverAsync = Zip{}
	_ Extractor     = Zip{}
)
