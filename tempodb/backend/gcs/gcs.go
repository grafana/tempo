package gcs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/grafana/tempo/pkg/bloom"
	"github.com/grafana/tempo/tempodb/backend/util"

	"cloud.google.com/go/storage"
	"github.com/google/uuid"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
	"google.golang.org/api/iterator"
)

type readerWriter struct {
	cfg    *Config
	client *storage.Client
	bucket *storage.BucketHandle
}

func New(cfg *Config) (backend.Reader, backend.Writer, backend.Compactor, error) {
	ctx := context.Background()

	option, err := instrumentation(ctx, storage.ScopeReadWrite)
	if err != nil {
		return nil, nil, nil, err
	}

	client, err := storage.NewClient(ctx, option)
	if err != nil {
		return nil, nil, nil, err
	}

	bucket := client.Bucket(cfg.BucketName)

	rw := &readerWriter{
		cfg:    cfg,
		client: client,
		bucket: bucket,
	}

	return rw, rw, rw, nil
}

func (rw *readerWriter) Write(ctx context.Context, meta *encoding.BlockMeta, bBloom [][]byte, bIndex []byte, objectFilePath string) error {
	blockID := meta.BlockID
	tenantID := meta.TenantID

	// copy traces file.
	if !fileExists(objectFilePath) {
		return fmt.Errorf("object file not found %s", objectFilePath)
	}

	src, err := os.Open(objectFilePath)
	if err != nil {
		return err
	}
	defer src.Close()

	w := rw.writer(ctx, util.ObjectFileName(blockID, tenantID))
	defer w.Close()
	_, err = io.Copy(w, src)
	if err != nil {
		return err
	}

	err = rw.WriteBlockMeta(ctx, nil, meta, bBloom, bIndex)
	if err != nil {
		return err
	}

	return nil
}

func (rw *readerWriter) WriteBlockMeta(ctx context.Context, tracker backend.AppendTracker, meta *encoding.BlockMeta, bBloom [][]byte, bIndex []byte) error {
	if tracker != nil {
		w := tracker.(*storage.Writer)
		_ = w.Close()
	}

	blockID := meta.BlockID
	tenantID := meta.TenantID

	for i := 0; i < bloom.GetShardNum(); i++ {
		err := rw.writeAll(ctx, util.BloomFileName(blockID, tenantID, uint64(i)), bBloom[i])
		if err != nil {
			return err
		}
	}

	err := rw.writeAll(ctx, util.IndexFileName(blockID, tenantID), bIndex)
	if err != nil {
		return err
	}

	bMeta, err := json.Marshal(meta)
	if err != nil {
		return err
	}

	// write meta last.  this will prevent blocklist from returning a partial block
	err = rw.writeAll(ctx, util.MetaFileName(blockID, tenantID), bMeta)
	if err != nil {
		return err
	}

	return nil
}

func (rw *readerWriter) AppendObject(ctx context.Context, tracker backend.AppendTracker, meta *encoding.BlockMeta, bObject []byte) (backend.AppendTracker, error) {
	var w *storage.Writer
	if tracker == nil {
		blockID := meta.BlockID
		tenantID := meta.TenantID

		w = rw.writer(ctx, util.ObjectFileName(blockID, tenantID))
	} else {
		w = tracker.(*storage.Writer)
	}

	_, err := w.Write(bObject)
	if err != nil {
		return nil, err
	}

	return w, nil
}

func (rw *readerWriter) Tenants() ([]string, error) {
	var warning error
	iter := rw.bucket.Objects(context.Background(), &storage.Query{
		Delimiter: "/",
		Versions:  false,
	})

	tenants := make([]string, 0)

	for {
		attrs, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			warning = err
			continue
		}
		tenants = append(tenants, strings.TrimSuffix(attrs.Prefix, "/"))
	}

	return tenants, warning
}

func (rw *readerWriter) Blocks(tenantID string) ([]uuid.UUID, error) {
	var warning error

	ctx := context.Background()
	iter := rw.bucket.Objects(ctx, &storage.Query{
		Prefix:    tenantID + "/",
		Delimiter: "/",
		Versions:  false,
	})

	blocks := make([]uuid.UUID, 0)
	for {
		attrs, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			warning = err
			continue
		}

		idString := strings.TrimSuffix(strings.TrimPrefix(attrs.Prefix, tenantID+"/"), "/")
		blockID, err := uuid.Parse(idString)
		if err != nil {
			warning = fmt.Errorf("failed parse on blockID %s: %v", idString, err)
			continue
		}

		blocks = append(blocks, blockID)
	}

	return blocks, warning
}

func (rw *readerWriter) BlockMeta(blockID uuid.UUID, tenantID string) (*encoding.BlockMeta, error) {
	name := util.MetaFileName(blockID, tenantID)

	bytes, err := rw.readAll(context.Background(), name)
	if err == storage.ErrObjectNotExist {
		return nil, backend.ErrMetaDoesNotExist
	}
	if err != nil {
		return nil, err
	}

	out := &encoding.BlockMeta{}
	err = json.Unmarshal(bytes, out)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (rw *readerWriter) Bloom(blockID uuid.UUID, tenantID string, shardNum uint64) ([]byte, error) {
	name := util.BloomFileName(blockID, tenantID, shardNum)
	return rw.readAll(context.Background(), name)
}

func (rw *readerWriter) Index(blockID uuid.UUID, tenantID string) ([]byte, error) {
	name := util.IndexFileName(blockID, tenantID)
	return rw.readAll(context.Background(), name)
}

func (rw *readerWriter) Object(blockID uuid.UUID, tenantID string, start uint64, buffer []byte) error {
	name := util.ObjectFileName(blockID, tenantID)
	return rw.readRange(context.Background(), name, int64(start), buffer)
}

func (rw *readerWriter) Shutdown() {

}

func (rw *readerWriter) writeAll(ctx context.Context, name string, b []byte) error {
	w := rw.writer(ctx, name)
	defer w.Close()

	_, err := w.Write(b)
	if err != nil {
		return err
	}

	return nil
}

func (rw *readerWriter) writer(ctx context.Context, name string) *storage.Writer {
	w := rw.bucket.Object(name).NewWriter(ctx)
	w.ChunkSize = rw.cfg.ChunkBufferSize

	return w
}

func (rw *readerWriter) readAll(ctx context.Context, name string) ([]byte, error) {
	r, err := rw.bucket.Object(name).NewReader(ctx)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	return ioutil.ReadAll(r)
}

func (rw *readerWriter) readAllWithModTime(ctx context.Context, name string) ([]byte, time.Time, error) {
	r, err := rw.bucket.Object(name).NewReader(ctx)
	if err != nil {
		return nil, time.Time{}, err
	}
	defer r.Close()

	bytes, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, time.Time{}, err
	}

	return bytes, r.Attrs.LastModified, nil
}

func (rw *readerWriter) readRange(ctx context.Context, name string, offset int64, buffer []byte) error {
	r, err := rw.bucket.Object(name).NewRangeReader(ctx, offset, int64(len(buffer)))
	if err != nil {
		return err
	}
	defer r.Close()

	totalBytes := 0
	for {
		byteCount, err := r.Read(buffer[totalBytes:])
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if byteCount == 0 {
			return nil
		}
		totalBytes += byteCount
	}
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}
