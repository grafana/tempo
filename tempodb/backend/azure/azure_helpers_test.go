package azure

import (
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
	t.Setenv("AZURE_STORAGE_ACCOUNT", TestStorageAccountName)

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
	t.Setenv("AZURE_STORAGE_KEY", TestStorageAccountKey)

	actual := getStorageAccountKey(&cfg)
	assert.Equal(t, TestStorageAccountKey, actual)
}

func TestGetStorageAccountKeyNotSet(t *testing.T) {
	cfg := Config{}

	actual := getStorageAccountKey(&cfg)
	assert.Equal(t, "", actual)
}
