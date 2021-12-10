// Package file implements a koanf.Provider that reads raw bytes
// from files on disk to be used with a koanf.Parser to parse
// into conf maps.
package file

import (
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

// File implements a File provider.
type File struct {
	path string
}

// Provider returns a file provider.
func Provider(path string) *File {
	return &File{path: filepath.Clean(path)}
}

// ReadBytes reads the contents of a file on disk and returns the bytes.
func (f *File) ReadBytes() ([]byte, error) {
	return ioutil.ReadFile(f.path)
}

// Read is not supported by the file provider.
func (f *File) Read() (map[string]interface{}, error) {
	return nil, errors.New("file provider does not support this method")
}

// Watch watches the file and triggers a callback when it changes. It is a
// blocking function that internally spawns a goroutine to watch for changes.
func (f *File) Watch(cb func(event interface{}, err error)) error {
	// Resolve symlinks and save the original path so that changes to symlinks
	// can be detected.
	realPath, err := filepath.EvalSymlinks(f.path)
	if err != nil {
		return err
	}
	realPath = filepath.Clean(realPath)

	// Although only a single file is being watched, fsnotify has to watch
	// the whole parent directory to pick up all events such as symlink changes.
	fDir, _ := filepath.Split(f.path)

	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	var (
		lastEvent     string
		lastEventTime time.Time
	)

	go func() {
	loop:
		for {
			select {
			case event, ok := <-w.Events:
				if !ok {
					cb(nil, errors.New("fsnotify watch channel closed"))
					break loop
				}

				// Use a simple timer to buffer events as certain events fire
				// multiple times on some platforms.
				if event.String() == lastEvent && time.Since(lastEventTime) < time.Millisecond*5 {
					continue
				}
				lastEvent = event.String()
				lastEventTime = time.Now()

				evFile := filepath.Clean(event.Name)

				// Since the event is triggered on a directory, is this
				// one on the file being watched?
				if evFile != realPath && evFile != f.path {
					continue
				}

				// The file was removed.
				if event.Op&fsnotify.Remove != 0 {
					cb(nil, fmt.Errorf("file %s was removed", event.Name))
					break loop
				}

				// Resolve symlink to get the real path, in case the symlink's
				// target has changed.
				curPath, err := filepath.EvalSymlinks(f.path)
				if err != nil {
					cb(nil, err)
					break loop
				}
				realPath = filepath.Clean(curPath)

				// Finally, we only care about create and write.
				if event.Op&(fsnotify.Write|fsnotify.Create) == 0 {
					continue
				}

				// Trigger event.
				cb(nil, nil)

			// There's an error.
			case err, ok := <-w.Errors:
				if !ok {
					cb(nil, errors.New("fsnotify err channel closed"))
					break loop
				}

				// Pass the error to the callback.
				cb(nil, err)
				break loop
			}
		}

		w.Close()
	}()

	// Watch the directory for changes.
	return w.Add(fDir)
}
