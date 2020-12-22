package s3

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	log_util "github.com/cortexproject/cortex/pkg/util"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/google/uuid"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/util"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/pkg/errors"
)

const (
	s3KeyDoesNotExist = "The specified key does not exist."
)

// readerWriter can read/write from an s3 backend
type readerWriter struct {
	logger log.Logger
	cfg    *Config
	core   *minio.Core
}

type overrideSignatureVersion struct {
	useV2    bool
	upstream credentials.Provider
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

func New(cfg *Config) (backend.Reader, backend.Writer, backend.Compactor, error) {
	l := log_util.Logger

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
				SecretAccessKey: cfg.SecretKey,
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
	opts := &minio.Options{
		Secure: !cfg.Insecure,
		Creds:  creds,
	}
	core, err := minio.NewCore(cfg.Endpoint, opts)
	if err != nil {
		return nil, nil, nil, err
	}

	// TODO: add custom transport with instrumentation.
	//client.SetCustomTransport(minio.DefaultTransport(!cfg.Insecure))
	exists, err := core.BucketExists(context.Background(), cfg.Bucket)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("unexpected error from BucketExists on %s: %w", cfg.Bucket, err)
	}

	if !exists {
		return nil, nil, nil, fmt.Errorf("s3 Bucket %s does not exist", cfg.Bucket)
	}

	// try listing objects
	_, err = core.ListObjects(cfg.Bucket, "", "", "/", 0)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("unexpected error from ListObjects on %s: %w", cfg.Bucket, err)
	}

	rw := &readerWriter{
		logger: l,
		cfg:    cfg,
		core:   core,
	}
	return rw, rw, rw, nil
}

// Write implements backend.Writer
func (rw *readerWriter) Write(ctx context.Context, meta *backend.BlockMeta, bBloom [][]byte, bIndex []byte, objectFilePath string) error {
	if err := util.FileExists(objectFilePath); err != nil {
		return err
	}

	objName := util.ObjectFileName(meta.BlockID, meta.TenantID)
	info, err := rw.core.FPutObject(
		ctx,
		rw.cfg.Bucket,
		objName,
		objectFilePath,
		minio.PutObjectOptions{PartSize: rw.cfg.PartSize},
	)
	if err != nil {
		return errors.Wrapf(err, "error writing object to s3 backend, object %s", objName)
	}

	level.Debug(rw.logger).Log("msg", "object uploaded to s3", "objectName", objName, "size", info.Size)

	err = rw.WriteBlockMeta(ctx, nil, meta, bBloom, bIndex)
	if err != nil {
		return err
	}

	return nil
}

