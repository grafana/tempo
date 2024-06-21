package s3

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/google/uuid"

	"github.com/grafana/tempo/tempodb/backend/instrumentation"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/cristalhq/hedgedhttp"
	gkLog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	minio "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/opentracing/opentracing-go"

	"github.com/grafana/tempo/pkg/blockboundary"
	tempo_io "github.com/grafana/tempo/pkg/io"
	"github.com/grafana/tempo/pkg/util/log"
	"github.com/grafana/tempo/tempodb/backend"
)

// readerWriter can read/write from an s3 backend
type readerWriter struct {
	logger     gkLog.Logger
	cfg        *Config
	core       *minio.Core
	hedgedCore *minio.Core
}

var (
	_ backend.RawReader             = (*readerWriter)(nil)
	_ backend.RawWriter             = (*readerWriter)(nil)
	_ backend.Compactor             = (*readerWriter)(nil)
	_ backend.VersionedReaderWriter = (*readerWriter)(nil)
)

// appendTracker is a struct used to track multipart uploads
type appendTracker struct {
	uploadID   string
	objectName string
	parts      []minio.ObjectPart
	partNum    int
}

type overrideSignatureVersion struct {
	upstream credentials.Provider
	useV2    bool
}

func (s *overrideSignatureVersion) Retrieve() (credentials.Value, error) {
	v, err := s.upstream.Retrieve()
	if err != nil {
		return v, err
	}

	if s.useV2 && !v.SignerType.IsAnonymous() {
		v.SignerType = credentials.SignatureV2
	}

	return v, nil
}

func (s *overrideSignatureVersion) IsExpired() bool {
	return s.upstream.IsExpired()
}

// NewNoConfirm gets the S3 backend without testing it
func NewNoConfirm(cfg *Config) (backend.RawReader, backend.RawWriter, backend.Compactor, error) {
	rw, err := internalNew(cfg, false)
	return rw, rw, rw, err
}

// New gets the S3 backend
func New(cfg *Config) (backend.RawReader, backend.RawWriter, backend.Compactor, error) {
	rw, err := internalNew(cfg, true)
	return rw, rw, rw, err
}

// NewVersionedReaderWriter creates a client to perform versioned requests. Note that write requests are
// best-effort since the S3 API does not support precondition headers.
func NewVersionedReaderWriter(cfg *Config) (backend.VersionedReaderWriter, error) {
	return internalNew(cfg, true)
}

func internalNew(cfg *Config, confirm bool) (*readerWriter, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}

	l := log.Logger

	core, err := createCore(cfg, false)
	if err != nil {
		return nil, fmt.Errorf("unexpected error creating core: %w", err)
	}

	hedgedCore, err := createCore(cfg, true)
	if err != nil {
		return nil, fmt.Errorf("unexpected error creating hedgedCore: %w", err)
	}

	// try listing objects
	if confirm {
		_, err = core.ListObjects(cfg.Bucket, cfg.Prefix, "", "/", 0)
		if err != nil {
			return nil, fmt.Errorf("unexpected error from ListObjects on %s: %w", cfg.Bucket, err)
		}
	}

	rw := &readerWriter{
		logger:     l,
		cfg:        cfg,
		core:       core,
		hedgedCore: hedgedCore,
	}
	return rw, nil
}

func getPutObjectOptions(rw *readerWriter) minio.PutObjectOptions {
	return minio.PutObjectOptions{
		PartSize:     rw.cfg.PartSize,
		UserTags:     rw.cfg.Tags,
		StorageClass: rw.cfg.StorageClass,
		UserMetadata: rw.cfg.Metadata,
	}
}

// Write implements backend.Writer
func (rw *readerWriter) Write(ctx context.Context, name string, keypath backend.KeyPath, data io.Reader, size int64, _ *backend.CacheInfo) error {
	span, derivedCtx := opentracing.StartSpanFromContext(ctx, "s3.Write")
	defer span.Finish()

	span.SetTag("object", name)

	keypath = backend.KeyPathWithPrefix(keypath, rw.cfg.Prefix)
	objName := backend.ObjectFileName(keypath, name)

	putObjectOptions := getPutObjectOptions(rw)

	info, err := rw.core.Client.PutObject(
		derivedCtx,
		rw.cfg.Bucket,
		objName,
		data,
		size,
		putObjectOptions,
	)
	if err != nil {
		span.SetTag("error", true)
		return fmt.Errorf("error writing object to s3 backend, object %s: %w", objName, err)
	}
	level.Debug(rw.logger).Log("msg", "object uploaded to s3", "objectName", objName, "size", info.Size)

	return nil
}

