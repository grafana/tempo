package archives

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

// FileInfo is a virtualized, generalized file abstraction for interacting with archives.
type FileInfo struct {
	fs.FileInfo

	// The file header as used/provided by the archive format.
	// Typically, you do not need to set this field when creating
	// an archive.
	Header any

	// The path of the file as it appears in the archive.
	// This is equivalent to Header.Name (for most Header
	// types). We require it to be specified here because
	// it is such a common field and we want to preserve
	// format-agnosticism (no type assertions) for basic
	// operations.
	//
	// When extracting, this name or path may not have
	// been sanitized; it should not be trusted at face
	// value. Consider using path.Clean() before using.
	//
	// If this is blank when inserting a file into an
	// archive, the filename's base may be assumed
	// by default to be the name in the archive.
	NameInArchive string

	// For symbolic and hard links, the target of the link.
	// Not supported by all archive formats.
	LinkTarget string

	// A callback function that opens the file to read its
	// contents. The file must be closed when reading is
	// complete.
	Open func() (fs.File, error)
}

func (f FileInfo) Stat() (fs.FileInfo, error) { return f.FileInfo, nil }

// FilesFromDisk is an opinionated function that returns a list of FileInfos
// by walking the directories in the filenames map. The keys are the names on
// disk, and the values become their associated names in the archive.
//
// Map keys that specify directories on disk will be walked and added to the
// archive recursively, rooted at the named directory. They should use the
// platform's path separator (backslash on Windows; slash on everything else).
// For convenience, map keys that end in a separator ('/', or '\' on Windows)
// will enumerate contents only, without adding the folder itself to the archive.
//
// Map values should typically use slash ('/') as the separator regardless of
// the platform, as most archive formats standardize on that rune as the
// directory separator for filenames within an archive. For convenience, map
// values that are empty string are interpreted as the base name of the file
// (sans path) in the root of the archive; and map values that end in a slash
// will use the base name of the file in that folder of the archive.
//
// File gathering will adhere to the settings specified in options.
//
// This function is used primarily when preparing a list of files to add to
// an archive.
func FilesFromDisk(ctx context.Context, options *FromDiskOptions, filenames map[string]string) ([]FileInfo, error) {
	var files []FileInfo
	for rootOnDisk, rootInArchive := range filenames {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		walkErr := filepath.WalkDir(rootOnDisk, func(filename string, d fs.DirEntry, err error) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			if err != nil {
				return err
			}

			info, err := d.Info()
			if err != nil {
				return err
			}

			nameInArchive := nameOnDiskToNameInArchive(filename, rootOnDisk, rootInArchive)
			// this is the root folder and we are adding its contents to target rootInArchive
			if info.IsDir() && nameInArchive == "" {
				return nil
			}

			// handle symbolic links
			var linkTarget string
			if isSymlink(info) {
				if options != nil && options.FollowSymlinks {
					// dereference symlinks
					filename, err = os.Readlink(filename)
					if err != nil {
						return fmt.Errorf("%s: readlink: %w", filename, err)
					}
					info, err = os.Stat(filename)
					if err != nil {
						return fmt.Errorf("%s: statting dereferenced symlink: %w", filename, err)
					}
				} else {
					// preserve symlinks
					linkTarget, err = os.Readlink(filename)
					if err != nil {
						return fmt.Errorf("%s: readlink: %w", filename, err)
					}
				}
			}

			// handle file attributes
			if options != nil && options.ClearAttributes {
				info = noAttrFileInfo{info}
			}

			file := FileInfo{
				FileInfo:      info,
				NameInArchive: nameInArchive,
				LinkTarget:    linkTarget,
				Open: func() (fs.File, error) {
					return os.Open(filename)
				},
			}

			files = append(files, file)
			return nil
		})
		if walkErr != nil {
			return nil, walkErr
		}
	}
	return files, nil
}

// nameOnDiskToNameInArchive converts a filename from disk to a name in an archive,
// respecting rules defined by FilesFromDisk. nameOnDisk is the full filename on disk
// which is expected to be prefixed by rootOnDisk (according to fs.WalkDirFunc godoc)
// and which will be placed into a folder rootInArchive in the archive.
func nameOnDiskToNameInArchive(nameOnDisk, rootOnDisk, rootInArchive string) string {
	// These manipulations of rootInArchive could be done just once instead of on
	// every walked file since they don't rely on nameOnDisk which is the only
	// variable that changes during the walk, but combining all the logic into this
	// one function is easier to reason about and test. I suspect the performance
	// penalty is insignificant.
	if strings.HasSuffix(rootOnDisk, string(filepath.Separator)) {
		// "map keys that end in a separator will enumerate contents only,
		// without adding the folder itself to the archive."
		rootInArchive = trimTopDir(rootInArchive)
	} else if rootInArchive == "" {
		// "map values that are empty string are interpreted as the base name
		// of the file (sans path) in the root of the archive"
		rootInArchive = filepath.Base(rootOnDisk)
	}
	if rootInArchive == "." {
		// an in-archive root of "." is an escape hatch for the above rule
		// where an empty in-archive root means to use the base name of the
		// file; if the user does not want this, they can specify a "." to
		// still put it in the root of the archive
		rootInArchive = ""
	}
	if strings.HasSuffix(rootInArchive, "/") {
		// "map values that end in a slash will use the base name of the file in
		// that folder of the archive."
		rootInArchive += filepath.Base(rootOnDisk)
	}
	truncPath := strings.TrimPrefix(nameOnDisk, rootOnDisk)
	return path.Join(rootInArchive, filepath.ToSlash(truncPath))
}

