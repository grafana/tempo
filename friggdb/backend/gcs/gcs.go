package gcs

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/google/uuid"
	"github.com/grafana/frigg/friggdb/backend"
	"google.golang.org/api/iterator"
)

type readerWriter struct {
	cfg    Config
	client *storage.Client
	bucket *storage.BucketHandle
}

func New(cfg Config) (backend.Reader, backend.Writer, error) {
	ctx := context.Background()

	option, err := instrumentation(ctx, storage.ScopeReadWrite)
	if err != nil {
		return nil, nil, err
	}

	client, err := storage.NewClient(ctx, option)
	if err != nil {
		return nil, nil, err
	}

	bucket := client.Bucket(cfg.BucketName)

	rw := &readerWriter{
		cfg:    cfg,
		client: client,
		bucket: bucket,
	}

	return rw, rw, nil
}

func (rw *readerWriter) Write(ctx context.Context, blockID uuid.UUID, tenantID string, bMeta []byte, bBloom []byte, bIndex []byte, objectFilePath string) error {

	err := rw.writeAll(ctx, rw.bloomFileName(blockID, tenantID), bBloom)
	if err != nil {
		return err
	}

	err = rw.writeAll(ctx, rw.indexFileName(blockID, tenantID), bIndex)
	if err != nil {
		return err
	}

	// copy traces file.
	if !fileExists(objectFilePath) {
		return fmt.Errorf("object file not found %s", objectFilePath)
	}

	src, err := os.Open(objectFilePath)
	if err != nil {
		return err
	}
	defer src.Close()

	w := rw.writer(ctx, rw.objectFileName(blockID, tenantID))
	defer w.Close()
	_, err = io.Copy(w, src)
	if err != nil {
		return err
	}

	// write meta last.  this will prevent blocklist from returning a partial block
	err = rw.writeAll(ctx, rw.metaFileName(blockID, tenantID), bMeta)
	if err != nil {
		return err
	}

	return nil
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

// erg...this is gross...need to rethink this in cloud storage.
//   jpe: implement regular, smaller updates? jam all block metas in one file?  panic?
func (rw *readerWriter) Blocklist(tenantID string) ([][]byte, error) {
	var warning error

	ctx := context.Background()
	iter := rw.bucket.Objects(ctx, &storage.Query{
		Prefix:    tenantID + "/",
		Delimiter: "/",
		Versions:  false,
	})

	blocklists := make([][]byte, 0)
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

		filename := rw.metaFileName(blockID, tenantID)
		b, err := rw.readAll(ctx, filename)
		if err != nil {
			warning = fmt.Errorf("failed read on blockID %s: %v", blockID, err)
			continue
		}
		blocklists = append(blocklists, b)
	}

	return blocklists, warning
}

func (rw *readerWriter) Bloom(blockID uuid.UUID, tenantID string) ([]byte, error) {
	name := rw.bloomFileName(blockID, tenantID)
	return rw.readAll(context.Background(), name)
}

func (rw *readerWriter) Index(blockID uuid.UUID, tenantID string) ([]byte, error) {
	name := rw.indexFileName(blockID, tenantID)
	return rw.readAll(context.Background(), name)
}

func (rw *readerWriter) Object(blockID uuid.UUID, tenantID string, start uint64, length uint32) ([]byte, error) {
	name := rw.indexFileName(blockID, tenantID)
	return rw.readRange(context.Background(), name, int64(start), int64(length))
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

func (rw *readerWriter) objectFileName(blockID uuid.UUID, tenantID string) string {
	return path.Join(rw.rootPath(blockID, tenantID), "data")
}

func (rw *readerWriter) rootPath(blockID uuid.UUID, tenantID string) string {
	return path.Join(tenantID, blockID.String())
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

func (rw *readerWriter) readRange(ctx context.Context, name string, offset int64, length int64) ([]byte, error) {
	r, err := rw.bucket.Object(name).NewRangeReader(ctx, offset, length)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	return ioutil.ReadAll(r)
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}
