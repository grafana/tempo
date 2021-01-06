package gcs

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/grafana/tempo/tempodb/backend/util"
	"github.com/pkg/errors"

	"cloud.google.com/go/storage"
	"github.com/google/uuid"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/opentracing/opentracing-go"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	google_http "google.golang.org/api/transport/http"
)

type readerWriter struct {
	cfg    *Config
	client *storage.Client
	bucket *storage.BucketHandle
}

func New(cfg *Config) (backend.Reader, backend.Writer, backend.Compactor, error) {
	ctx := context.Background()

	customTransport := http.DefaultTransport.(*http.Transport).Clone()

	transportOptions := []option.ClientOption{}
	if cfg.Insecure {
		transportOptions = append(transportOptions, option.WithoutAuthentication())
		customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	transport, err := google_http.NewTransport(ctx, customTransport, transportOptions...)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "creating transport")
	}
	transport = newInstrumentedTransport(transport)

	storageClientOptions := []option.ClientOption{
		option.WithHTTPClient(&http.Client{
			Transport: transport,
		}),
		option.WithScopes(storage.ScopeReadWrite),
	}
	if cfg.Endpoint != "" {
		storageClientOptions = append(storageClientOptions, option.WithEndpoint(cfg.Endpoint))
	}

	client, err := storage.NewClient(ctx, storageClientOptions...)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "creating storage client")
	}

	bucket := client.Bucket(cfg.BucketName)

	// Check bucket exists by getting attrs
	if _, err = bucket.Attrs(ctx); err != nil {
		return nil, nil, nil, errors.Wrap(err, "getting bucket attrs")
	}

	rw := &readerWriter{
		cfg:    cfg,
		client: client,
		bucket: bucket,
	}

	return rw, rw, rw, nil
}

// Write implements backend.Writer
func (rw *readerWriter) Write(ctx context.Context, name string, blockID uuid.UUID, tenantID string, data io.Reader, _ int64) error {
	w := rw.writer(ctx, util.ObjectFileName(blockID, tenantID, name))
	_, err := io.Copy(w, data)
	if err != nil {
		w.Close()
		return err
	}

	return w.Close()
}

// WriteBlockMeta implements backend.Writer
func (rw *readerWriter) WriteBlockMeta(ctx context.Context, meta *backend.BlockMeta) error {
	blockID := meta.BlockID
	tenantID := meta.TenantID

	bMeta, err := json.Marshal(meta)
	if err != nil {
		return err
	}

	err = rw.writeAll(ctx, util.MetaFileName(blockID, tenantID), bMeta)
	if err != nil {
		return err
	}

	return nil
}

// Append implements backend.Writer
func (rw *readerWriter) Append(ctx context.Context, name string, blockID uuid.UUID, tenantID string, tracker backend.AppendTracker, buffer []byte) (backend.AppendTracker, error) {
	var w *storage.Writer
	if tracker == nil {
		w = rw.writer(ctx, util.ObjectFileName(blockID, tenantID, name))
	} else {
		w = tracker.(*storage.Writer)
	}

	_, err := w.Write(buffer)
	if err != nil {
		return nil, err
	}

	return w, nil
}

// CloseAppend implements backend.Writer
func (rw *readerWriter) CloseAppend(_ context.Context, tracker backend.AppendTracker) error {
	w := tracker.(*storage.Writer)
	return w.Close()
}

func (rw *readerWriter) Tenants(ctx context.Context) ([]string, error) {
	iter := rw.bucket.Objects(ctx, &storage.Query{
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
			return tenants, errors.Wrap(err, "iterating tenants")
		}

		tenants = append(tenants, strings.TrimSuffix(attrs.Prefix, "/"))
	}

	return tenants, nil
}

func (rw *readerWriter) Blocks(ctx context.Context, tenantID string) ([]uuid.UUID, error) {
	var warning error

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
			return blocks, errors.Wrap(err, "iterating blocks")
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

func (rw *readerWriter) BlockMeta(ctx context.Context, blockID uuid.UUID, tenantID string) (*backend.BlockMeta, error) {
	name := util.MetaFileName(blockID, tenantID)

	bytes, err := rw.readAll(ctx, name)
	if err == storage.ErrObjectNotExist {
		return nil, backend.ErrMetaDoesNotExist
	}
	if err != nil {
		return nil, errors.Wrap(err, "read block meta from gcs")
	}

	out := &backend.BlockMeta{}
	err = json.Unmarshal(bytes, out)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (rw *readerWriter) Bloom(ctx context.Context, blockID uuid.UUID, tenantID string, shardNum int) ([]byte, error) {
	span, derivedCtx := opentracing.StartSpanFromContext(ctx, "gcs.Bloom")
	defer span.Finish()

	name := util.BloomFileName(blockID, tenantID, shardNum)
	return rw.readAll(derivedCtx, name)
}

func (rw *readerWriter) Index(ctx context.Context, blockID uuid.UUID, tenantID string) ([]byte, error) {
	span, derivedCtx := opentracing.StartSpanFromContext(ctx, "gcs.Index")
	defer span.Finish()

	name := util.IndexFileName(blockID, tenantID)
	return rw.readAll(derivedCtx, name)
}

func (rw *readerWriter) Object(ctx context.Context, blockID uuid.UUID, tenantID string, start uint64, buffer []byte) error {
	span, derivedCtx := opentracing.StartSpanFromContext(ctx, "gcs.Object")
	defer span.Finish()

	name := util.ObjectFileName(blockID, tenantID)
	return rw.readRange(derivedCtx, name, int64(start), buffer)
}

func (rw *readerWriter) Shutdown() {

}

func (rw *readerWriter) writeAll(ctx context.Context, name string, b []byte) error {
	w := rw.writer(ctx, name)

	_, err := w.Write(b)
	if err != nil {
		w.Close()
		return err
	}

	err = w.Close()
	return err
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
