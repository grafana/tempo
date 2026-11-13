// Package s3express is the storage backend for S3 directory buckets, in an Availability Zone (S3
// Express One Zone) or a Local Zone. It wraps the s3 backend and only replaces what directory
// buckets do differently: requests are authenticated with S3 Express sessions, and objects can
// only be listed with ListObjectsV2, without start-after, and aren't listed in order.
// https://docs.aws.amazon.com/AmazonS3/latest/userguide/s3-express-differences.html
package s3express

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"

	gkLog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	minio "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"go.opentelemetry.io/otel"

	"github.com/grafana/tempo/v3/pkg/util/log"
	"github.com/grafana/tempo/v3/tempodb/backend"
	"github.com/grafana/tempo/v3/tempodb/backend/s3"
)

// readerWriter can read/write from an S3 directory bucket. The s3 backend it wraps sends the
// requests, through a transport that signs them with S3 Express sessions. Every method is
// written out, so that a method added to the backend interfaces has to be implemented here
// instead of being inherited from the s3 backend.
type readerWriter struct {
	s3     *s3.Backend
	cfg    *Config
	logger gkLog.Logger
}

var tracer = otel.Tracer("tempodb/backend/s3express")

var (
	_ backend.RawReader             = (*readerWriter)(nil)
	_ backend.RawWriter             = (*readerWriter)(nil)
	_ backend.Compactor             = (*readerWriter)(nil)
	_ backend.VersionedReaderWriter = (*readerWriter)(nil)
)

// NewNoConfirm gets the s3express backend without testing it
func NewNoConfirm(cfg *Config) (backend.RawReader, backend.RawWriter, backend.Compactor, error) {
	rw, err := internalNew(cfg, false, nil)
	if err != nil {
		return nil, nil, nil, err
	}
	return rw, rw, rw, nil
}

// New gets the s3express backend
func New(cfg *Config) (backend.RawReader, backend.RawWriter, backend.Compactor, error) {
	rw, err := internalNew(cfg, true, nil)
	if err != nil {
		return nil, nil, nil, err
	}
	return rw, rw, rw, nil
}

// NewVersionedReaderWriter creates a client to perform versioned requests. Like with the s3
// backend, write requests are best-effort.
func NewVersionedReaderWriter(cfg *Config) (backend.VersionedReaderWriter, error) {
	rw, err := internalNew(cfg, true, nil)
	if err != nil {
		return nil, err
	}
	return rw, nil
}

// internalNew returns the backend for cfg. When next is set, signed requests are sent with it
// instead of the transport of the s3 backend, for tests.
func internalNew(cfg *Config, confirm bool, next http.RoundTripper) (*readerWriter, error) {
	if cfg == nil {
		return nil, errors.New("config is nil")
	}
	if _, err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid s3express config: %w", err)
	}

	s3Cfg := cfg.Config
	endpoint, lookup, bucketURL := bucketEndpoint(cfg)
	s3Cfg.Endpoint, s3Cfg.BucketLookupType = endpoint, int(lookup)

	b, err := s3.NewWithTransport(&s3Cfg, func(base http.RoundTripper, creds *credentials.Credentials) http.RoundTripper {
		if next != nil {
			base = next
		}
		return newTransport(base, creds, cfg.Region, bucketURL)
	})
	if err != nil {
		return nil, err
	}

	// like the s3 backend, check the bucket can be listed
	if confirm {
		_, err = b.Core().ListObjectsV2(cfg.Bucket, listPrefix(backend.KeyPathWithPrefix(nil, cfg.Prefix)), "", "", "/", 1)
		if err != nil {
			return nil, fmt.Errorf("unexpected error from ListObjectsV2 on %s: %w", cfg.Bucket, err)
		}
	}

	return &readerWriter{
		s3:     b,
		cfg:    cfg,
		logger: log.Logger,
	}, nil
}

// bucketEndpoint returns the endpoint and the bucket lookup type to reach the bucket of cfg with,
// and the URL of the bucket. The endpoint defaults to the bucket's Zonal endpoint. Requests to
// amazonaws.com endpoints are virtual-hosted style, the only style directory buckets accept,
// unless forcepathstyle or bucket_lookup_type ask for path-style requests.
func bucketEndpoint(cfg *Config) (string, minio.BucketLookupType, *url.URL) {
	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = zonalEndpoint(cfg.Bucket, cfg.Region)
	}

	lookup := minio.BucketLookupType(cfg.BucketLookupType)
	if cfg.ForcePathStyle {
		lookup = minio.BucketLookupPath
	}
	if lookup == minio.BucketLookupAuto {
		lookup = minio.BucketLookupPath
		if strings.HasSuffix((&url.URL{Host: endpoint}).Hostname(), ".amazonaws.com") {
			lookup = minio.BucketLookupDNS
		}
	}

	bucketURL := &url.URL{Scheme: "https", Host: endpoint, Path: "/" + cfg.Bucket + "/"}
	if lookup == minio.BucketLookupDNS {
		bucketURL.Host, bucketURL.Path = cfg.Bucket+"."+endpoint, "/"
	}
	return endpoint, lookup, bucketURL
}

