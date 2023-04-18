package s3

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"

	"github.com/grafana/tempo/tempodb/backend/instrumentation"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/cristalhq/hedgedhttp"
	gkLog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/grafana/tempo/tempodb/backend"
	minio "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"

	tempo_io "github.com/grafana/tempo/pkg/io"
	"github.com/grafana/tempo/pkg/util/log"
)

// readerWriter can read/write from an s3 backend
type readerWriter struct {
	logger     gkLog.Logger
	cfg        *Config
	core       *minio.Core
	hedgedCore *minio.Core
}

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
	return internalNew(cfg, false)
}

// New gets the S3 backend
func New(cfg *Config) (backend.RawReader, backend.RawWriter, backend.Compactor, error) {
	return internalNew(cfg, true)
}

func internalNew(cfg *Config, confirm bool) (backend.RawReader, backend.RawWriter, backend.Compactor, error) {
	l := log.Logger

	core, err := createCore(cfg, false)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("unexpected error creating core: %w", err)
	}

	hedgedCore, err := createCore(cfg, true)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("unexpected error creating hedgedCore: %w", err)
	}

	// try listing objects
	if confirm {
		_, err = core.ListObjects(cfg.Bucket, "", "", "/", 0)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("unexpected error from ListObjects on %s: %w", cfg.Bucket, err)
		}
	}

	rw := &readerWriter{
		logger:     l,
		cfg:        cfg,
		core:       core,
		hedgedCore: hedgedCore,
	}
	return rw, rw, rw, nil
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
func (rw *readerWriter) Write(ctx context.Context, name string, keypath backend.KeyPath, data io.Reader, size int64, _ bool) error {
	span, derivedCtx := opentracing.StartSpanFromContext(ctx, "s3.Write")
	defer span.Finish()

	span.SetTag("object", name)

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
		return errors.Wrapf(err, "error writing object to s3 backend, object %s", objName)
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
		"",
		"",
		nil,
	)
	if err != nil {
		return a, errors.Wrap(err, "error in multipart upload")
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

	etag, err := rw.core.CompleteMultipartUpload(
		ctx,
		rw.cfg.Bucket,
		a.objectName,
		a.uploadID,
		completeParts,
		minio.PutObjectOptions{},
	)
	if err != nil {
		return errors.Wrapf(err, "error completing multipart upload, object: %s, obj etag: %s", a.objectName, etag)
	}

	return nil
}

// List implements backend.Reader
func (rw *readerWriter) List(ctx context.Context, keypath backend.KeyPath) ([]string, error) {
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
			return nil, errors.Wrapf(err, "error listing blocks in s3 bucket, bucket: %s", rw.cfg.Bucket)
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

// Read implements backend.Reader
func (rw *readerWriter) Read(ctx context.Context, name string, keypath backend.KeyPath, _ bool) (io.ReadCloser, int64, error) {
	span, derivedCtx := opentracing.StartSpanFromContext(ctx, "s3.Read")
	defer span.Finish()

	b, err := rw.readAll(derivedCtx, backend.ObjectFileName(keypath, name))
	if err != nil {
		return nil, 0, readError(err)
	}

	return io.NopCloser(bytes.NewReader(b)), int64(len(b)), err
}

// ReadRange implements backend.Reader
func (rw *readerWriter) ReadRange(ctx context.Context, name string, keypath backend.KeyPath, offset uint64, buffer []byte, _ bool) error {
	span, derivedCtx := opentracing.StartSpanFromContext(ctx, "s3.ReadRange", opentracing.Tags{
		"len":    len(buffer),
		"offset": offset,
	})
	defer span.Finish()

	return readError(rw.readRange(derivedCtx, backend.ObjectFileName(keypath, name), int64(offset), buffer))
}

// Shutdown implements backend.Reader
func (rw *readerWriter) Shutdown() {
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
		return nil, minio.ObjectInfo{}, errors.Wrap(err, "error fetching object from s3 backend")
	}
	defer reader.Close()

	buf, err := tempo_io.ReadAllWithEstimate(reader, info.Size)
	if err != nil {
		return nil, minio.ObjectInfo{}, errors.Wrap(err, "error reading response from s3 backend")
	}
	return buf, info, nil
}

func (rw *readerWriter) readRange(ctx context.Context, objName string, offset int64, buffer []byte) error {
	options := minio.GetObjectOptions{}
	err := options.SetRange(offset, offset+int64(len(buffer)))
	if err != nil {
		return errors.Wrap(err, "error setting headers for range read in s3")
	}
	reader, _, _, err := rw.hedgedCore.GetObject(ctx, rw.cfg.Bucket, objName, options)
	if err != nil {
		return errors.Wrapf(err, "error in range read from s3 backend, bucket: %s, objName: %s", rw.cfg.Bucket, objName)
	}
	defer reader.Close()

	totalBytes := 0
	for {
		byteCount, err := reader.Read(buffer[totalBytes:])
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return errors.Wrap(err, "error in range read from s3 backend")
		}
		if byteCount == 0 {
			return nil
		}
		totalBytes += byteCount
	}
}

func createCore(cfg *Config, hedge bool) (*minio.Core, error) {
	wrapCredentialsProvider := func(p credentials.Provider) credentials.Provider {
		if cfg.SignatureV2 {
			return &overrideSignatureVersion{useV2: cfg.SignatureV2, upstream: p}
		}
		return p
	}

	creds := credentials.NewChainCredentials([]credentials.Provider{
		wrapCredentialsProvider(&credentials.EnvAWS{}),
		wrapCredentialsProvider(&credentials.Static{
			Value: credentials.Value{
				AccessKeyID:     cfg.AccessKey,
				SecretAccessKey: cfg.SecretKey.String(),
				SessionToken:    cfg.SessionToken.String(),
			},
		}),
		wrapCredentialsProvider(&credentials.EnvMinio{}),
		wrapCredentialsProvider(&credentials.FileAWSCredentials{}),
		wrapCredentialsProvider(&credentials.FileMinioClient{}),
		wrapCredentialsProvider(&credentials.IAM{
			Client: &http.Client{
				Transport: http.DefaultTransport,
			},
		}),
	})

	customTransport, err := minio.DefaultTransport(!cfg.Insecure)
	if err != nil {
		return nil, errors.Wrap(err, "create minio.DefaultTransport")
	}

	if cfg.InsecureSkipVerify {
		customTransport.TLSClientConfig.InsecureSkipVerify = true
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

	return minio.NewCore(cfg.Endpoint, opts)
}

func readError(err error) error {
	if err != nil && minio.ToErrorResponse(err).Code == s3.ErrCodeNoSuchKey {
		return backend.ErrDoesNotExist
	}
	return err
}
