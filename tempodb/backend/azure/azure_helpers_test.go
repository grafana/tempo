package azure

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/grafana/dskit/flagext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	TestStorageAccountName = "foobar"
	TestStorageAccountKey  = "abc123"
)

// TestGetStorageAccountName* explicitly broken out into
// separate tests instead of table-driven due to usage of t.SetEnv
func TestGetStorageAccountNameInConfig(t *testing.T) {
	cfg := Config{StorageAccountName: TestStorageAccountName}

	actual := getStorageAccountName(&cfg)
	assert.Equal(t, TestStorageAccountName, actual)
}

func TestGetStorageAccountNameInEnv(t *testing.T) {
	cfg := Config{}
	os.Setenv("AZURE_STORAGE_ACCOUNT", TestStorageAccountName)
	defer os.Unsetenv("AZURE_STORAGE_ACCOUNT")

	actual := getStorageAccountName(&cfg)
	assert.Equal(t, TestStorageAccountName, actual)
}

func TestGetStorageAccountNameNotSet(t *testing.T) {
	cfg := Config{}

	actual := getStorageAccountName(&cfg)
	assert.Equal(t, "", actual)
}

// TestGetStorageAccountKey* explicitly broken out into
// separate tests instead of table-driven due to usage of t.SetEnv
func TestGetStorageAccountKeyInConfig(t *testing.T) {
	storageAccountKeySecret := flagext.SecretWithValue(TestStorageAccountKey)
	cfg := Config{StorageAccountKey: storageAccountKeySecret}

	actual := getStorageAccountKey(&cfg)
	assert.Equal(t, TestStorageAccountKey, actual)
}

func TestGetStorageAccountKeyInEnv(t *testing.T) {
	cfg := Config{}
	os.Setenv("AZURE_STORAGE_KEY", TestStorageAccountKey)
	defer os.Unsetenv("AZURE_STORAGE_KEY")

	actual := getStorageAccountKey(&cfg)
	assert.Equal(t, TestStorageAccountKey, actual)
}

func TestGetStorageAccountKeyNotSet(t *testing.T) {
	cfg := Config{}

	actual := getStorageAccountKey(&cfg)
	assert.Equal(t, "", actual)
}

func TestGetContainerClient(t *testing.T) {
	cfg := Config{
		StorageAccountName: "devstoreaccount1",
		StorageAccountKey:  flagext.SecretWithValue("dGVzdAo="),
		ContainerName:      "traces",
	}

	tests := []struct {
		name        string
		endpoint    string
		expectedURL string
	}{
		{
			name:        "localhost",
			endpoint:    "localhost:10000",
			expectedURL: "http://localhost:10000/devstoreaccount1/traces",
		},
		{
			name:        "Azure China",
			endpoint:    "blob.core.chinacloudapi.cn",
			expectedURL: "https://devstoreaccount1.blob.core.chinacloudapi.cn/traces",
		},
		{
			name:        "Azure US Government",
			endpoint:    "blob.core.usgovcloudapi.net",
			expectedURL: "https://devstoreaccount1.blob.core.usgovcloudapi.net/traces",
		},
		{
			name:        "Azure German",
			endpoint:    "blob.core.cloudapi.de",
			expectedURL: "https://devstoreaccount1.blob.core.cloudapi.de/traces",
		},
		// FQDN test cases for Kubernetes ndots=5 support (issue #1726).
		// Users can add a trailing dot to endpoint_suffix in config to hint
		// kube-dns to skip local search and perform a direct DNS lookup.
		// The URL must RETAIN the trailing dot (for FQDN DNS resolution),
		// and the fqdnTransport strips it from the Host header before sending.
		{
			name:        "Azure Global FQDN with trailing dot",
			endpoint:    "blob.core.windows.net.",
			expectedURL: "https://devstoreaccount1.blob.core.windows.net./traces",
		},
		{
			name:        "Azure China FQDN with trailing dot",
			endpoint:    "blob.core.chinacloudapi.cn.",
			expectedURL: "https://devstoreaccount1.blob.core.chinacloudapi.cn./traces",
		},
		{
			name:        "Azure US Government FQDN with trailing dot",
			endpoint:    "blob.core.usgovcloudapi.net.",
			expectedURL: "https://devstoreaccount1.blob.core.usgovcloudapi.net./traces",
		},
		{
			name:        "Azure German FQDN with trailing dot",
			endpoint:    "blob.core.cloudapi.de.",
			expectedURL: "https://devstoreaccount1.blob.core.cloudapi.de./traces",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg.Endpoint = tc.endpoint

			client, err := getContainerClient(context.Background(), &cfg, false)
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedURL, client.URL())
		})
	}
}

// TestFqdnTransport_StripsTrailingDotFromHostHeader verifies that fqdnTransport
// strips the trailing dot from the Host header before sending the request,
// while keeping the URL unchanged so DNS resolution uses the FQDN.
//
// This is the core mechanism for issue #1726: the trailing dot in the URL
// tells kube-dns to skip local search, and stripping it from the Host header
// satisfies Azure/S3/GCS APIs that reject a trailing dot with 400/404/301.
func TestFqdnTransport_StripsTrailingDotFromHostHeader(t *testing.T) {
	tests := []struct {
		name             string
		requestHost      string
		expectedHostHdr  string
	}{
		{
			name:            "trailing dot is stripped from Host header",
			requestHost:     "devstoreaccount1.blob.core.windows.net.",
			expectedHostHdr: "devstoreaccount1.blob.core.windows.net",
		},
		{
			name:            "no trailing dot is unchanged",
			requestHost:     "devstoreaccount1.blob.core.windows.net",
			expectedHostHdr: "devstoreaccount1.blob.core.windows.net",
		},
		{
			name:            "Azure China FQDN trailing dot stripped",
			requestHost:     "devstoreaccount1.blob.core.chinacloudapi.cn.",
			expectedHostHdr: "devstoreaccount1.blob.core.chinacloudapi.cn",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Capture the Host header received by the server
			var receivedHost string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedHost = r.Host
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			// Build a request with the test Host value
			req, err := http.NewRequest(http.MethodGet, server.URL, nil)
			require.NoError(t, err)
			req.Host = tc.requestHost

			// Use fqdnTransport wrapping the default transport
			transport := &fqdnTransport{wrapped: http.DefaultTransport}
			resp, err := transport.RoundTrip(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			// The Host header received by the server must never have a trailing dot
			assert.Equal(t, tc.expectedHostHdr, receivedHost,
				"Host header must have trailing dot stripped before sending to server")
		})
	}
}