// List implements backend.Reader
func (rw *readerWriter) List(ctx context.Context, keypath backend.KeyPath) ([]string, error) {
	keypath = backend.KeyPathWithPrefix(keypath, rw.cfg.Prefix)
	prefix := listPrefix(keypath)

	var objects []string
	err := rw.listObjects(ctx, prefix, "/", func(res minio.ListBucketV2Result) {
		for _, cp := range res.CommonPrefixes {
			objects = append(objects, strings.Split(strings.TrimPrefix(cp.Prefix, prefix), "/")[0])
		}
	})
	if err != nil {
		return nil, fmt.Errorf("error listing blocks in s3express bucket, bucket: %s: %w", rw.cfg.Bucket, err)
	}

	return objects, nil
}

// ListBlocks implements backend.Reader. The s3 backend lists ranges of block IDs concurrently,
// starting each listing after the lowest block ID of its range and relying on keys being listed in
// order. Directory buckets support neither, so their blocks are listed with a single listing.
func (rw *readerWriter) ListBlocks(ctx context.Context, tenant string) ([]uuid.UUID, []uuid.UUID, []uuid.UUID, error) {
	ctx, span := tracer.Start(ctx, "readerWriter.ListBlocks")
	defer span.End()

	var (
		prefix            = listPrefix(backend.KeyPathWithPrefix(backend.KeyPath{tenant}, rw.cfg.Prefix))
		blockIDs          = make([]uuid.UUID, 0, 1000)
		compactedBlockIDs = make([]uuid.UUID, 0, 1000)
		noCompactBlockIDs = make([]uuid.UUID, 0)
	)
	err := rw.listObjects(ctx, prefix, "", func(res minio.ListBucketV2Result) {
		for _, c := range res.Contents {
			// i.e: <blockID>/meta.json
			blockID, name, _ := strings.Cut(strings.TrimPrefix(c.Key, prefix), "/")

			var ids *[]uuid.UUID
			switch name {
			case backend.MetaName:
				ids = &blockIDs
			case backend.CompactedMetaName:
				ids = &compactedBlockIDs
			case backend.NoCompactFileName:
				ids = &noCompactBlockIDs
			default:
				continue
			}

			if id, err := uuid.Parse(blockID); err == nil {
				*ids = append(*ids, id)
			}
		}
	})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error finding objects in s3express bucket, bucket: %s: %w", rw.cfg.Bucket, err)
	}

	level.Debug(rw.logger).Log("msg", "listing blocks complete", "blockIDs", len(blockIDs), "compactedBlockIDs", len(compactedBlockIDs))

	return blockIDs, compactedBlockIDs, noCompactBlockIDs, nil
}

// ClearBlock implements backend.Compactor
func (rw *readerWriter) ClearBlock(blockID uuid.UUID, tenantID string) error {
	if len(tenantID) == 0 {
		return backend.ErrEmptyTenantID
	}
	if blockID == uuid.Nil {
		return backend.ErrEmptyBlockID
	}

	blockPath := backend.RootPath(blockID, tenantID, rw.cfg.Prefix) + "/"
	level.Debug(rw.logger).Log("msg", "deleting block", "block path", blockPath)

	// list every object before deleting any, so the deletes can't affect the listing
	var keys []string
	err := rw.listObjects(context.TODO(), blockPath, "/", func(res minio.ListBucketV2Result) {
		for _, obj := range res.Contents {
			keys = append(keys, obj.Key)
		}
	})
	if err != nil {
		return fmt.Errorf("error listing objects in bucket %s: %w", rw.cfg.Bucket, err)
	}

	level.Debug(rw.logger).Log("msg", "listing objects", "found", len(keys))
	for _, key := range keys {
		err = rw.s3.Core().RemoveObject(context.TODO(), rw.cfg.Bucket, key, minio.RemoveObjectOptions{})
		if err != nil {
			return fmt.Errorf("error deleting obj from s3express: %s: %w", key, err)
		}
	}

	return nil
}

// The methods below are the operations directory buckets support like other buckets, which the
// s3 backend implements.