// WriteBlockMeta implements backend.Writer
func (rw *readerWriter) WriteBlockMeta(ctx context.Context, tracker backend.AppendTracker, meta *backend.BlockMeta, bBloom [][]byte, bIndex []byte) error {
	if tracker != nil {
		a := tracker.(AppenderTracker)
		completeParts := make([]minio.CompletePart, 0)
		for _, p := range a.parts {
			completeParts = append(completeParts, minio.CompletePart{
				PartNumber: p.PartNumber,
				ETag:       p.ETag,
			})
		}
		level.Debug(rw.logger).Log("msg", "marking compacted block complete", "parts", len(completeParts))
		objName := util.ObjectFileName(meta.BlockID, meta.TenantID)
		etag, err := rw.core.CompleteMultipartUpload(
			ctx,
			rw.cfg.Bucket,
			objName,
			a.uploadID,
			completeParts,
		)
		if err != nil {
			return errors.Wrapf(err, "error completing multipart upload, object: %s, obj etag: %s", objName, etag)
		}
	}

	blockID := meta.BlockID
	tenantID := meta.TenantID
	options := minio.PutObjectOptions{
		PartSize: rw.cfg.PartSize,
	}

	for i, b := range bBloom {
		info, err := rw.core.Client.PutObject(
			ctx,
			rw.cfg.Bucket,
			util.BloomFileName(blockID, tenantID, i),
			bytes.NewReader(b),
			int64(len(b)),
			options,
		)
		if err != nil {
			return errors.Wrap(err, "error uploading bloom filter to s3")
		}
		level.Debug(rw.logger).Log("msg", "block bloom uploaded to s3", "shard", i, "size", info.Size)
	}

	info, err := rw.core.Client.PutObject(
		ctx,
		rw.cfg.Bucket,
		util.IndexFileName(blockID, tenantID),
		bytes.NewReader(bIndex),
		int64(len(bIndex)),
		options,
	)
	if err != nil {
		return errors.Wrap(err, "error uploading index to s3")
	}
	level.Debug(rw.logger).Log("msg", "block index uploaded to s3", "size", info.Size)

	bMeta, err := json.Marshal(meta)
	if err != nil {
		return errors.Wrap(err, "error unmarshalling block meta json")
	}

	// write meta last.  this will prevent blocklist from returning a partial block
	info, err = rw.core.Client.PutObject(
		ctx,
		rw.cfg.Bucket,
		util.MetaFileName(blockID, tenantID),
		bytes.NewReader(bMeta),
		int64(len(bMeta)),
		options,
	)
	if err != nil {
		return errors.Wrap(err, "error uploading block meta to s3")
	}
	level.Debug(rw.logger).Log("msg", "block meta uploaded to s3", "size", info.Size)

	return nil
}

type AppenderTracker struct {
	uploadID string
	partNum  int
	parts    []minio.ObjectPart
}

