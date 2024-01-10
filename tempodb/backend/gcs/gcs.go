package gcs

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"
	"sync"

	"github.com/google/uuid"

	"github.com/grafana/tempo/tempodb/backend/instrumentation"

	"cloud.google.com/go/storage"
	"github.com/cristalhq/hedgedhttp"
	opentracing "github.com/opentracing/opentracing-go"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	google_http "google.golang.org/api/transport/http"

	"github.com/grafana/tempo/pkg/blockboundary"
	tempo_io "github.com/grafana/tempo/pkg/io"
	"github.com/grafana/tempo/tempodb/backend"
)

type readerWriter struct {
	cfg          *Config
	bucket       *storage.BucketHandle
	hedgedBucket *storage.BucketHandle
}

var (
	_ backend.RawReader             = (*readerWriter)(nil)
	_ backend.RawWriter             = (*readerWriter)(nil)
	_ backend.Compactor             = (*readerWriter)(nil)
	_ backend.VersionedReaderWriter = (*readerWriter)(nil)
)

// NewNoConfirm gets the GCS backend without testing it
func NewNoConfirm(cfg *Config) (backend.RawReader, backend.RawWriter, backend.Compactor, error) {
	rw, err := internalNew(cfg, false)
	return rw, rw, rw, err
}

// New gets the GCS backend
func New(cfg *Config) (backend.RawReader, backend.RawWriter, backend.Compactor, error) {
	rw, err := internalNew(cfg, true)
	return rw, rw, rw, err
}

func NewVersionedReaderWriter(cfg *Config, confirmVersioning bool) (backend.VersionedReaderWriter, error) {
	rw, err := internalNew(cfg, true)
	if err != nil {
		return nil, err
	}

	if confirmVersioning {
		bucketAttrs, err := rw.bucket.Attrs(context.Background())
		if err != nil {
			return nil, fmt.Errorf("getting bucket attrs: %w", err)
		}

		if !bucketAttrs.VersioningEnabled {
			return nil, errors.New("versioning is not enabled on bucket")
		}
	}

	return rw, nil
}

func internalNew(cfg *Config, confirm bool) (*readerWriter, error) {
	ctx := context.Background()

	bucket, err := createBucket(ctx, cfg, false)
	if err != nil {
		return nil, fmt.Errorf("creating bucket: %w", err)
	}

	hedgedBucket, err := createBucket(ctx, cfg, true)
	if err != nil {
		return nil, fmt.Errorf("creating hedged bucket: %w", err)
	}

	// Check bucket exists by getting attrs
	if confirm {
		if _, err = bucket.Attrs(ctx); err != nil {
			return nil, fmt.Errorf("getting bucket attrs: %w", err)
		}
	}

	rw := &readerWriter{
		cfg:          cfg,
		bucket:       bucket,
		hedgedBucket: hedgedBucket,
	}

	return rw, nil
}

// Write implements backend.Writer
func (rw *readerWriter) Write(ctx context.Context, name string, keypath backend.KeyPath, data io.Reader, _ int64, _ *backend.CacheInfo) error {
	keypath = backend.KeyPathWithPrefix(keypath, rw.cfg.Prefix)
	span, derivedCtx := opentracing.StartSpanFromContext(ctx, "gcs.Write")
	defer span.Finish()

	span.SetTag("object", name)

	w := rw.writer(derivedCtx, backend.ObjectFileName(keypath, name), nil)

	_, err := io.Copy(w, data)
	if err != nil {
		w.Close()
		span.SetTag("error", true)
		return fmt.Errorf("failed to write: %w", err)
	}

	return w.Close()
}

