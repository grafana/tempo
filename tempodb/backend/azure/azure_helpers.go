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
	"github.com/cristalhq/hedgedhttp"

	"github.com/grafana/tempo/tempodb/backend/instrumentation"
)

const (
	maxRetries = 1
)

func getContainerClient(ctx context.Context, cfg *Config, hedge bool) (container.Client, error) {
	var err error

	retry := policy.RetryOptions{
		MaxRetries: maxRetries,
		// The values for TryTimeout, RetryDelay and MaxRetryDelay are inherited from the old Azure SDK
		// (azure-storage-blob-go).
		//
		// See https://github.com/Azure/azure-storage-blob-go/blob/905b628ceb292e8d769ae62fb7cc5c5e949360db/azblob/zc_policy_retry.go#L89.
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
		ApplicationID: "Tempo",
	}

	accountName := getStorageAccountName(cfg)
	u, err := url.Parse(fmt.Sprintf("https://%s.%s", accountName, cfg.Endpoint))

	// If the endpoint doesn't start with blob. we can assume Azurite is being used
	// So the endpoint should follow Azurite URL style
	// https://learn.microsoft.com/en-us/rest/api/storageservices/get-blob#emulated-storage-service-uri
	if !strings.HasPrefix(cfg.Endpoint, "blob.") {
		u, err = url.Parse(fmt.Sprintf("http://%s/%s", cfg.Endpoint, accountName))
	}

	if err != nil {
		return container.Client{}, err
	}

	var client *azblob.Client

	switch {
	case cfg.UseFederatedToken:
		credential, err := azidentity.NewWorkloadIdentityCredential(&azidentity.WorkloadIdentityCredentialOptions{})
		if err != nil {
			return container.Client{}, err
		}

		client, err = azblob.NewClient(u.String(), credential, &opts)

		if err != nil {
			return container.Client{}, err
		}
	case cfg.UseManagedIdentity:
		var id azidentity.ManagedIDKind

		if cfg.UserAssignedID != "" {
			id = azidentity.ClientID(cfg.UserAssignedID)
		}

		// azidentity.NewManagedIdentityCredential defaults to a system-assigned identity.
		// We only set options.ID if we want a user-assigned identity.
		// See azidentity.ManagedIdentityCredential.
		credential, err := azidentity.NewManagedIdentityCredential(&azidentity.ManagedIdentityCredentialOptions{
			ID: id,
		})
		if err != nil {
			return container.Client{}, err
		}

		client, err = azblob.NewClient(u.String(), credential, &opts)

		if err != nil {
			return container.Client{}, err
		}
	// If no authentication mechanism has been explicitly specified, assume shared key credential.
	default:
		credential, err := azblob.NewSharedKeyCredential(accountName, getStorageAccountKey(cfg))
		if err != nil {
			return container.Client{}, err
		}

		client, err = azblob.NewClientWithSharedKeyCredential(u.String(), credential, &opts)

		if err != nil {
			return container.Client{}, err
		}
	}

	return *client.ServiceClient().NewContainerClient(cfg.ContainerName), nil
}

func getBlobClient(ctx context.Context, conf *Config, blobName string) (blob.Client, error) {
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
