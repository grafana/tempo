package cos

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/cristalhq/hedgedhttp"
	gkLog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	"github.com/tencentyun/cos-go-sdk-v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	tempo_io "github.com/grafana/tempo/pkg/io"
	"github.com/grafana/tempo/pkg/util/log"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/instrumentation"
)

type readerWriter struct {
	logger     gkLog.Logger
	cfg        *Config
	client     *cos.Client
	hedgedClient *cos.Client
}

var tracer = otel.Tracer("tempodb/backend/cos")

var (
	_ backend.RawReader             = (*readerWriter)(nil)
	_ backend.RawWriter             = (*readerWriter)(nil)
	_ backend.Compactor             = (*readerWriter)(nil)
	_ backend.VersionedReaderWriter = (*readerWriter)(nil)
)

type appendTracker struct {
	uploadID   string
	objectName string
	parts      []cos.Object
	partNum    int
}

func New(cfg *Config) (backend.RawReader, backend.RawWriter, backend.Compactor, error) {
	rw, err := internalNew(cfg, true)
	return rw, rw, rw, err
}

func NewNoConfirm(cfg *Config) (backend.RawReader, backend.RawWriter, backend.Compactor, error) {
	rw, err := internalNew(cfg, false)
	return rw, rw, rw, err
}

func NewVersionedReaderWriter(cfg *Config) (backend.VersionedReaderWriter, error) {
	return internalNew(cfg, true)
}

func internalNew(cfg *Config, confirm bool) (*readerWriter, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}

	if cfg.Bucket == "" {
		return nil, fmt.Errorf("bucket is required")
	}
	if cfg.SecretID == "" {
		return nil, fmt.Errorf("secret_id is required")
	}
	if cfg.SecretKey.String() == "" {
		return nil, fmt.Errorf("secret_key is required")
	}
	if cfg.Endpoint == "" && (cfg.Region == "" || cfg.AppID == "") {
		return nil, fmt.Errorf("region and app_id are required when endpoint is not set")
	}

	l := log.Logger

	client, err := createClient(cfg, false)
	if err != nil {
		return nil, fmt.Errorf("unexpected error creating cos client: %w", err)
	}

	hedgedClient, err := createClient(cfg, true)
	if err != nil {
		return nil, fmt.Errorf("unexpected error creating hedged cos client: %w", err)
	}

	if confirm {
		_, err = hedgedClient.Bucket.Head(context.Background())
		if err != nil {
			return nil, fmt.Errorf("unexpected error from Bucket.Head on %s: %w", cfg.Bucket, err)
		}
	}

	rw := &readerWriter{
		logger:       l,
		cfg:          cfg,
		client:       client,
		hedgedClient: hedgedClient,
	}

	return rw, nil
}

func bucketURL(cfg *Config) string {
	scheme := "https"
	if cfg.Insecure {
		scheme = "http"
	}

	host := cfg.Endpoint
	if host == "" {
		host = fmt.Sprintf("cos.%s.myqcloud.com", cfg.Region)
	}

	return fmt.Sprintf("%s://%s-%s.%s", scheme, cfg.Bucket, cfg.AppID, host)
}

