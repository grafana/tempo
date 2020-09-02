package s3

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"strings"

	"github.com/pkg/errors"
	log_util "github.com/cortexproject/cortex/pkg/util"
	"github.com/go-kit/kit/log"
	"github.com/google/uuid"
	"github.com/go-kit/kit/log/level"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/util"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/minio/minio-go/v6"
)

const (
	s3KeyDoesNotExist = "The specified key does not exist."
)

// readerWriter can read/write from an s3 backend
type readerWriter struct {
	logger log.Logger
	cfg *Config
	core *minio.Core
}

func New(cfg *Config) (backend.Reader, backend.Writer, backend.Compactor, error) {
	l := log_util.Logger
	core, err := minio.NewCore(cfg.Endpoint, cfg.AccessKey, cfg.SecretKey, !cfg.Insecure)
	if err != nil {
		return nil, nil, nil, err
	}

	// TODO: add custom transport with instrumentation.
	//client.SetCustomTransport(minio.DefaultTransport(!cfg.Insecure))

	// make bucket name if doesn't exist already
	err = core.MakeBucket(cfg.Bucket, cfg.Region)
	if err != nil {
		if exists, errBucketExists := core.BucketExists(cfg.Bucket); !exists || errBucketExists != nil {
			return nil, nil, nil, err
		}
	}

	rw := &readerWriter{
		logger: l,
		cfg: cfg,
		core: core,
	}
	return rw, rw, rw, nil
}

// Write implements backend.Writer
func (rw *readerWriter) Write(ctx context.Context, meta *encoding.BlockMeta, bBloom []byte, bIndex []byte, objectFilePath string) error {
	if err := util.FileExists(objectFilePath); err != nil {
		return err
	}

	objName := util.ObjectFileName(meta.BlockID, meta.TenantID)
	size, err := rw.core.FPutObjectWithContext(
		ctx,
		rw.cfg.Bucket,
		objName,
		objectFilePath,
		minio.PutObjectOptions{PartSize: rw.cfg.PartSize},
		)
	if err != nil {
		return errors.Wrapf(err, "error writing object to s3 backend, object %s", objName)
	}

	level.Debug(rw.logger).Log("msg", "object uploaded to s3", "objectName", objName, "size", size)

	err = rw.WriteBlockMeta(ctx, nil, meta, bBloom, bIndex)
	if err != nil {
		return err
	}

	return nil
}

