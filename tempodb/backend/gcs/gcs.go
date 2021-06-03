package gcs

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/cristalhq/hedgedhttp"
	"github.com/google/uuid"
	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	google_http "google.golang.org/api/transport/http"

	tempo_io "github.com/grafana/tempo/pkg/io"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/util"
)

type readerWriter struct {
	cfg          *Config
	bucket       *storage.BucketHandle
	hedgedBucket *storage.BucketHandle
}

func New(cfg *Config) (backend.Reader, backend.Writer, backend.Compactor, error) {
	ctx := context.Background()

	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	transportOptions := []option.ClientOption{
		option.WithScopes(storage.ScopeReadWrite),
	}
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

	// hedged client
	hedgedTransport := hedgedhttp.NewRoundTripper(500*time.Millisecond, 2, transport)
	hedgedClientOptions := []option.ClientOption{
		option.WithHTTPClient(&http.Client{
			Transport: hedgedTransport,
		}),
		option.WithScopes(storage.ScopeReadWrite),
	}
	if cfg.Endpoint != "" {
		hedgedClientOptions = append(hedgedClientOptions, option.WithEndpoint(cfg.Endpoint))
	}

	hedgedClient, err := storage.NewClient(ctx, hedgedClientOptions...)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "creating hedged storage client")
	}
	hedgedBucket := hedgedClient.Bucket(cfg.BucketName)

	// Check bucket exists by getting attrs
	if _, err = bucket.Attrs(ctx); err != nil {
		return nil, nil, nil, errors.Wrap(err, "getting bucket attrs")
	}

	rw := &readerWriter{
		cfg:          cfg,
		bucket:       bucket,
		hedgedBucket: hedgedBucket,
	}

	return rw, rw, rw, nil
}

// Write implements backend.Writer
func (rw *readerWriter) Write(ctx context.Context, name string, blockID uuid.UUID, tenantID string, buffer []byte) error {
	return rw.WriteReader(ctx, name, blockID, tenantID, bytes.NewBuffer(buffer), int64(len(buffer)))
}

// WriteReader implements backend.Writer
func (rw *readerWriter) WriteReader(ctx context.Context, name string, blockID uuid.UUID, tenantID string, data io.Reader, _ int64) error {
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
	if tracker == nil {
		return nil
	}

	w := tracker.(*storage.Writer)
	return w.Close()
}

// Tenants implements backend.Reader
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

// Tenants implements backend.Reader
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

// BlockMeta implements backend.Reader
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

// Read implements backend.Reader
func (rw *readerWriter) Read(ctx context.Context, name string, blockID uuid.UUID, tenantID string) ([]byte, error) {
	span, derivedCtx := opentracing.StartSpanFromContext(ctx, "gcs.Read")
	defer span.Finish()

	span.SetTag("object", name)

	bytes, err := rw.readAll(derivedCtx, util.ObjectFileName(blockID, tenantID, name))
	if err != nil {
		span.SetTag("error", true)
	}
	return bytes, err
}

func (rw *readerWriter) ReadReader(ctx context.Context, name string, blockID uuid.UUID, tenantID string) (io.ReadCloser, int64, error) {
	panic("ReadReader is not yet supported for GCS backend")
}

// ReadRange implements backend.Reader
func (rw *readerWriter) ReadRange(ctx context.Context, name string, blockID uuid.UUID, tenantID string, offset uint64, buffer []byte) error {
	span, derivedCtx := opentracing.StartSpanFromContext(ctx, "gcs.ReadRange")
	defer span.Finish()

	err := rw.readRange(derivedCtx, util.ObjectFileName(blockID, tenantID, name), int64(offset), buffer)
	if err != nil {
		span.SetTag("error", true)
	}
	return err
}

// Shutdown implements backend.Reader
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
	r, err := rw.hedgedBucket.Object(name).NewReader(ctx)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	return tempo_io.ReadAllWithEstimate(r, r.Attrs.Size)
}

func (rw *readerWriter) readAllWithModTime(ctx context.Context, name string) ([]byte, time.Time, error) {
	r, err := rw.hedgedBucket.Object(name).NewReader(ctx)
	if err != nil {
		return nil, time.Time{}, err
	}
	defer r.Close()

	buf, err := tempo_io.ReadAllWithEstimate(r, r.Attrs.Size)
	if err != nil {
		return nil, time.Time{}, err
	}

	return buf, r.Attrs.LastModified, nil
}

func (rw *readerWriter) readRange(ctx context.Context, name string, offset int64, buffer []byte) error {
	r, err := rw.hedgedBucket.Object(name).NewRangeReader(ctx, offset, int64(len(buffer)))
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