func createClient(cfg *Config, hedge bool) (*cos.Client, error) {
	u, err := url.Parse(bucketURL(cfg))
	if err != nil {
		return nil, fmt.Errorf("failed to parse bucket URL: %w", err)
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxIdleConnsPerHost = 100
	transport.MaxIdleConns = 100

	var rt http.RoundTripper = transport
	rt = instrumentation.NewTransport(rt)

	if hedge && cfg.HedgeRequestsAt != 0 {
		var stats *hedgedhttp.Stats
		rt, stats, err = hedgedhttp.NewRoundTripperAndStats(cfg.HedgeRequestsAt, cfg.HedgeRequestsUpTo, rt)
		if err != nil {
			return nil, err
		}
		instrumentation.PublishHedgedMetrics(stats)
	}

	authTransport := &cos.AuthorizationTransport{
		SecretID:  cfg.SecretID,
		SecretKey: cfg.SecretKey.String(),
		Transport: rt,
	}

	client := cos.NewClient(&cos.BaseURL{BucketURL: u}, &http.Client{
		Transport: authTransport,
	})

	return client, nil
}

func (rw *readerWriter) Write(ctx context.Context, name string, keypath backend.KeyPath, data io.Reader, size int64, _ *backend.CacheInfo) error {
	derivedCtx, span := tracer.Start(ctx, "cos.Write")
	defer span.End()

	span.SetAttributes(attribute.String("object", name))

	keypath = backend.KeyPathWithPrefix(keypath, rw.cfg.Prefix)
	objName := backend.ObjectFileName(keypath, name)

	opt := &cos.ObjectPutOptions{
		ObjectPutHeaderOptions: &cos.ObjectPutHeaderOptions{
			ContentLength: size,
		},
	}
	_, err := rw.client.Object.Put(derivedCtx, objName, data, opt)
	if err != nil {
		span.SetStatus(codes.Error, "error writing object to cos backend")
		return fmt.Errorf("error writing object to cos backend, object %s: %w", objName, err)
	}
	level.Debug(rw.logger).Log("msg", "object uploaded to cos", "objectName", objName, "size", size)

	return nil
}

func (rw *readerWriter) Append(ctx context.Context, name string, keypath backend.KeyPath, tracker backend.AppendTracker, buffer []byte) (backend.AppendTracker, error) {
	ctx, span := tracer.Start(ctx, "cos.Append")
	defer span.End()

	var a appendTracker
	keypath = backend.KeyPathWithPrefix(keypath, rw.cfg.Prefix)
	objectName := backend.ObjectFileName(keypath, name)

	if tracker != nil {
		a = tracker.(appendTracker)
	} else {
		result, _, err := rw.client.Object.InitiateMultipartUpload(ctx, objectName, nil)
		if err != nil {
			return nil, fmt.Errorf("error initiating multipart upload: %w", err)
		}
		a.uploadID = result.UploadID
		a.objectName = objectName
	}

	a.partNum++
	opt := &cos.ObjectUploadPartOptions{
		ContentLength: int64(len(buffer)),
	}
	resp, err := rw.client.Object.UploadPart(ctx, objectName, a.uploadID, a.partNum, bytes.NewReader(buffer), opt)
	if err != nil {
		return a, fmt.Errorf("error in multipart upload: %w", err)
	}

	a.parts = append(a.parts, cos.Object{
		PartNumber: a.partNum,
		ETag:       resp.Header.Get("ETag"),
	})

	return a, nil
}

func (rw *readerWriter) CloseAppend(ctx context.Context, tracker backend.AppendTracker) error {
	if tracker == nil {
		return nil
	}

	a := tracker.(appendTracker)

	completeOpt := &cos.CompleteMultipartUploadOptions{
		Parts: a.parts,
	}
	_, _, err := rw.client.Object.CompleteMultipartUpload(ctx, a.objectName, a.uploadID, completeOpt)
	if err != nil {
		_, _ = rw.client.Object.AbortMultipartUpload(ctx, a.objectName, a.uploadID)
		return fmt.Errorf("error completing multipart upload, object: %s: %w", a.objectName, err)
	}

	return nil
}

func (rw *readerWriter) Delete(ctx context.Context, name string, keypath backend.KeyPath, _ *backend.CacheInfo) error {
	keypath = backend.KeyPathWithPrefix(keypath, rw.cfg.Prefix)
	_, err := rw.client.Object.Delete(ctx, backend.ObjectFileName(keypath, name))
	if err != nil {
		return readError(err)
	}
	return nil
}

func (rw *readerWriter) List(ctx context.Context, keypath backend.KeyPath) ([]string, error) {
	keypath = backend.KeyPathWithPrefix(keypath, rw.cfg.Prefix)
	prefix := path.Join(keypath...)
	var objects []string

	if len(prefix) > 0 {
		prefix += "/"
	}

	marker := ""
	isTruncated := true
	for isTruncated {
		opt := &cos.BucketGetOptions{
			Prefix:    prefix,
			Delimiter: "/",
			Marker:    marker,
			MaxKeys:   1000,
		}
		result, _, err := rw.client.Bucket.Get(ctx, opt)
		if err != nil {
			return nil, fmt.Errorf("error listing objects in cos bucket, bucket: %s: %w", rw.cfg.Bucket, err)
		}
		isTruncated = result.IsTruncated
		marker = result.NextMarker

		for _, cp := range result.CommonPrefixes {
			objects = append(objects, strings.Split(strings.TrimPrefix(cp, prefix), "/")[0])
		}
	}

	return objects, nil
}

func (rw *readerWriter) ListBlocks(
	ctx context.Context,
	tenant string,
) ([]uuid.UUID, []uuid.UUID, error) {
	ctx, span := tracer.Start(ctx, "readerWriter.ListBlocks")
	defer span.End()

	var (
		blockIDs          = make([]uuid.UUID, 0, 1000)
		compactedBlockIDs = make([]uuid.UUID, 0, 1000)
		keypath           = backend.KeyPathWithPrefix(backend.KeyPath{tenant}, rw.cfg.Prefix)
	)

	prefix := path.Join(keypath...)
	if len(prefix) > 0 {
		prefix += "/"
	}

	marker := ""
	isTruncated := true
	for isTruncated {
		opt := &cos.BucketGetOptions{
			Prefix:  prefix,
			Marker:  marker,
			MaxKeys: 1000,
		}
		result, _, err := rw.client.Bucket.Get(ctx, opt)
		if err != nil {
			return nil, nil, fmt.Errorf("error listing blocks in cos bucket, bucket: %s: %w", rw.cfg.Bucket, err)
		}
		isTruncated = result.IsTruncated
		marker = result.NextMarker

		for _, c := range result.Contents {
			parts := strings.Split(strings.TrimPrefix(c.Key, prefix), "/")
			if len(parts) != 2 {
				continue
			}

			switch parts[1] {
			case backend.MetaName:
			case backend.CompactedMetaName:
			default:
				continue
			}

			id, err := uuid.Parse(parts[0])
			if err != nil {
				continue
			}

			switch parts[1] {
			case backend.MetaName:
				blockIDs = append(blockIDs, id)
			case backend.CompactedMetaName:
				compactedBlockIDs = append(compactedBlockIDs, id)
			}
		}
	}

	level.Debug(rw.logger).Log("msg", "listing blocks complete", "blockIDs", len(blockIDs), "compactedBlockIDs", len(compactedBlockIDs))

	return blockIDs, compactedBlockIDs, nil
}

func (rw *readerWriter) Find(ctx context.Context, keypath backend.KeyPath, f backend.FindFunc) (err error) {
	keypath = backend.KeyPathWithPrefix(keypath, rw.cfg.Prefix)
	prefix := path.Join(keypath...)

	if len(prefix) > 0 {
		prefix += "/"
	}

	marker := ""
	isTruncated := true
	for isTruncated {
		select {
		case <-ctx.Done():
			return
		default:
			opt := &cos.BucketGetOptions{
				Prefix:  prefix,
				Marker:  marker,
				MaxKeys: 1000,
			}
			result, _, err := rw.client.Bucket.Get(ctx, opt)
			if err != nil {
				return fmt.Errorf("error finding objects in cos bucket, bucket: %s: %w", rw.cfg.Bucket, err)
			}

			isTruncated = result.IsTruncated
			marker = result.NextMarker

			for _, c := range result.Contents {
				lastMod, _ := time.Parse(time.RFC3339, c.LastModified)
				opts := backend.FindMatch{
					Key:      strings.TrimPrefix(c.Key, rw.cfg.Prefix),
					Modified: lastMod,
				}
				f(opts)
			}
		}
	}

	return
}

func (rw *readerWriter) Read(ctx context.Context, name string, keypath backend.KeyPath, _ *backend.CacheInfo) (io.ReadCloser, int64, error) {
	derivedCtx, span := tracer.Start(ctx, "cos.Read")
	defer span.End()

	keypath = backend.KeyPathWithPrefix(keypath, rw.cfg.Prefix)
	b, _, err := rw.readAll(derivedCtx, backend.ObjectFileName(keypath, name))
	if err != nil {
		return nil, 0, readError(err)
	}

	return io.NopCloser(bytes.NewReader(b)), int64(len(b)), nil
}

func (rw *readerWriter) ReadRange(ctx context.Context, name string, keypath backend.KeyPath, offset uint64, buffer []byte, _ *backend.CacheInfo) error {
	derivedCtx, span := tracer.Start(ctx, "cos.ReadRange")
	defer span.End()

	keypath = backend.KeyPathWithPrefix(keypath, rw.cfg.Prefix)
	return readError(rw.readRange(derivedCtx, backend.ObjectFileName(keypath, name), int64(offset), buffer))
}

func (rw *readerWriter) Shutdown() {
}

func (rw *readerWriter) WriteVersioned(ctx context.Context, name string, keypath backend.KeyPath, data io.Reader, size int64, version backend.Version) (backend.Version, error) {
	_, currentVersion, err := rw.ReadVersioned(ctx, name, keypath)
	if err != nil && !errors.Is(err, backend.ErrDoesNotExist) {
		return "", err
	}

	level.Info(rw.logger).Log("msg", "WriteVersioned - fetching data", "currentVersion", currentVersion, "err", err, "version", version)

	if errors.Is(err, backend.ErrDoesNotExist) && version != backend.VersionNew {
		return "", backend.ErrVersionDoesNotMatch
	}
	if !errors.Is(err, backend.ErrDoesNotExist) && version != currentVersion {
		return "", backend.ErrVersionDoesNotMatch
	}

	err = rw.Write(ctx, name, keypath, data, size, nil)
	if err != nil {
		return "", err
	}

	_, currentVersion, err = rw.ReadVersioned(ctx, name, keypath)
	return currentVersion, err
}

func (rw *readerWriter) DeleteVersioned(ctx context.Context, name string, keypath backend.KeyPath, version backend.Version) error {
	keypath = backend.KeyPathWithPrefix(keypath, rw.cfg.Prefix)

	_, currentVersion, err := rw.ReadVersioned(ctx, name, keypath)
	if err != nil && !errors.Is(err, backend.ErrDoesNotExist) {
		return err
	}
	if !errors.Is(err, backend.ErrDoesNotExist) && currentVersion != version {
		return backend.ErrVersionDoesNotMatch
	}

	return rw.Delete(ctx, name, keypath, nil)
}

func (rw *readerWriter) ReadVersioned(ctx context.Context, name string, keypath backend.KeyPath) (io.ReadCloser, backend.Version, error) {
	derivedCtx, span := tracer.Start(ctx, "cos.ReadVersioned")
	defer span.End()

	keypath = backend.KeyPathWithPrefix(keypath, rw.cfg.Prefix)
	b, etag, err := rw.readAll(derivedCtx, backend.ObjectFileName(keypath, name))
	if err != nil {
		return nil, "", readError(err)
	}

	return io.NopCloser(bytes.NewReader(b)), backend.Version(etag), nil
}

func (rw *readerWriter) readAll(ctx context.Context, name string) ([]byte, string, error) {
	resp, err := rw.hedgedClient.Object.Get(ctx, name, nil)
	if err != nil {
		if cos.IsNotFoundError(err) {
			return nil, "", backend.ErrDoesNotExist
		}
		return nil, "", fmt.Errorf("error fetching object from cos backend: %w", err)
	}
	defer resp.Body.Close()

	buf, err := tempo_io.ReadAllWithEstimate(resp.Body, resp.ContentLength)
	if err != nil {
		return nil, "", fmt.Errorf("error reading response from cos backend: %w", err)
	}

	return buf, resp.Header.Get("ETag"), nil
}

func (rw *readerWriter) readAllWithObjInfo(ctx context.Context, name string) ([]byte, cos.Object, error) {
	resp, err := rw.hedgedClient.Object.Get(ctx, name, nil)
	if err != nil {
		if cos.IsNotFoundError(err) {
			return nil, cos.Object{}, backend.ErrDoesNotExist
		}
		return nil, cos.Object{}, fmt.Errorf("error fetching object from cos backend: %w", err)
	}
	defer resp.Body.Close()

	buf, err := tempo_io.ReadAllWithEstimate(resp.Body, resp.ContentLength)
	if err != nil {
		return nil, cos.Object{}, fmt.Errorf("error reading response from cos backend: %w", err)
	}

	return buf, cos.Object{
		Key:          name,
		ETag:         resp.Header.Get("ETag"),
		LastModified: resp.Header.Get("Last-Modified"),
		Size:         resp.ContentLength,
	}, nil
}

func (rw *readerWriter) readRange(ctx context.Context, objName string, offset int64, buffer []byte) error {
	opt := &cos.ObjectGetOptions{
		Range: fmt.Sprintf("bytes=%d-%d", offset, offset+int64(len(buffer))-1),
	}
	resp, err := rw.hedgedClient.Object.Get(ctx, objName, opt)
	if err != nil {
		return fmt.Errorf("error in range read from cos backend, objName: %s: %w", objName, err)
	}
	defer resp.Body.Close()

	_, err = io.ReadFull(resp.Body, buffer)
	if err == nil {
		var dummy [1]byte
		_, _ = resp.Body.Read(dummy[:])
		return nil
	}

	return err
}

func readError(err error) error {
	if err != nil && cos.IsNotFoundError(err) {
		return backend.ErrDoesNotExist
	}
	return err
}
