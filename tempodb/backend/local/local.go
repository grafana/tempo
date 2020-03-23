package local

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"

	"github.com/google/uuid"
	"github.com/grafana/tempo/tempodb/backend"
)

type readerWriter struct {
	cfg *Config
}

func New(cfg *Config) (backend.Reader, backend.Writer, backend.Compactor, error) {
	err := os.MkdirAll(cfg.Path, os.ModePerm)
	if err != nil {
		return nil, nil, nil, err
	}

	rw := &readerWriter{
		cfg: cfg,
	}

	return rw, rw, rw, nil
}

func (rw *readerWriter) Write(ctx context.Context, meta *backend.BlockMeta, bBloom []byte, bIndex []byte, tracesFilePath string) error {
	err := rw.WriteBlockMeta(ctx, nil, meta, bBloom, bIndex)
	if err != nil {
		return err
	}

	blockID := meta.BlockID
	tenantID := meta.TenantID
	blockFolder := rw.rootPath(blockID, tenantID)

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

	tracesFileName := rw.tracesFileName(blockID, tenantID)
	dst, err := os.Create(tracesFileName)
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

func (rw *readerWriter) WriteBlockMeta(_ context.Context, tracker backend.AppendTracker, meta *backend.BlockMeta, bBloom []byte, bIndex []byte) error {
	blockID := meta.BlockID
	tenantID := meta.TenantID

	if tracker != nil {
		var dst *os.File = tracker.(*os.File)
		_ = dst.Close()
	}

	blockFolder := rw.rootPath(blockID, tenantID)
	err := os.MkdirAll(blockFolder, os.ModePerm)
	if err != nil {
		return err
	}

	bMeta, err := json.Marshal(meta)
	if err != nil {
		return err
	}

	metaFileName := rw.metaFileName(blockID, tenantID)
	err = ioutil.WriteFile(metaFileName, bMeta, 0644)
	if err != nil {
		os.RemoveAll(blockFolder)
		return err
	}

	bloomFileName := rw.bloomFileName(blockID, tenantID)
	err = ioutil.WriteFile(bloomFileName, bBloom, 0644)
	if err != nil {
		os.RemoveAll(blockFolder)
		return err
	}

	indexFileName := rw.indexFileName(blockID, tenantID)
	err = ioutil.WriteFile(indexFileName, bIndex, 0644)
	if err != nil {
		os.RemoveAll(blockFolder)
		return err
	}

	return nil
}

func (rw *readerWriter) AppendObject(ctx context.Context, tracker backend.AppendTracker, meta *backend.BlockMeta, bObject []byte) (backend.AppendTracker, error) {
	blockID := meta.BlockID
	tenantID := meta.TenantID

	var dst *os.File
	if tracker == nil {
		blockFolder := rw.rootPath(blockID, tenantID)
		err := os.MkdirAll(blockFolder, os.ModePerm)
		if err != nil {
			return nil, err
		}

		tracesFileName := rw.tracesFileName(blockID, tenantID)
		dst, err = os.Create(tracesFileName)
		if err != nil {
			return nil, err
		}
	} else {
		dst = tracker.(*os.File)
	}

	_, err := dst.Write(bObject)
	if err != nil {
		return nil, err
	}

	return dst, nil
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

func (rw *readerWriter) Blocks(tenantID string) ([]uuid.UUID, error) {
	var warning error
	path := path.Join(rw.cfg.Path, tenantID)
	folders, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}

	blocks := make([]uuid.UUID, 0, len(folders))
	for _, f := range folders {
		if !f.IsDir() {
			continue
		}
		blockID, err := uuid.Parse(f.Name())
		if err != nil {
			warning = err
			continue
		}
		blocks = append(blocks, blockID)
	}

	return blocks, warning
}

func (rw *readerWriter) BlockMeta(blockID uuid.UUID, tenantID string) (*backend.BlockMeta, error) {
	filename := rw.metaFileName(blockID, tenantID)
	bytes, err := ioutil.ReadFile(filename)
	if os.IsNotExist(err) {
		return nil, backend.ErrMetaDoesNotExist
	}
	if err != nil {
		return nil, err
	}

	out := &backend.BlockMeta{}
	err = json.Unmarshal(bytes, out)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (rw *readerWriter) Bloom(blockID uuid.UUID, tenantID string) ([]byte, error) {
	filename := rw.bloomFileName(blockID, tenantID)
	return ioutil.ReadFile(filename)
}

func (rw *readerWriter) Index(blockID uuid.UUID, tenantID string) ([]byte, error) {
	filename := rw.indexFileName(blockID, tenantID)
	return ioutil.ReadFile(filename)
}

func (rw *readerWriter) Object(blockID uuid.UUID, tenantID string, start uint64, buffer []byte) error {
	filename := rw.tracesFileName(blockID, tenantID)

	f, err := os.OpenFile(filename, os.O_RDONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.ReadAt(buffer, int64(start))
	if err != nil {
		return err
	}

	return nil
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