// AppendObject implements backend.Writer
func (rw *readerWriter) Append(ctx context.Context, name string, keypath backend.KeyPath, tracker backend.AppendTracker, buffer []byte) (backend.AppendTracker, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "s3.Append", opentracing.Tags{
		"len": len(buffer),
	})
	defer span.Finish()

	var a appendTracker
	keypath = backend.KeyPathWithPrefix(keypath, rw.cfg.Prefix)
	objectName := backend.ObjectFileName(keypath, name)

	options := getPutObjectOptions(rw)
	if tracker != nil {
		a = tracker.(appendTracker)
	} else {
		id, err := rw.core.NewMultipartUpload(
			ctx,
			rw.cfg.Bucket,
			objectName,
			options,
		)
		if err != nil {
			return nil, err
		}
		a.uploadID = id
		a.objectName = objectName
	}

	level.Debug(rw.logger).Log("msg", "appending object to s3", "objectName", objectName)

	a.partNum++
	objPart, err := rw.core.PutObjectPart(
		ctx,
		rw.cfg.Bucket,
		objectName,
		a.uploadID,
		a.partNum,
		bytes.NewReader(buffer),
		int64(len(buffer)),
		minio.PutObjectPartOptions{},
	)
	if err != nil {
		return a, fmt.Errorf("error in multipart upload: %w", err)
	}
	a.parts = append(a.parts, objPart)

	return a, nil
}

// CloseAppend implements backend.Writer
func (rw *readerWriter) CloseAppend(ctx context.Context, tracker backend.AppendTracker) error {
	if tracker == nil {
		return nil
	}

	a := tracker.(appendTracker)
	completeParts := make([]minio.CompletePart, 0)
	for _, p := range a.parts {
		completeParts = append(completeParts, minio.CompletePart{
			PartNumber: p.PartNumber,
			ETag:       p.ETag,
		})
	}

	uploadInfo, err := rw.core.CompleteMultipartUpload(
		ctx,
		rw.cfg.Bucket,
		a.objectName,
		a.uploadID,
		completeParts,
		minio.PutObjectOptions{},
	)
	if err != nil {
		return fmt.Errorf("error completing multipart upload, object: %s, obj etag: %s: %w", a.objectName, uploadInfo.ETag, err)
	}

	return nil
}

func (rw *readerWriter) Delete(ctx context.Context, name string, keypath backend.KeyPath, _ *backend.CacheInfo) error {
	filename := backend.ObjectFileName(keypath, name)
	return rw.core.RemoveObject(ctx, rw.cfg.Bucket, filename, minio.RemoveObjectOptions{})
}

// List implements backend.Reader
func (rw *readerWriter) List(_ context.Context, keypath backend.KeyPath) ([]string, error) {
	keypath = backend.KeyPathWithPrefix(keypath, rw.cfg.Prefix)
	prefix := path.Join(keypath...)
	var objects []string

	if len(prefix) > 0 {
		prefix = prefix + "/"
	}

	nextMarker := ""
	isTruncated := true
	for isTruncated {
		// ListObjects(bucket, prefix, nextMarker, delimiter string, maxKeys int)
		res, err := rw.core.ListObjects(rw.cfg.Bucket, prefix, nextMarker, "/", 0)
		if err != nil {
			return nil, fmt.Errorf("error listing blocks in s3 bucket, bucket: %s: %w", rw.cfg.Bucket, err)
		}
		isTruncated = res.IsTruncated
		nextMarker = res.NextMarker

		level.Debug(rw.logger).Log("msg", "listing blocks", "keypath", path.Join(keypath...)+"/",
			"found", len(res.CommonPrefixes), "IsTruncated", res.IsTruncated, "NextMarker", res.NextMarker)

		for _, cp := range res.CommonPrefixes {
			objects = append(objects, strings.Split(strings.TrimPrefix(cp.Prefix, prefix), "/")[0])
		}
	}

	return objects, nil
}

