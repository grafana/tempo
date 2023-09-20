package azure

import (
	"context"
	"os"
	"testing"

	"github.com/grafana/dskit/flagext"
	"github.com/stretchr/testify/assert"
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