// AppendObject implements backend.Writer
func (rw *readerWriter) AppendObject(ctx context.Context, tracker backend.AppendTracker, meta *backend.BlockMeta, bObject []byte) (backend.AppendTracker, error) {
	var a AppenderTracker
	options := minio.PutObjectOptions{
		PartSize: rw.cfg.PartSize,
	}
	if tracker != nil {
		a = tracker.(AppenderTracker)
	} else {
		id, err := rw.core.NewMultipartUpload(
			ctx,
			rw.cfg.Bucket,
			util.ObjectFileName(meta.BlockID, meta.TenantID),
			options,
		)
		if err != nil {
			return nil, err
		}
		a.uploadID = id
	}

	level.Debug(rw.logger).Log("msg", "appending object to s3", "objectName", util.ObjectFileName(meta.BlockID, meta.TenantID))

	a.partNum++
	objPart, err := rw.core.PutObjectPart(
		ctx,
		rw.cfg.Bucket,
		util.ObjectFileName(meta.BlockID, meta.TenantID),
		a.uploadID,
		a.partNum,
		bytes.NewReader(bObject),
		int64(len(bObject)),
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

// Tenants implements backend.Reader
func (rw *readerWriter) Tenants(ctx context.Context) ([]string, error) {
	// ListObjects(bucket, prefix, marker, delimiter string, maxKeys int)
	res, err := rw.core.ListObjects(rw.cfg.Bucket, "", "", "/", 0)
	if err != nil {
		return nil, errors.Wrapf(err, "error listing tenants in bucket %s", rw.cfg.Bucket)
	}

	level.Debug(rw.logger).Log("msg", "listing tenants", "found", len(res.CommonPrefixes))
	var tenants []string
	for _, cp := range res.CommonPrefixes {
		tenants = append(tenants, strings.Split(cp.Prefix, "/")[0])
	}
	return tenants, nil
}

// Blocks implements backend.Reader
func (rw *readerWriter) Blocks(ctx context.Context, tenantID string) ([]uuid.UUID, error) {
	// ListObjects(bucket, prefix, marker, delimiter string, maxKeys int)
	res, err := rw.core.ListObjects(rw.cfg.Bucket, tenantID+"/", "", "/", 0)
	if err != nil {
		return nil, errors.Wrapf(err, "error listing blocks in s3 bucket, bucket: %s", rw.cfg.Bucket)
	}

	level.Debug(rw.logger).Log("msg", "listing blocks", "tenantID", tenantID, "found", len(res.CommonPrefixes))
	var blockIDs []uuid.UUID
	for _, cp := range res.CommonPrefixes {
		blockID, err := uuid.Parse(strings.Split(strings.TrimPrefix(cp.Prefix, res.Prefix), "/")[0])
		if err != nil {
			return nil, errors.Wrapf(err, "error parsing uuid of obj, objectName: %s", cp.Prefix)
		}
		blockIDs = append(blockIDs, blockID)
	}
	return blockIDs, nil
}

// BlockMeta implements backend.Reader
func (rw *readerWriter) BlockMeta(ctx context.Context, blockID uuid.UUID, tenantID string) (*backend.BlockMeta, error) {
	blockMetaFileName := util.MetaFileName(blockID, tenantID)
	body, err := rw.readAll(ctx, blockMetaFileName)
	if err != nil && err.Error() == s3KeyDoesNotExist {
		return nil, backend.ErrMetaDoesNotExist
	}
	out := &backend.BlockMeta{}
	err = json.Unmarshal(body, out)
	if err != nil {
		return nil, err
	}
	level.Debug(rw.logger).Log("msg", "fetched block meta", "tenantID", out.TenantID, "blockID", out.BlockID.String())
	return out, nil
}

// Bloom implements backend.Reader
func (rw *readerWriter) Bloom(ctx context.Context, blockID uuid.UUID, tenantID string, bloomShard int) ([]byte, error) {
	bloomFileName := util.BloomFileName(blockID, tenantID, bloomShard)
	return rw.readAll(ctx, bloomFileName)
}

// Index implements backend.Reader
func (rw *readerWriter) Index(ctx context.Context, blockID uuid.UUID, tenantID string) ([]byte, error) {
	indexFileName := util.IndexFileName(blockID, tenantID)
	return rw.readAll(ctx, indexFileName)
}

// Object implements backend.Reader
func (rw *readerWriter) Object(ctx context.Context, blockID uuid.UUID, tenantID string, start uint64, buffer []byte) error {
	objFileName := util.ObjectFileName(blockID, tenantID)
	return rw.readRange(ctx, objFileName, int64(start), buffer)
}

// Shutdown implements backend.Reader
func (rw *readerWriter) Shutdown() {
}

func (rw *readerWriter) readAll(ctx context.Context, name string) ([]byte, error) {
	reader, _, _, err := rw.core.GetObject(ctx, rw.cfg.Bucket, name, minio.GetObjectOptions{})
	if err != nil {
		// do not change or wrap this error
		// we need to compare the specific err message
		return nil, err
	}
	defer reader.Close()

	body, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func (rw *readerWriter) readAllWithObjInfo(ctx context.Context, name string) ([]byte, minio.ObjectInfo, error) {
	reader, info, _, err := rw.core.GetObject(ctx, rw.cfg.Bucket, name, minio.GetObjectOptions{})
	if err != nil && err.Error() == s3KeyDoesNotExist {
		return nil, minio.ObjectInfo{}, backend.ErrMetaDoesNotExist
	} else if err != nil {
		return nil, minio.ObjectInfo{}, errors.Wrap(err, "error fetching object from s3 backend")
	}
	defer reader.Close()

	body, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, minio.ObjectInfo{}, errors.Wrap(err, "error reading response from s3 backend")
	}
	return body, info, nil
}

func (rw *readerWriter) readRange(ctx context.Context, objName string, offset int64, buffer []byte) error {
	options := minio.GetObjectOptions{}
	err := options.SetRange(offset, offset+int64(len(buffer)))
	if err != nil {
		return errors.Wrap(err, "error setting headers for range read in s3")
	}
	reader, _, _, err := rw.core.GetObject(ctx, rw.cfg.Bucket, objName, options)
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