func (rw *readerWriter) ListBlocks(
	ctx context.Context,
	tenant string,
) ([]uuid.UUID, []uuid.UUID, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "readerWriter.ListBlocks")
	defer span.Finish()

	blockIDs := make([]uuid.UUID, 0, 1000)
	compactedBlockIDs := make([]uuid.UUID, 0, 1000)

	keypath := backend.KeyPathWithPrefix(backend.KeyPath{tenant}, rw.cfg.Prefix)
	prefix := path.Join(keypath...)
	if len(prefix) > 0 {
		prefix += "/"
	}

	bb := blockboundary.CreateBlockBoundaries(rw.cfg.ListBlocksConcurrency)

	errChan := make(chan error, len(bb))
	wg := sync.WaitGroup{}
	mtx := sync.Mutex{}

	var min uuid.UUID
	var max uuid.UUID

	for i := 0; i < len(bb)-1; i++ {

		min = uuid.UUID(bb[i])
		max = uuid.UUID(bb[i+1])

		wg.Add(1)
		go func(min, max uuid.UUID) {
			defer wg.Done()

			var (
				err        error
				res        minio.ListBucketV2Result
				startAfter = prefix + min.String()
			)

			for res.IsTruncated = true; res.IsTruncated; {
				if ctx.Err() != nil {
					return
				}

				res, err = rw.core.ListObjectsV2(rw.cfg.Bucket, prefix, startAfter, res.NextContinuationToken, "", 0)
				if err != nil {
					errChan <- fmt.Errorf("error finding objects in s3 bucket, bucket: %s: %w", rw.cfg.Bucket, err)
					return
				}

				for _, c := range res.Contents {
					// i.e: <blockID>/meta
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
					switch parts[1] {
					case backend.MetaName:
						blockIDs = append(blockIDs, id)
					case backend.CompactedMetaName:
						compactedBlockIDs = append(compactedBlockIDs, id)
					}
					mtx.Unlock()
				}
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

	level.Debug(rw.logger).Log("msg", "listing blocks complete", "blockIDs", len(blockIDs), "compactedBlockIDs", len(compactedBlockIDs))

	return blockIDs, compactedBlockIDs, nil
}

// Find implements backend.Reader
func (rw *readerWriter) Find(ctx context.Context, keypath backend.KeyPath, f backend.FindFunc) (err error) {
	keypath = backend.KeyPathWithPrefix(keypath, rw.cfg.Prefix)
	prefix := path.Join(keypath...)

	if len(prefix) > 0 {
		prefix = prefix + "/"
	}

	nextToken := ""
	isTruncated := true
	var res minio.ListBucketV2Result

	for isTruncated {
		select {
		case <-ctx.Done():
			return
		default:
			res, err = rw.core.ListObjectsV2(rw.cfg.Bucket, prefix, "", nextToken, "", 0)
			if err != nil {
				return fmt.Errorf("error finding objects in s3 bucket, bucket: %s: %w", rw.cfg.Bucket, err)
			}

			isTruncated = res.IsTruncated
			nextToken = res.NextContinuationToken

			if len(res.Contents) > 0 {
				for _, c := range res.Contents {
					opts := backend.FindMatch{
						Key:      c.Key,
						Modified: c.LastModified,
					}
					f(opts)
				}
			}
		}
	}

	return
}

// Read implements backend.Reader
func (rw *readerWriter) Read(ctx context.Context, name string, keypath backend.KeyPath, _ *backend.CacheInfo) (io.ReadCloser, int64, error) {
	span, derivedCtx := opentracing.StartSpanFromContext(ctx, "s3.Read")
	defer span.Finish()

	keypath = backend.KeyPathWithPrefix(keypath, rw.cfg.Prefix)
	b, err := rw.readAll(derivedCtx, backend.ObjectFileName(keypath, name))
	if err != nil {
		return nil, 0, readError(err)
	}

	return io.NopCloser(bytes.NewReader(b)), int64(len(b)), err
}

// ReadRange implements backend.Reader
func (rw *readerWriter) ReadRange(ctx context.Context, name string, keypath backend.KeyPath, offset uint64, buffer []byte, _ *backend.CacheInfo) error {
	span, derivedCtx := opentracing.StartSpanFromContext(ctx, "s3.ReadRange", opentracing.Tags{
		"len":    len(buffer),
		"offset": offset,
	})
	defer span.Finish()

	keypath = backend.KeyPathWithPrefix(keypath, rw.cfg.Prefix)
	return readError(rw.readRange(derivedCtx, backend.ObjectFileName(keypath, name), int64(offset), buffer))
}

// Shutdown implements backend.Reader
func (rw *readerWriter) Shutdown() {
}

func (rw *readerWriter) WriteVersioned(ctx context.Context, name string, keypath backend.KeyPath, data io.Reader, version backend.Version) (backend.Version, error) {
	// Note there is a potential data race here because S3 does not support conditional headers. If
	// another process writes to the same object in between ReadVersioned and Write its changes will
	// be overwritten.
	// TODO use rw.hedgedCore.GetObject, don't download the full object
	_, currentVersion, err := rw.ReadVersioned(ctx, name, keypath)
	if err != nil && !errors.Is(err, backend.ErrDoesNotExist) {
		return "", err
	}

	level.Info(rw.logger).Log("msg", "WriteVersioned - fetching data", "currentVersion", currentVersion, "err", err, "version", version)

	// object does not exist - supplied version must be "0"
	if errors.Is(err, backend.ErrDoesNotExist) && version != backend.VersionNew {
		return "", backend.ErrVersionDoesNotMatch
	}
	if !errors.Is(err, backend.ErrDoesNotExist) && version != currentVersion {
		return "", backend.ErrVersionDoesNotMatch
	}

	// TODO extract Write to a separate method which returns minio.UploadInfo, saves us a GetObject request
	err = rw.Write(ctx, name, keypath, data, -1, nil)
	if err != nil {
		return "", err
	}

	_, currentVersion, err = rw.ReadVersioned(ctx, name, keypath)
	return currentVersion, err
}

func (rw *readerWriter) DeleteVersioned(ctx context.Context, name string, keypath backend.KeyPath, version backend.Version) error {
	// Note there is a potential data race here because S3 does not support conditional headers. If
	// another process writes to the same object in between ReadVersioned and Delete its changes will
	// be overwritten.
	// TODO use rw.hedgedCore.GetObject, don't download the full object
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
	span, derivedCtx := opentracing.StartSpanFromContext(ctx, "s3.ReadVersioned")
	defer span.Finish()

	keypath = backend.KeyPathWithPrefix(keypath, rw.cfg.Prefix)
	b, objectInfo, err := rw.readAllWithObjInfo(derivedCtx, backend.ObjectFileName(keypath, name))
	if err != nil {
		return nil, "", readError(err)
	}

	return io.NopCloser(bytes.NewReader(b)), backend.Version(objectInfo.ETag), nil
}

func (rw *readerWriter) readAll(ctx context.Context, name string) ([]byte, error) {
	reader, info, _, err := rw.hedgedCore.GetObject(ctx, rw.cfg.Bucket, name, minio.GetObjectOptions{})
	if err != nil {
		// do not change or wrap this error
		// we need to compare the specific err message
		return nil, err
	}
	defer reader.Close()

	return tempo_io.ReadAllWithEstimate(reader, info.Size)
}

func (rw *readerWriter) readAllWithObjInfo(ctx context.Context, name string) ([]byte, minio.ObjectInfo, error) {
	reader, info, _, err := rw.hedgedCore.GetObject(ctx, rw.cfg.Bucket, name, minio.GetObjectOptions{})
	if err != nil && minio.ToErrorResponse(err).Code == s3.ErrCodeNoSuchKey {
		return nil, minio.ObjectInfo{}, backend.ErrDoesNotExist
	} else if err != nil {
		return nil, minio.ObjectInfo{}, fmt.Errorf("error fetching object from s3 backend: %w", err)
	}
	defer reader.Close()

	buf, err := tempo_io.ReadAllWithEstimate(reader, info.Size)
	if err != nil {
		return nil, minio.ObjectInfo{}, fmt.Errorf("error reading response from s3 backend: %w", err)
	}
	return buf, info, nil
}

func (rw *readerWriter) readRange(ctx context.Context, objName string, offset int64, buffer []byte) error {
	options := minio.GetObjectOptions{}
	err := options.SetRange(offset, offset+int64(len(buffer)))
	if err != nil {
		return fmt.Errorf("error setting headers for range read in s3: %w", err)
	}
	reader, _, _, err := rw.hedgedCore.GetObject(ctx, rw.cfg.Bucket, objName, options)
	if err != nil {
		return fmt.Errorf("error in range read from s3 backend, bucket: %s, objName: %s: %w", rw.cfg.Bucket, objName, err)
	}
	defer reader.Close()

	/* bytes read == len(buffer) if and only if err == nil */
	_, err = io.ReadFull(reader, buffer)

	if err == nil {
		/* read EOF so connection can be reused */
		var dummy [1]byte
		_, _ = reader.Read(dummy[:])
		return nil
	}

	return fmt.Errorf("error in range read from s3 backend: %w", err)
}

func fetchCreds(cfg *Config) (*credentials.Credentials, error) {
	wrapCredentialsProvider := func(p credentials.Provider) credentials.Provider {
		if cfg.SignatureV2 {
			return &overrideSignatureVersion{useV2: cfg.SignatureV2, upstream: p}
		}
		return p
	}

	chain := []credentials.Provider{
		wrapCredentialsProvider(&credentials.Static{
			Value: credentials.Value{
				AccessKeyID:     cfg.AccessKey,
				SecretAccessKey: cfg.SecretKey.String(),
				SessionToken:    cfg.SessionToken.String(),
			},
		}),
		wrapCredentialsProvider(&credentials.EnvAWS{}),
		wrapCredentialsProvider(&credentials.EnvMinio{}),
		wrapCredentialsProvider(&credentials.FileAWSCredentials{}),
		wrapCredentialsProvider(&credentials.FileMinioClient{}),
		wrapCredentialsProvider(&credentials.IAM{
			Client: &http.Client{
				Transport: http.DefaultTransport,
			},
			Endpoint: os.Getenv("TEST_IAM_ENDPOINT"),
		}),
	}

	creds := credentials.NewChainCredentials(chain)

	// error early if we cannot obtain credentials
	if _, err := creds.Get(); err != nil {
		return nil, fmt.Errorf("failed to get credentials: %w", err)
	}

	return creds, nil
}

func createCore(cfg *Config, hedge bool) (*minio.Core, error) {
	creds, err := fetchCreds(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch credentials: %w", err)
	}

	customTransport, err := minio.DefaultTransport(!cfg.Insecure)
	if err != nil {
		return nil, fmt.Errorf("create minio.DefaultTransport: %w", err)
	}

	/* minio sets MaxIdleConns to 100 but we should also increase per host to 100 */
	customTransport.MaxIdleConnsPerHost = 100
	customTransport.MaxIdleConns = 100

	tlsConfig, err := cfg.GetTLSConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create TLS config: %w", err)
	}

	if tlsConfig != nil {
		customTransport.TLSClientConfig = tlsConfig
	}

	// add instrumentation
	transport := instrumentation.NewTransport(customTransport)
	var stats *hedgedhttp.Stats
	if hedge && cfg.HedgeRequestsAt != 0 {
		transport, stats, err = hedgedhttp.NewRoundTripperAndStats(cfg.HedgeRequestsAt, cfg.HedgeRequestsUpTo, transport)
		if err != nil {
			return nil, err
		}
		instrumentation.PublishHedgedMetrics(stats)
	}

	opts := &minio.Options{
		Region:    cfg.Region,
		Secure:    !cfg.Insecure,
		Creds:     creds,
		Transport: transport,
	}

	if cfg.ForcePathStyle {
		opts.BucketLookup = minio.BucketLookupPath
	} else {
		opts.BucketLookup = minio.BucketLookupType(cfg.BucketLookupType)
	}

	core, err := minio.NewCore(cfg.Endpoint, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	core.SetS3EnableDualstack(cfg.UseDualStack)
	return core, err
}

func readError(err error) error {
	if err != nil && minio.ToErrorResponse(err).Code == s3.ErrCodeNoSuchKey {
		return backend.ErrDoesNotExist
	}
	return err
}
