package azure

import (
	"context"
	"os"
	"testing"

	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/grafana/dskit/flagext"
	"github.com/stretchr/testify/assert"
)

const (
	TestStorageAccountName = "foobar"
	TestStorageAccountKey  = "abc123"
	TestAzureClientID      = "myClientId"
	TestAzureTenantID      = "myTenantId"
	TestAzureADEndpoint    = "https://example.com/"
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

func TestGetContainerURL(t *testing.T) {
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

			url, err := GetContainerURL(context.Background(), &cfg, false)
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedURL, url.String())
		})
	}
}

func TestServicePrincipalTokenFromFederatedToken(t *testing.T) {
	os.Setenv("AZURE_CLIENT_ID", TestAzureClientID)
	defer os.Unsetenv("AZURE_CLIENT_ID")
	os.Setenv("AZURE_TENANT_ID", TestAzureTenantID)
	defer os.Unsetenv("AZURE_TENANT_ID")
	os.Setenv("AZURE_AUTHORITY_HOST", TestAzureADEndpoint)
	defer os.Unsetenv("AZURE_AUTHORITY_HOST")

	mockOAuthConfig, _ := adal.NewOAuthConfig(TestAzureADEndpoint, "bar")
	mockedServicePrincipalToken := new(adal.ServicePrincipalToken)

	tmpDir := t.TempDir()
	_ = os.WriteFile(tmpDir+"/jwtToken", []byte("myJwtToken"), 0666)
	os.Setenv("AZURE_FEDERATED_TOKEN_FILE", tmpDir+"/jwtToken")
	defer os.Unsetenv("AZURE_FEDERATED_TOKEN_FILE")

	newOAuthConfigFunc := func(activeDirectoryEndpoint, tenantID string) (*adal.OAuthConfig, error) {
		assert.Equal(t, TestAzureADEndpoint, activeDirectoryEndpoint)
		assert.Equal(t, TestAzureTenantID, tenantID)

		_, err := adal.NewOAuthConfig(activeDirectoryEndpoint, tenantID)
		assert.NoError(t, err)

		return mockOAuthConfig, nil
	}

	servicePrincipalTokenFromFederatedTokenFunc := func(oauthConfig adal.OAuthConfig, clientID string, jwt string, resource string, callbacks ...adal.TokenRefreshCallback) (*adal.ServicePrincipalToken, error) {
		assert.True(t, *mockOAuthConfig == oauthConfig, "should return the mocked object")
		assert.Equal(t, TestAzureClientID, clientID)
		assert.Equal(t, "myJwtToken", jwt)
		assert.Equal(t, "https://bar.blob.core.windows.net", resource)
		return mockedServicePrincipalToken, nil
	}

	token, err := servicePrincipalTokenFromFederatedToken("https://bar.blob.core.windows.net", authFunctions{
		newOAuthConfigFunc,
		servicePrincipalTokenFromFederatedTokenFunc,
	})

	assert.NoError(t, err)
	assert.True(t, mockedServicePrincipalToken == token, "should return the mocked object")
}