// Append implements backend.Writer
func (rw *readerWriter) Append(ctx context.Context, name string, keypath backend.KeyPath, tracker backend.AppendTracker, buffer []byte) (backend.AppendTracker, error) {
	keypath = backend.KeyPathWithPrefix(keypath, rw.cfg.Prefix)
	span, ctx := opentracing.StartSpanFromContext(ctx, "gcs.Append", opentracing.Tags{
		"len": len(buffer),
	})
	defer span.Finish()

	var w *storage.Writer
	if tracker == nil {
		w = rw.writer(ctx, backend.ObjectFileName(keypath, name), nil)
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

func (rw *readerWriter) Delete(ctx context.Context, name string, keypath backend.KeyPath, _ *backend.CacheInfo) error {
	return readError(rw.bucket.Object(backend.ObjectFileName(keypath, name)).Delete(ctx))
}

// List implements backend.Reader
func (rw *readerWriter) List(ctx context.Context, keypath backend.KeyPath) ([]string, error) {
	keypath = backend.KeyPathWithPrefix(keypath, rw.cfg.Prefix)
	prefix := path.Join(keypath...)
	if len(prefix) > 0 {
		prefix = prefix + "/"
	}
	iter := rw.bucket.Objects(ctx, &storage.Query{
		Prefix:    prefix,
		Delimiter: "/",
		Versions:  false,
	})

	var objects []string
	for {
		attrs, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("iterating blocks: %w", err)
		}

		obj := strings.TrimSuffix(strings.TrimPrefix(attrs.Prefix, prefix), "/")
		objects = append(objects, obj)
	}

	return objects, nil
}

// ListBlocks implements backend.Reader
func (rw *readerWriter) ListBlocks(ctx context.Context, tenant string) ([]uuid.UUID, []uuid.UUID, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "readerWriter.ListBlocks")
	defer span.Finish()

	var (
		wg                sync.WaitGroup
		mtx               sync.Mutex
		bb                = blockboundary.CreateBlockBoundaries(rw.cfg.ListBlocksConcurrency)
		errChan           = make(chan error, len(bb))
		keypath           = backend.KeyPathWithPrefix(backend.KeyPath{tenant}, rw.cfg.Prefix)
		min               uuid.UUID
		max               uuid.UUID
		blockIDs          = make([]uuid.UUID, 0, 1000)
		compactedBlockIDs = make([]uuid.UUID, 0, 1000)
	)

	prefix := path.Join(keypath...)
	if len(prefix) > 0 {
		prefix += "/"
	}

	for i := 0; i < len(bb)-1; i++ {
		min = uuid.UUID(bb[i])
		max = uuid.UUID(bb[i+1])

		wg.Add(1)
		go func(min, max uuid.UUID) {
			defer wg.Done()

			var (
				query = &storage.Query{
					Prefix:      prefix,
					Delimiter:   "",
					Versions:    false,
					StartOffset: prefix + min.String(),
				}
				parts []string
				id    uuid.UUID
			)

			// If max is global max, then we don't want to set an end offset to
			// ensure we reach the end.  EndOffset is exclusive.
			if max != backend.GlobalMaxBlockID {
				query.EndOffset = prefix + max.String()
			}

			iter := rw.bucket.Objects(ctx, query)

			for {
				if ctx.Err() != nil {
					return
				}

				attrs, err := iter.Next()
				if errors.Is(err, iterator.Done) {
					return
				}
				if err != nil {
					errChan <- fmt.Errorf("iterating blocks: %w", err)
					return
				}

				parts = strings.Split(attrs.Name, "/")
				// ie: <tenant>/<blockID>/meta.json
				if len(parts) != 3 {
					continue
				}

				switch parts[2] {
				case backend.MetaName:
				case backend.CompactedMetaName:
				default:
					continue
				}

				id, err = uuid.Parse(parts[1])
				if err != nil {
					continue
				}

				if bytes.Compare(id[:], min[:]) < 0 {
					errChan <- fmt.Errorf("block UUID below shard minimum")
					return
				}

				if max != backend.GlobalMaxBlockID {
					if bytes.Compare(id[:], max[:]) >= 0 {
						return
					}
				}

				mtx.Lock()
				switch parts[2] {
				case backend.MetaName:
					blockIDs = append(blockIDs, id)
				case backend.CompactedMetaName:
					compactedBlockIDs = append(compactedBlockIDs, id)
				}
				mtx.Unlock()
			}
		}(min, max)
	}
	wg.Wait()
	close(errChan)

	errs := make([]error, 0, len(errChan))
	for e := range errChan {
		errs = append(errs, e)
	}

	if len(errs) > 0 {
		return nil, nil, errors.Join(errs...)
	}

	return blockIDs, compactedBlockIDs, nil
}

// Read implements backend.Reader
func (rw *readerWriter) Read(ctx context.Context, name string, keypath backend.KeyPath, _ *backend.CacheInfo) (io.ReadCloser, int64, error) {
	keypath = backend.KeyPathWithPrefix(keypath, rw.cfg.Prefix)
	span, derivedCtx := opentracing.StartSpanFromContext(ctx, "gcs.Read")
	defer span.Finish()

	span.SetTag("object", name)

	b, _, err := rw.readAll(derivedCtx, backend.ObjectFileName(keypath, name))
	if err != nil {
		span.SetTag("error", true)
	}
	return io.NopCloser(bytes.NewReader(b)), int64(len(b)), readError(err)
}

// ReadRange implements backend.Reader
func (rw *readerWriter) ReadRange(ctx context.Context, name string, keypath backend.KeyPath, offset uint64, buffer []byte, _ *backend.CacheInfo) error {
	keypath = backend.KeyPathWithPrefix(keypath, rw.cfg.Prefix)
	span, derivedCtx := opentracing.StartSpanFromContext(ctx, "gcs.ReadRange", opentracing.Tags{
		"len":    len(buffer),
		"offset": offset,
	})
	defer span.Finish()

	err := rw.readRange(derivedCtx, backend.ObjectFileName(keypath, name), int64(offset), buffer)
	if err != nil {
		span.SetTag("error", true)
	}
	return readError(err)
}

// Shutdown implements backend.Reader
func (rw *readerWriter) Shutdown() {
}

