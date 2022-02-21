package remotewrite

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	prometheus_common_config "github.com/prometheus/common/config"
	"github.com/prometheus/prometheus/config"
	"github.com/stretchr/testify/assert"
)

func Test_remoteWriteClient(t *testing.T) {
	var err error
	var capturedHeaders http.Header
	var capturedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		capturedHeaders = req.Header
		capturedBody, err = io.ReadAll(req.Body)
		assert.NoError(t, err)
	}))
	defer server.Close()

	url, err := url.Parse(fmt.Sprintf("http://%s/receive", server.Listener.Addr().String()))
	assert.NoError(t, err)

	cfg := &config.DefaultRemoteWriteConfig
	cfg.URL = &prometheus_common_config.URL{URL: url}

	t.Run("remoteWriteClient with custom headers", func(t *testing.T) {
		cfg.Headers = map[string]string{
			// User-Agent header can not be overridden
			userAgentHeader: "my-custom-user-agent",
			"Authorization": "Basic *****",
		}
		data := []byte("test request #1")

		client, err := newRemoteWriteClient(cfg, "")
		assert.NoError(t, err)

		err = client.Store(context.Background(), data)
		assert.NoError(t, err)

		assert.Equal(t, data, capturedBody)
		expectedHeaders := http.Header{
			"Authorization":                     {"Basic *****"},
			"Content-Encoding":                  {"snappy"},
			"Content-Length":                    {strconv.Itoa(len(data))},
			"Content-Type":                      {"application/x-protobuf"},
			userAgentHeader:                     {remoteWriteUserAgent},
			"X-Prometheus-Remote-Write-Version": {"0.1.0"},
		}
		assert.Equal(t, expectedHeaders, capturedHeaders)
	})

	t.Run("remoteWriteClient with tenantID", func(t *testing.T) {
		cfg.Headers = nil
		data := []byte("test request #2")

		clientWithXScopeOrg, err := newRemoteWriteClient(cfg, "my-tenant")
		assert.NoError(t, err)

		err = clientWithXScopeOrg.Store(context.Background(), data)
		assert.NoError(t, err)

		assert.Equal(t, data, capturedBody)
		expectedHeaders := http.Header{
			"Content-Encoding":                  {"snappy"},
			"Content-Length":                    {strconv.Itoa(len(data))},
			"Content-Type":                      {"application/x-protobuf"},
			userAgentHeader:                     {remoteWriteUserAgent},
			"X-Prometheus-Remote-Write-Version": {"0.1.0"},
			xScopeOrgIDHeader:                   {"my-tenant"},
		}
		assert.Equal(t, expectedHeaders, capturedHeaders)
	})

}

func Test_copyMap(t *testing.T) {
	original := map[string]string{
		"k1": "v1",
		"k2": "v2",
	}

	copied := copyMap(original)

	assert.Equal(t, original, copied)

	copied["k2"] = "other value"
	copied["k3"] = "v3"

	assert.Len(t, original, 2)
	assert.Equal(t, "v2", original["k2"])
	assert.Equal(t, "", original["k3"])
}