// WriteBlockMeta implements backend.Writer
func (rw *readerWriter) WriteBlockMeta(ctx context.Context, tracker backend.AppendTracker, meta *encoding.BlockMeta, bBloom []byte, bIndex []byte) error {
	if tracker != nil {
		a := tracker.(AppenderTracker)
		completeParts := make([]minio.CompletePart, 0)
		for _, p := range a.parts {
			completeParts = append(completeParts, minio.CompletePart{
				PartNumber: p.PartNumber,
				ETag: p.ETag,
			})
		}
		objName := util.ObjectFileName(meta.BlockID, meta.TenantID)
		etag, err := rw.core.CompleteMultipartUploadWithContext(
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

	size, err := rw.core.Client.PutObjectWithContext(
		ctx,
		rw.cfg.Bucket,
		util.BloomFileName(blockID, tenantID),
		bytes.NewReader(bBloom),
		int64(len(bBloom)),
		options,
		)
	if err != nil {
		return err
	}
	level.Debug(rw.logger).Log("msg", "block bloom uploaded to s3", "size", size)

	size, err = rw.core.Client.PutObjectWithContext(
		ctx,
		rw.cfg.Bucket,
		util.IndexFileName(blockID, tenantID),
		bytes.NewReader(bIndex),
		int64(len(bIndex)),
		options,
	)
	if err != nil {
		return err
	}
	level.Debug(rw.logger).Log("msg", "block index uploaded to s3", "size", size)

	bMeta, err := json.Marshal(meta)
	if err != nil {
		return err
	}

	// write meta last.  this will prevent blocklist from returning a partial block
	size, err = rw.core.Client.PutObjectWithContext(
		ctx,
		rw.cfg.Bucket,
		util.MetaFileName(blockID, tenantID),
		bytes.NewReader(bMeta),
		int64(len(bMeta)),
		options,
		)
	if err != nil {
		return err
	}
	level.Debug(rw.logger).Log("msg", "block meta uploaded to s3", "size", size)

	return nil
}

type AppenderTracker struct {
	uploadID string
	partNum int
	parts []minio.ObjectPart
}

// AppendObject implements backend.Writer
func (rw *readerWriter) AppendObject(ctx context.Context, tracker backend.AppendTracker, meta *encoding.BlockMeta, bObject []byte) (backend.AppendTracker, error) {
	var a AppenderTracker
	options := minio.PutObjectOptions{
		PartSize: rw.cfg.PartSize,
	}
	if tracker != nil {
		a = tracker.(AppenderTracker)
	} else {
		id, err := rw.core.NewMultipartUpload(
			rw.cfg.Bucket,
			util.ObjectFileName(meta.BlockID, meta.TenantID),
			options,
			)
		if err != nil {
			return nil, err
		}
		a.uploadID = id
	}

	objPart, err := rw.core.PutObjectPartWithContext(
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
	a.partNum ++
	a.parts = append(a.parts, objPart)

	return a, nil
}

// Tenants implements backend.Reader
func (rw *readerWriter) Tenants() ([]string, error) {
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
func (rw *readerWriter) Blocks(tenantID string) ([]uuid.UUID, error) {
	// ListObjects(bucket, prefix, marker, delimiter string, maxKeys int)
	res, err := rw.core.ListObjects(rw.cfg.Bucket, tenantID+"/", "", "/", 0)
	if err != nil {
		return nil, errors.Wrapf(err, "error listing blocks in s3 bucket, bucket: %s", rw.cfg.Bucket)
	}

	level.Debug(rw.logger).Log("msg", "listing blocks", "tenantID", tenantID, "found", len(res.CommonPrefixes))
	var blockIDs []uuid.UUID
	for _, cp := range res.CommonPrefixes {
		blockID, err := uuid.Parse(strings.Split(strings.TrimPrefix(cp.Prefix, res.Prefix),"/")[0])
		if err != nil {
			return nil, errors.Wrapf(err, "error parsing uuid of obj, objectName: %s", cp.Prefix)
		}
		blockIDs = append(blockIDs, blockID)
	}
	return blockIDs, nil
}

// BlockMeta implements backend.Reader
func (rw *readerWriter) BlockMeta(blockID uuid.UUID, tenantID string) (*encoding.BlockMeta, error) {
	blockMetaFileName := util.MetaFileName(blockID, tenantID)
	body, err := rw.readAll(context.Background(), blockMetaFileName)
	if err != nil && err.Error() == s3KeyDoesNotExist {
		return nil, backend.ErrMetaDoesNotExist
	}
	out := &encoding.BlockMeta{}
	err = json.Unmarshal(body, out)
	if err != nil {
		return nil, err
	}
	level.Debug(rw.logger).Log("msg", "fetched block meta", "tenantID", out.TenantID, "blockID", out.BlockID.String())
	return out, nil
}

// Bloom implements backend.Reader
func (rw *readerWriter) Bloom(blockID uuid.UUID, tenantID string) ([]byte, error) {
	bloomFileName := util.BloomFileName(blockID, tenantID)
	return rw.readAll(context.Background(), bloomFileName)
}

// Index implements backend.Reader
func (rw *readerWriter) Index(blockID uuid.UUID, tenantID string) ([]byte, error) {
	indexFileName := util.IndexFileName(blockID, tenantID)
	return rw.readAll(context.Background(), indexFileName)
}

// Object implements backend.Reader
func (rw *readerWriter) Object(blockID uuid.UUID, tenantID string, start uint64, buffer []byte) error {
	objFileName := util.ObjectFileName(blockID, tenantID)
	return rw.readRange(context.Background(), objFileName, int64(start), buffer)
}

// Shutdown implements backend.Reader
func (rw *readerWriter) Shutdown() {
}

func (rw *readerWriter) readAll(ctx context.Context, name string) ([]byte, error) {
	reader, _, _, err := rw.core.GetObjectWithContext(ctx, rw.cfg.Bucket, name, minio.GetObjectOptions{})
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

func (rw *readerWriter) readAllWithObjInfo(ctx context.Context, name string) ([]byte, minio.ObjectInfo,error) {
	reader, info, _, err := rw.core.GetObjectWithContext(ctx, rw.cfg.Bucket, name, minio.GetObjectOptions{})
	if err != nil {
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
	err := options.SetRange(offset, int64(len(buffer)))
	if err != nil {
		return err
	}
	reader, _, _, err := rw.core.GetObjectWithContext(ctx, rw.cfg.Bucket, objName, options)
	if err != nil {
		return errors.Wrapf(err, "error in range read from s3 backend, bucket: %s, objName: %s", rw.cfg.Bucket, objName)
	}
	defer reader.Close()

	buffer, err = ioutil.ReadAll(reader)
	if err != nil {
		return errors.Wrap(err, "error reading range response from backend")
	}
	return nil
}