// trimTopDir strips the top or first directory from the path.
// It expects a forward-slashed path.
//
// Examples: "a/b/c" => "b/c", "/a/b/c" => "b/c"
func trimTopDir(dir string) string {
	return strings.TrimPrefix(dir, topDir(dir)+"/")
}

// topDir returns the top or first directory in the path.
// It expects a forward-slashed path.
//
// Examples: "a/b/c" => "a", "/a/b/c" => "/a"
func topDir(dir string) string {
	var start int
	if len(dir) > 0 && dir[0] == '/' {
		start = 1
	}
	if pos := strings.Index(dir[start:], "/"); pos >= 0 {
		return dir[:pos+start]
	}
	return dir
}

// noAttrFileInfo is used to zero out some file attributes (issue #280).
type noAttrFileInfo struct{ fs.FileInfo }

// Mode preserves only the type and permission bits.
func (no noAttrFileInfo) Mode() fs.FileMode {
	return no.FileInfo.Mode() & (fs.ModeType | fs.ModePerm)
}
func (noAttrFileInfo) ModTime() time.Time { return time.Time{} }
func (noAttrFileInfo) Sys() any           { return nil }

// FromDiskOptions specifies various options for gathering files from disk.
type FromDiskOptions struct {
	// If true, symbolic links will be dereferenced, meaning that
	// the link will not be added as a link, but what the link
	// points to will be added as a file.
	FollowSymlinks bool

	// If true, some file attributes will not be preserved.
	// Name, size, type, and permissions will still be preserved.
	ClearAttributes bool
}

// FileHandler is a callback function that is used to handle files as they are read
// from an archive; it is kind of like fs.WalkDirFunc. Handler functions that open
// their files must not overlap or run concurrently, as files may be read from the
// same sequential stream; always close the file before returning.
//
// If the special error value fs.SkipDir is returned, the directory of the file
// (or the file itself if it is a directory) will not be walked. Note that because
// archive contents are not necessarily ordered, skipping directories requires
// memory, and skipping lots of directories may run up your memory bill.
//
// Any other returned error will terminate a walk and be returned to the caller.
type FileHandler func(ctx context.Context, info FileInfo) error

// openAndCopyFile opens file for reading, copies its
// contents to w, then closes file.
func openAndCopyFile(file FileInfo, w io.Writer) error {
	fileReader, err := file.Open()
	if err != nil {
		return err
	}
	defer fileReader.Close()
	// When file is in use and size is being written to, creating the compressed
	// file will fail with "archive/tar: write too long." Using CopyN gracefully
	// handles this.
	_, err = io.Copy(w, fileReader)
	if err != nil && err != io.EOF {
		return err
	}
	return nil
}

// fileIsIncluded returns true if filename is included according to
// filenameList; meaning it is in the list, its parent folder/path
// is in the list, or the list is nil.
func fileIsIncluded(filenameList []string, filename string) bool {
	// include all files if there is no specific list
	if filenameList == nil {
		return true
	}
	for _, fn := range filenameList {
		// exact matches are of course included
		if filename == fn {
			return true
		}
		// also consider the file included if its parent folder/path is in the list
		if strings.HasPrefix(filename, strings.TrimSuffix(fn, "/")+"/") {
			return true
		}
	}
	return false
}

func isSymlink(info fs.FileInfo) bool {
	return info.Mode()&os.ModeSymlink != 0
}

// streamSizeBySeeking determines the size of the stream by
// seeking to the end, then back again, so the resulting
// seek position upon returning is the same as when called
// (assuming no errors).
func streamSizeBySeeking(s io.Seeker) (int64, error) {
	currentPosition, err := s.Seek(0, io.SeekCurrent)
	if err != nil {
		return 0, fmt.Errorf("getting current offset: %w", err)
	}
	maxPosition, err := s.Seek(0, io.SeekEnd)
	if err != nil {
		return 0, fmt.Errorf("fast-forwarding to end: %w", err)
	}
	_, err = s.Seek(currentPosition, io.SeekStart)
	if err != nil {
		return 0, fmt.Errorf("returning to prior offset %d: %w", currentPosition, err)
	}
	return maxPosition, nil
}

// skipList keeps a list of non-intersecting paths
// as long as its add method is used. Identical
// elements are rejected, more specific paths are
// replaced with broader ones, and more specific
// paths won't be added when a broader one already
// exists in the list. Trailing slashes are ignored.
type skipList []string

func (s *skipList) add(dir string) {
	trimmedDir := strings.TrimSuffix(dir, "/")
	var dontAdd bool
	for i := 0; i < len(*s); i++ {
		trimmedElem := strings.TrimSuffix((*s)[i], "/")
		if trimmedDir == trimmedElem {
			return
		}
		// don't add dir if a broader path already exists in the list
		if strings.HasPrefix(trimmedDir, trimmedElem+"/") {
			dontAdd = true
			continue
		}
		// if dir is broader than a path in the list, remove more specific path in list
		if strings.HasPrefix(trimmedElem, trimmedDir+"/") {
			*s = append((*s)[:i], (*s)[i+1:]...)
			i--
		}
	}
	if !dontAdd {
		*s = append(*s, dir)
	}
}
