package local

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"

	"github.com/google/uuid"
	"github.com/grafana/frigg/friggdb/backend"
)

type readerWriter struct {
	cfg *Config
}

func New(cfg *Config) (backend.Reader, backend.Writer, error) {
	err := os.MkdirAll(cfg.Path, os.ModePerm)
	if err != nil {
		return nil, nil, err
	}

	rw := &readerWriter{
		cfg: cfg,
	}

	return rw, rw, nil
}

func (rw *readerWriter) Write(_ context.Context, blockID uuid.UUID, tenantID string, bMeta []byte, bBloom []byte, bIndex []byte, tracesFilePath string) error {
	blockFolder := rw.rootPath(blockID, tenantID)
	err := os.MkdirAll(blockFolder, os.ModePerm)
	if err != nil {
		return err
	}

	meta := rw.metaFileName(blockID, tenantID)
	err = ioutil.WriteFile(meta, bMeta, 0644)
	if err != nil {
		os.RemoveAll(blockFolder)
		return err
	}

	bloom := rw.bloomFileName(blockID, tenantID)
	err = ioutil.WriteFile(bloom, bBloom, 0644)
	if err != nil {
		os.RemoveAll(blockFolder)
		return err
	}

	index := rw.indexFileName(blockID, tenantID)
	err = ioutil.WriteFile(index, bIndex, 0644)
	if err != nil {
		os.RemoveAll(blockFolder)
		return err
	}

	traces := rw.tracesFileName(blockID, tenantID)
	if !fileExists(tracesFilePath) {
		os.RemoveAll(blockFolder)
		return fmt.Errorf("traces file not found %s", tracesFilePath)
	}

	// copy traces file.
	src, err := os.Open(tracesFilePath)
	if err != nil {
		os.RemoveAll(blockFolder)
		return err
	}
	defer src.Close()
	dst, err := os.Create(traces)
	if err != nil {
		os.RemoveAll(blockFolder)
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	if err != nil {
		os.RemoveAll(blockFolder)
	}
	return err
}

func (rw *readerWriter) Tenants() ([]string, error) {
	folders, err := ioutil.ReadDir(rw.cfg.Path)
	if err != nil {
		return nil, err
	}

	tenants := make([]string, 0, len(folders))
	for _, f := range folders {
		if !f.IsDir() {
			continue
		}
		tenants = append(tenants, f.Name())
	}

	return tenants, nil
}

func (rw *readerWriter) Blocklist(tenantID string) ([][]byte, error) {
	var warning error
	path := path.Join(rw.cfg.Path, tenantID)
	folders, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}

	blocklists := make([][]byte, 0, len(folders))
	for _, f := range folders {
		if !f.IsDir() {
			continue
		}
		blockID, err := uuid.Parse(f.Name())
		if err != nil {
			warning = err
			continue
		}
		filename := rw.metaFileName(blockID, tenantID)
		bytes, err := ioutil.ReadFile(filename)
		if err != nil {
			warning = err
			continue
		}

		blocklists = append(blocklists, bytes)
	}

	return blocklists, warning
}

func (rw *readerWriter) Bloom(blockID uuid.UUID, tenantID string) ([]byte, error) {
	filename := rw.bloomFileName(blockID, tenantID)
	return ioutil.ReadFile(filename)
}

func (rw *readerWriter) Index(blockID uuid.UUID, tenantID string) ([]byte, error) {
	filename := rw.indexFileName(blockID, tenantID)
	return ioutil.ReadFile(filename)
}

func (rw *readerWriter) Object(blockID uuid.UUID, tenantID string, start uint64, length uint32) ([]byte, error) {
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

func (rw *readerWriter) Shutdown() {

}

func (rw *readerWriter) metaFileName(blockID uuid.UUID, tenantID string) string {
	return path.Join(rw.rootPath(blockID, tenantID), "meta.json")
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
	return !os.IsNotExist(err)
}
