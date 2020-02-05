package local

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"

	"github.com/google/uuid"
	"github.com/grafana/frigg/friggdb/backend"
)

type readerWriter struct {
	cfg Config
}

func New(cfg Config) (backend.Reader, backend.Writer, error) {
	err := os.MkdirAll(cfg.Path, os.ModePerm)
	if err != nil {
		return nil, nil, err
	}

	rw := &readerWriter{
		cfg: cfg,
	}

	return rw, rw, nil
}

func (rw *readerWriter) Write(blockID uuid.UUID, tenantID string, bMeta []byte, bBloom []byte, bIndex []byte, tracesFilePath string) error {
	err := os.MkdirAll(rw.rootPath(blockID, tenantID), os.ModePerm)
	if err != nil {
		return err
	}

	meta := rw.metaFileName(blockID, tenantID)
	if fileExists(meta) {
		return fmt.Errorf("unexpectedly found meta file at %s", meta)
	}
	err = ioutil.WriteFile(meta, bMeta, 0644)
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
			return nil, err
		}
		filename := rw.metaFileName(blockID, tenantID)
		bytes, err := ioutil.ReadFile(filename)
		if err != nil {
			return nil, err
		}

		blocklists = append(blocklists, bytes)
	}

	return blocklists, nil
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
