package s3

import (
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/grafana/dskit/flagext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/tempodb/backend"
)

// CompactedBlockMeta's NotFound path (blocklist/poller.go's pollBlock treats it
// as a benign "block in an intermediate state", not a poll error) has no
// coverage: TestReadError only tests the classification helper in isolation,
// never that CompactedBlockMeta actually routes a real 404 through it.
func TestCompactedBlockMeta_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<Error><Code>NoSuchKey</Code><Message>The specified key does not exist.</Message></Error>`))
	}))
	t.Cleanup(server.Close)

	_, _, c, err := NewNoConfirm(&Config{
		Region:    "blerg",
		AccessKey: "test",
		SecretKey: flagext.SecretWithValue("test"),
		Bucket:    "blerg",
		Insecure:  true,
		Endpoint:  server.URL[7:], // [7:] -> strip http://
	})
	require.NoError(t, err)

	_, err = c.CompactedBlockMeta(uuid.New(), "tenant1")
	require.Error(t, err)
	require.True(t, errors.Is(err, backend.ErrDoesNotExist))
}

func TestMarkBlockCompacted(t *testing.T) {
	sseConfig := SSEConfig{
		Type:                 SSEKMS,
		KMSKeyID:             "my-kms-key-id",
		KMSEncryptionContext: "{}",
	}

	tags := map[string]string{"env": "prod", "app": "thing"}

	testedHeaders := []string{
		sseHeader,
		sseKMSKeyIDHeader,
		sseKMSContextHeader,
		tagHeader,
	}

	tests := []struct {
		name                 string
		tags                 map[string]string
		sse                  SSEConfig
		expectedHeaderValues map[string]string
	}{
		{
			"sse and tags",
			tags,
			sseConfig,
			map[string]string{
				sseHeader:           "aws:kms",
				sseKMSKeyIDHeader:   sseConfig.KMSKeyID,
				sseKMSContextHeader: base64.StdEncoding.EncodeToString([]byte(sseConfig.KMSEncryptionContext)),
				tagHeader:           "app=thing&env=prod",
			},
		},
		{
			"tags without sse",
			tags,
			SSEConfig{},
			map[string]string{
				tagHeader: "app=thing&env=prod",
			},
		},
		{
			"sse without tags",
			nil,
			sseConfig,
			map[string]string{
				sseHeader:           "aws:kms",
				sseKMSKeyIDHeader:   sseConfig.KMSKeyID,
				sseKMSContextHeader: base64.StdEncoding.EncodeToString([]byte(sseConfig.KMSEncryptionContext)),
			},
		},
		{
			"no sse or tag headers",
			nil,
			SSEConfig{},
			map[string]string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// rawObject := raw.Object{}
			var httpHeader http.Header

			server := fakeServerWithHeader(t, &httpHeader)
			_, _, c, err := New(&Config{
				Region:    "blerg",
				AccessKey: "test",
				SecretKey: flagext.SecretWithValue("test"),
				Bucket:    "blerg",
				Insecure:  true,
				Endpoint:  server.URL[7:], // [7:] -> strip http://
				SSE:       tc.sse,
				Tags:      tc.tags,
			})
			require.NoError(t, err)

			_ = c.MarkBlockCompacted(uuid.New(), "tenant1")

			// check expected headers to be set with expected values
			for headerKey, expectedHeaderValue := range tc.expectedHeaderValues {
				headerValue := httpHeader.Get(headerKey)
				assert.Equal(t, expectedHeaderValue, headerValue, "expected header %s to have value %s", headerKey, expectedHeaderValue)
			}

			// check no unexpected headers are set
			for _, testedHeader := range testedHeaders {
				_, ok := tc.expectedHeaderValues[testedHeader]
				if !ok {
					require.Empty(t, httpHeader.Get(testedHeader), "expected header %s to be empty", testedHeader)
				}
			}
		})
	}
}