func (rw *readerWriter) WriteVersioned(ctx context.Context, name string, keypath backend.KeyPath, data io.Reader, version backend.Version) (backend.Version, error) {
	keypath = backend.KeyPathWithPrefix(keypath, rw.cfg.Prefix)
	span, derivedCtx := opentracing.StartSpanFromContext(ctx, "gcs.WriteVersioned", opentracing.Tags{
		"object": name,
	})
	defer span.Finish()

	preconditions, err := createPreconditions(version)
	if err != nil {
		return "", err
	}

	w := rw.writer(derivedCtx, backend.ObjectFileName(keypath, name), &preconditions)

	_, err = io.Copy(w, data)
	if err != nil {
		w.Close()
		span.SetTag("error", true)
		return "", fmt.Errorf("failed to write: %w", err)
	}

	err = w.Close()
	if err != nil {
		return "", err
	}

	return toVersion(w.Attrs().Generation), nil
}

func (rw *readerWriter) DeleteVersioned(ctx context.Context, name string, keypath backend.KeyPath, version backend.Version) error {
	o := rw.bucket.Object(backend.ObjectFileName(keypath, name))

	preconditions, err := createPreconditions(version)
	if err != nil {
		return err
	}
	o = o.If(preconditions)

	return o.Delete(ctx)
}

func (rw *readerWriter) ReadVersioned(ctx context.Context, name string, keypath backend.KeyPath) (io.ReadCloser, backend.Version, error) {
	keypath = backend.KeyPathWithPrefix(keypath, rw.cfg.Prefix)
	span, derivedCtx := opentracing.StartSpanFromContext(ctx, "gcs.ReadVersioned", opentracing.Tags{
		"object": name,
	})
	defer span.Finish()

	b, attrs, err := rw.readAll(derivedCtx, backend.ObjectFileName(keypath, name))
	if err != nil {
		span.SetTag("error", true)
		return nil, "", readError(err)
	}
	return io.NopCloser(bytes.NewReader(b)), toVersion(attrs.Generation), nil
}

func toVersion(generation int64) backend.Version {
	return backend.Version(fmt.Sprint(generation))
}

func (rw *readerWriter) writer(ctx context.Context, name string, conditions *storage.Conditions) *storage.Writer {
	o := rw.bucket.Object(name)
	if conditions != nil {
		o = o.If(*conditions)
	}
	w := o.NewWriter(ctx)
	w.ChunkSize = rw.cfg.ChunkBufferSize

	if rw.cfg.ObjectMetadata != nil {
		w.Metadata = rw.cfg.ObjectMetadata
	}

	if rw.cfg.ObjectCacheControl != "" {
		w.CacheControl = rw.cfg.ObjectCacheControl
	}

	return w
}

func (rw *readerWriter) readAll(ctx context.Context, name string) ([]byte, *storage.ReaderObjectAttrs, error) {
	r, err := rw.hedgedBucket.Object(name).NewReader(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer r.Close()

	buf, err := tempo_io.ReadAllWithEstimate(r, r.Attrs.Size)
	if err != nil {
		return nil, nil, err
	}

	return buf, &r.Attrs, nil
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
		if errors.Is(err, io.EOF) {
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

func createBucket(ctx context.Context, cfg *Config, hedge bool) (*storage.BucketHandle, error) {
	// start with default transport
	customTransport := http.DefaultTransport.(*http.Transport).Clone()

	// add google auth
	transportOptions := []option.ClientOption{
		option.WithScopes(storage.ScopeReadWrite),
	}
	if cfg.Insecure {
		transportOptions = append(transportOptions, option.WithoutAuthentication())
		customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	transport, err := google_http.NewTransport(ctx, customTransport, transportOptions...)
	if err != nil {
		return nil, fmt.Errorf("creating google http transport: %w", err)
	}

	// add instrumentation
	transport = instrumentation.NewTransport(transport)
	var stats *hedgedhttp.Stats

	// hedge if desired (0 means disabled)
	if hedge && cfg.HedgeRequestsAt != 0 {
		transport, stats, err = hedgedhttp.NewRoundTripperAndStats(cfg.HedgeRequestsAt, cfg.HedgeRequestsUpTo, transport)
		if err != nil {
			return nil, err
		}
		instrumentation.PublishHedgedMetrics(stats)
	}

	// Build client
	storageClientOptions := []option.ClientOption{
		option.WithHTTPClient(&http.Client{
			Transport: transport,
		}),
		option.WithScopes(storage.ScopeReadWrite),
	}
	if cfg.Endpoint != "" {
		storageClientOptions = append(storageClientOptions, option.WithEndpoint(cfg.Endpoint))
		storageClientOptions = append(storageClientOptions, storage.WithJSONReads())
	}
	client, err := storage.NewClient(ctx, storageClientOptions...)
	if err != nil {
		return nil, fmt.Errorf("creating storage client: %w", err)
	}

	// Build bucket
	return client.Bucket(cfg.BucketName), nil
}

func readError(err error) error {
	if errors.Is(err, storage.ErrObjectNotExist) {
		return backend.ErrDoesNotExist
	}

	return err
}

func createPreconditions(version backend.Version) (preconditions storage.Conditions, err error) {
	if version == backend.VersionNew {
		preconditions.DoesNotExist = true
		return
	}

	generation, err := strconv.ParseInt(string(version), 10, 64)
	if err != nil {
		return storage.Conditions{}, backend.ErrVersionInvalid
	}
	preconditions.GenerationMatch = generation
	return
}
