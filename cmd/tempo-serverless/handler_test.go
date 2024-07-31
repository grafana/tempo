package serverless

import (
	"os"
	"testing"
	"time"

	"github.com/grafana/tempo/v2/tempodb/backend"
)

func TestLoadConfig(t *testing.T) {
	os.Setenv("TEMPO_GCS_BUCKET_NAME", "some-random-gcs-bucket")
	os.Setenv("TEMPO_GCS_CHUNK_BUFFER_SIZE", "10")
	os.Setenv("TEMPO_GCS_HEDGE_REQUESTS_AT", "400ms")
	os.Setenv("TEMPO_BACKEND", backend.GCS)

	// purposefully not using testfiy to reduce dependencies and keep the serverless packages small
	cfg, err := loadConfig()
	if err != nil {
		t.Error("failed to load config:", err)
		return
	}
	if cfg.Backend != backend.GCS {
		t.Error("backend should be gcs", cfg.Backend)
	}
	if cfg.GCS.BucketName != "some-random-gcs-bucket" {
		t.Error("gcs bucket name should be some-random-gcs-bucket", cfg.GCS.BucketName)
	}
	if cfg.GCS.ChunkBufferSize != 10 {
		t.Error("gcs chunk buffer size should be 10", cfg.GCS.ChunkBufferSize)
	}
	if cfg.GCS.HedgeRequestsAt != 400*time.Millisecond {
		t.Error("gcs hedge requests at should be 400ms", cfg.GCS.HedgeRequestsAt)
	}
}

func TestLoadConfigS3(t *testing.T) {
	os.Setenv("TEMPO_S3_BUCKET", "tempo")
	os.Setenv("TEMPO_S3_ENDPOINT", "glerg")
	os.Setenv("TEMPO_BACKEND", backend.S3)
	os.Setenv("TEMPO_S3_ACCESS_KEY", "access")
	os.Setenv("TEMPO_S3_SECRET_KEY", "secret")

	cfg, err := loadConfig()
	if err != nil {
		t.Error("failed to load config", err)
		return
	}
	if cfg.Backend != backend.S3 {
		t.Error("backend should be s3", cfg.Backend)
	}
	if cfg.S3.Bucket != "tempo" {
		t.Error("s3 bucket name should be tempo", cfg.S3.Bucket)
	}
	if cfg.S3.Endpoint != "glerg" {
		t.Error("s3 endpoint should be glerg", cfg.S3.Endpoint)
	}
	if cfg.S3.AccessKey != "access" {
		t.Error("s3 access key should be access", cfg.S3.AccessKey)
	}
	if cfg.S3.SecretKey.String() != "secret" {
		t.Error("s3 secret key should be secret", cfg.S3.SecretKey)
	}
}

func TestLoadConfigAzure(t *testing.T) {
	os.Setenv("TEMPO_BACKEND", backend.Azure)
	os.Setenv("TEMPO_AZURE_MAX_BUFFERS", "3")

	cfg, err := loadConfig()
	if err != nil {
		t.Error("failed to load config", err)
		return
	}
	if cfg.Backend != backend.Azure {
		t.Error("backend should be azure", cfg.Backend)
	}
	if cfg.Azure.MaxBuffers != 3 {
		t.Error("azure max buffers should be 3", cfg.Azure.MaxBuffers)
	}
}
