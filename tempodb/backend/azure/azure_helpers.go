package azure

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/cristalhq/hedgedhttp"

	"github.com/grafana/tempo/tempodb/backend/instrumentation"
)

const (
	maxRetries = 1
)

type authFunctions struct {
	NewOAuthConfigFunc                             func(activeDirectoryEndpoint, tenantID string) (*adal.OAuthConfig, error)
	NewServicePrincipalTokenFromFederatedTokenFunc func(oauthConfig adal.OAuthConfig, clientID string, jwt string, resource string, callbacks ...adal.TokenRefreshCallback) (*adal.ServicePrincipalToken, error)
}

func getContainerClient(ctx context.Context, cfg *Config, hedge bool) (container.Client, error) {
	var err error

	retry := policy.RetryOptions{
		// This configuration option was called "MaxTries" in the old blob.RetryOptions.
		// It stated:
		//
		// "A value of 1 means 1 try and no retries."
		//
		// Since the new option is "MaxRetries" (retries instead of tries), we need to set it to (previous value - 1)
		// to get the same behaviour.
		MaxRetries: int32(maxRetries) - 1,
		// The below three values were set with blob.RetryPolicyExponential in the old SDK.
		// blob.RetryPolicyExponential was a shorthand for setting these defaults:
		//
		//    IfDefault(&o.TryTimeout, 1*time.Minute)
		//    IfDefault(&o.RetryDelay, 4*time.Second)
		//    IfDefault(&o.MaxRetryDelay, 120*time.Second)
		//
		// The new SDK contains no shorthand, so we manually set the same defaults.
		// TODO should these be configuration options in Tempo?
		TryTimeout:    1 * time.Minute,
		RetryDelay:    4 * time.Second,
		MaxRetryDelay: 120 * time.Second,
	}
	if deadline, ok := ctx.Deadline(); ok {
		retry.TryTimeout = time.Until(deadline)
	}

	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	// Default MaxIdleConnsPerHost is 2, increase that to reduce connection turnover
	customTransport.MaxIdleConnsPerHost = 100
	// set total max idle connections to a high number
	customTransport.MaxIdleConns = 100

	// add instrumentation
	transport := instrumentation.NewTransport(customTransport)
	var stats *hedgedhttp.Stats

	// hedge if desired (0 means disabled)
	if hedge && cfg.HedgeRequestsAt != 0 {
		transport, stats, err = hedgedhttp.NewRoundTripperAndStats(cfg.HedgeRequestsAt, cfg.HedgeRequestsUpTo, transport)
		if err != nil {
			return container.Client{}, err
		}
		instrumentation.PublishHedgedMetrics(stats)
	}

	opts := azblob.ClientOptions{}
	opts.Transport = &http.Client{Transport: transport}
	opts.Retry = retry
	opts.Telemetry = policy.TelemetryOptions{
		// "ApplicationID" was called "Value" in the old SDK.
		ApplicationID: "Tempo",
	}

	accountName := getStorageAccountName(cfg)
	u, err := url.Parse(fmt.Sprintf("https://%s.%s", accountName, cfg.Endpoint))

	// If the endpoint doesn't start with blob.core we can assume Azurite is being used
	// So the endpoint should follow Azurite URL style
	// https://learn.microsoft.com/en-us/rest/api/storageservices/get-blob#emulated-storage-service-uri
	if !strings.HasPrefix(cfg.Endpoint, "blob.core") {
		u, err = url.Parse(fmt.Sprintf("http://%s/%s", cfg.Endpoint, accountName))
	}

	if err != nil {
		return container.Client{}, err
	}

	var client *azblob.Client

	if !cfg.UseFederatedToken && !cfg.UseManagedIdentity && cfg.UserAssignedID == "" {
		credential, err := azblob.NewSharedKeyCredential(getStorageAccountName(cfg), getStorageAccountKey(cfg))
		if err != nil {
			return container.Client{}, err
		}

		client, err = azblob.NewClientWithSharedKeyCredential(u.String(), credential, &opts)

		if err != nil {
			return container.Client{}, err
		}
	} else {
		// TODO does this cover all of our previous authentication mechanisms?
		credential, err := azidentity.NewDefaultAzureCredential(&azidentity.DefaultAzureCredentialOptions{})
		if err != nil {
			return container.Client{}, err
		}

		client, err = azblob.NewClient(u.String(), credential, &opts)

		if err != nil {
			return container.Client{}, err
		}
	}

	return *client.ServiceClient().NewContainerClient(cfg.ContainerName), nil
}

// TODO is this function necessary?
func GetContainer(ctx context.Context, conf *Config, hedge bool) (container.Client, error) {
	return getContainerClient(ctx, conf, hedge)
}

func GetBlobURL(ctx context.Context, conf *Config, blobName string) (blob.Client, error) {
	c, err := getContainerClient(ctx, conf, false)
	if err != nil {
		return blob.Client{}, err
	}

	return *c.NewBlobClient(blobName), nil
}

func CreateContainer(ctx context.Context, conf *Config) (container.Client, error) {
	c, err := getContainerClient(ctx, conf, false)
	if err != nil {
		return container.Client{}, err
	}
	_, err = c.Create(ctx, &container.CreateOptions{})
	return c, err
}

func getStorageAccountName(cfg *Config) string {
	accountName := cfg.StorageAccountName
	if accountName == "" {
		accountName = os.Getenv("AZURE_STORAGE_ACCOUNT")
	}

	return accountName
}

func getStorageAccountKey(cfg *Config) string {
	accountKey := cfg.StorageAccountKey.String()
	if accountKey == "" {
		accountKey = os.Getenv("AZURE_STORAGE_KEY")
	}

	return accountKey
}

func servicePrincipalTokenFromFederatedToken(resource string, authFunctions authFunctions) (*adal.ServicePrincipalToken, error) {
	azClientID := os.Getenv("AZURE_CLIENT_ID")
	azTenantID := os.Getenv("AZURE_TENANT_ID")

	azADEndpoint, ok := os.LookupEnv("AZURE_AUTHORITY_HOST")
	if !ok {
		azADEndpoint = azure.PublicCloud.ActiveDirectoryEndpoint
	}

	jwtBytes, err := os.ReadFile(os.Getenv("AZURE_FEDERATED_TOKEN_FILE"))
	if err != nil {
		return nil, err
	}

	jwt := string(jwtBytes)

	oauthConfig, err := authFunctions.NewOAuthConfigFunc(azADEndpoint, azTenantID)
	if err != nil {
		return nil, err
	}

	return authFunctions.NewServicePrincipalTokenFromFederatedTokenFunc(*oauthConfig, azClientID, jwt, resource)
}
