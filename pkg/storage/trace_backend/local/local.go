package local

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"

	"github.com/google/uuid"
	"github.com/grafana/frigg/pkg/storage/trace_backend"
)

type readerWriter struct {
	cfg Config
}

func New(cfg Config) (trace_backend.Reader, trace_backend.Writer, error) {
	err := os.MkdirAll(cfg.Path, os.ModePerm)
	if err != nil {
		return nil, nil, err
	}

	rw := &readerWriter{
		cfg: cfg,
	}

	return rw, rw, nil
}

func (rw *readerWriter) Write(_ context.Context, blockID uuid.UUID, tenantID string, bBloom []byte, bIndex []byte, tracesFilePath string) error {
	err := os.MkdirAll(rw.rootPath(blockID, tenantID), os.ModePerm)
	if err != nil {
		return err
	}

	bloom := rw.bloomFileName(blockID, tenantID)
	if fileExists(bloom) {
		return fmt.Errorf("unexpectedly found bloom file at %s", bloom)
	}
	err = ioutil.WriteFile(bloom, bBloom, 0644)
	if err != nil {
		return err
	}

	index := rw.indexFileName(blockID, tenantID)
	if fileExists(index) {
		return fmt.Errorf("unexpectedly found index file at %s", index)
	}
	err = ioutil.WriteFile(index, bIndex, 0644)
	if err != nil {
		return err
	}

	traces := rw.tracesFileName(blockID, tenantID)
	if fileExists(traces) {
		return fmt.Errorf("unexpectedly found traces file at %s", index)
	}
	if !fileExists(tracesFilePath) {
		return fmt.Errorf("traces file not found %s", tracesFilePath)
	}

	// copy traces file.
	//  todo:  consider having the storage backend responsible for removing the block.  in this case we could just
	//   do a rename here which would be way faster.
	src, err := os.Open(tracesFilePath)
	if err != nil {
		return err
	}
	defer src.Close()
	dst, err := os.Create(traces)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	return err
}

func (rw *readerWriter) Bloom(tenantID string, fn trace_backend.BloomIter) error {
	path := path.Join(rw.cfg.Path, tenantID)
	folders, err := ioutil.ReadDir(path)
	if err != nil {
		return err
	}

	for _, f := range folders {
		if !f.IsDir() {
			continue
		}

		blockID, err := uuid.Parse(f.Name())
		if err != nil {
			return err
		}

		filename := rw.bloomFileName(blockID, tenantID)
		bytes, err := ioutil.ReadFile(filename)

		more, err := fn(bytes, blockID)

		if err != nil {
			return err
		}

		if !more {
			break
		}
	}

	return nil
}

func (rw *readerWriter) Index(blockID uuid.UUID, tenantID string) ([]byte, error) {
	filename := rw.indexFileName(blockID, tenantID)
	return ioutil.ReadFile(filename)
}

func (rw *readerWriter) Trace(blockID uuid.UUID, tenantID string, start uint64, length uint32) ([]byte, error) {
	filename := rw.tracesFileName(blockID, tenantID)

	f, err := os.OpenFile(filename, os.O_RDONLY, 0644)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	b := make([]byte, length)
	_, err = f.ReadAt(b, int64(start))
	if err != nil {
		return nil, err
	}

	return b, nil
}

func (rw *readerWriter) bloomFileName(blockID uuid.UUID, tenantID string) string {
	return path.Join(rw.rootPath(blockID, tenantID), "bloom")
}

func (rw *readerWriter) indexFileName(blockID uuid.UUID, tenantID string) string {
	return path.Join(rw.rootPath(blockID, tenantID), "index")
}

func (rw *readerWriter) tracesFileName(blockID uuid.UUID, tenantID string) string {
	return path.Join(rw.rootPath(blockID, tenantID), "traces")
}

func (rw *readerWriter) rootPath(blockID uuid.UUID, tenantID string) string {
	return path.Join(rw.cfg.Path, tenantID, blockID.String())
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return true
}
