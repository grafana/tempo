package handler

import (
	"os"
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	os.Setenv("TEMPO_GCS_BUCKET_NAME", "some-random-gcs-bucket")
	os.Setenv("TEMPO_GCS_CHUNK_BUFFER_SIZE", "10")
	os.Setenv("TEMPO_GCS_HEDGE_REQUESTS_AT", "400ms")
	os.Setenv("TEMPO_BACKEND", "gcs")

	// purposefully not using testfiy to reduce dependencies and keep the serverless packages small
	cfg, err := loadConfig()
	if err != nil {
		t.Error("failed to load config", err)
	}
	if cfg.Backend != "gcs" {
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