// Find implements backend.Reader. The s3 backend lists with ListObjectsV2, without start-after.
func (rw *readerWriter) Find(ctx context.Context, keypath backend.KeyPath, f backend.FindFunc) error {
	return rw.s3.Find(ctx, keypath, f)
}

// Read implements backend.Reader
func (rw *readerWriter) Read(ctx context.Context, name string, keypath backend.KeyPath, cacheInfo *backend.CacheInfo) (io.ReadCloser, int64, error) {
	return rw.s3.Read(ctx, name, keypath, cacheInfo)
}

// ReadRange implements backend.Reader
func (rw *readerWriter) ReadRange(ctx context.Context, name string, keypath backend.KeyPath, offset uint64, buffer []byte, cacheInfo *backend.CacheInfo) error {
	return rw.s3.ReadRange(ctx, name, keypath, offset, buffer, cacheInfo)
}

// Shutdown implements backend.Reader
func (rw *readerWriter) Shutdown() {
	rw.s3.Shutdown()
}

// Write implements backend.Writer
func (rw *readerWriter) Write(ctx context.Context, name string, keypath backend.KeyPath, data io.Reader, size int64, cacheInfo *backend.CacheInfo) error {
	return rw.s3.Write(ctx, name, keypath, data, size, cacheInfo)
}

// Append implements backend.Writer. The s3 backend numbers parts consecutively from 1, as
// directory buckets require.
func (rw *readerWriter) Append(ctx context.Context, name string, keypath backend.KeyPath, tracker backend.AppendTracker, buffer []byte) (backend.AppendTracker, error) {
	return rw.s3.Append(ctx, name, keypath, tracker, buffer)
}

// CloseAppend implements backend.Writer
func (rw *readerWriter) CloseAppend(ctx context.Context, tracker backend.AppendTracker) error {
	return rw.s3.CloseAppend(ctx, tracker)
}

// Delete implements backend.Writer
func (rw *readerWriter) Delete(ctx context.Context, name string, keypath backend.KeyPath, cacheInfo *backend.CacheInfo) error {
	return rw.s3.Delete(ctx, name, keypath, cacheInfo)
}

// MarkBlockCompacted implements backend.Compactor. The transport signs its CopyObject request
// with the IAM credentials, as directory buckets require.
func (rw *readerWriter) MarkBlockCompacted(blockID uuid.UUID, tenantID string) error {
	return rw.s3.MarkBlockCompacted(blockID, tenantID)
}

// CompactedBlockMeta implements backend.Compactor
func (rw *readerWriter) CompactedBlockMeta(blockID uuid.UUID, tenantID string) (*backend.CompactedBlockMeta, error) {
	return rw.s3.CompactedBlockMeta(blockID, tenantID)
}

// WriteVersioned implements backend.VersionedReaderWriter
func (rw *readerWriter) WriteVersioned(ctx context.Context, name string, keypath backend.KeyPath, data io.Reader, size int64, version backend.Version) (backend.Version, error) {
	return rw.s3.WriteVersioned(ctx, name, keypath, data, size, version)
}

// DeleteVersioned implements backend.VersionedReaderWriter
func (rw *readerWriter) DeleteVersioned(ctx context.Context, name string, keypath backend.KeyPath, version backend.Version) error {
	return rw.s3.DeleteVersioned(ctx, name, keypath, version)
}

// ReadVersioned implements backend.VersionedReaderWriter
func (rw *readerWriter) ReadVersioned(ctx context.Context, name string, keypath backend.KeyPath) (io.ReadCloser, backend.Version, error) {
	return rw.s3.ReadVersioned(ctx, name, keypath)
}

// listObjects lists the objects under prefix with ListObjectsV2, the only listing directory
// buckets support, calling f with each page.
func (rw *readerWriter) listObjects(ctx context.Context, prefix, delimiter string, f func(minio.ListBucketV2Result)) error {
	token := ""
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		// ListObjectsV2(bucket, prefix, startAfter, continuationToken, delimiter string, maxKeys int)
		res, err := rw.s3.Core().ListObjectsV2(rw.cfg.Bucket, prefix, "", token, delimiter, 0)
		if err != nil {
			return err
		}
		f(res)

		if !res.IsTruncated || res.NextContinuationToken == "" {
			return nil
		}
		token = res.NextContinuationToken
	}
}

// listPrefix returns the prefix to list the objects under keypath with. Directory buckets only
// accept list prefixes that end in a delimiter, so a non-empty prefix always ends in "/".
func listPrefix(keypath backend.KeyPath) string {
	prefix := path.Join(keypath...)
	if len(prefix) > 0 {
		prefix += "/"
	}
	return prefix
}
