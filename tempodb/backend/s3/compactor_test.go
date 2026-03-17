package s3

import (
	"encoding/base64"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/grafana/dskit/flagext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
