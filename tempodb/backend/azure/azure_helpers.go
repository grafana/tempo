package azure

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	blob "github.com/Azure/azure-storage-blob-go/azblob"
	adal "github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/pkg/errors"
)

const maxRetries = 3

// getAccessToken retrieves Azure API access token.
func getAccessToken(conf *Config) (*adal.ServicePrincipalToken, error) {

	environment, err := azure.EnvironmentFromName(conf.AzureEnvironment)
	if err != nil {
		return nil, errors.Wrap(err, "invalid cloud value")
	}
	// Try to retrieve token with service principal credentials.
	if len(conf.ClientID.String()) > 0 &&
		len(conf.ClientSecret.String()) > 0 {
		oauthConfig, err := adal.NewOAuthConfig(environment.ActiveDirectoryEndpoint, conf.TenantID.String())
		if err != nil {
			return nil, errors.Wrap(err, "failed to retrieve OAuth config")
		}
		token, err := adal.NewServicePrincipalToken(*oauthConfig, conf.ClientID.String(), conf.ClientSecret.String(), environment.ResourceIdentifiers.Storage)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create service principal token")
		}
		return token, nil
	}

	// Try to retrieve token with MSI.
	msiEndpoint, err := adal.GetMSIVMEndpoint()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the managed service identity endpoint")
	}
	token, err := adal.NewServicePrincipalTokenFromMSI(msiEndpoint, environment.ServiceManagementEndpoint)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create the managed service identity token")
	}
	return token, nil
}
func GetContainerURL(ctx context.Context, conf *Config) (blob.ContainerURL, error) {

	var spt blob.Credential
	if conf.UseManagedIdentity {
		token, err := getAccessToken(conf)
		if err != nil {
			return blob.ContainerURL{}, err
		}
		err = token.Refresh()
		if err != nil {
			return blob.ContainerURL{}, err
		}
		spt = blob.NewTokenCredential(token.Token().AccessToken, func(tc blob.TokenCredential) time.Duration {
			_ = token.Refresh()
			return time.Until(token.Token().Expires())
		})

	} else {
		var err error
		spt, err = blob.NewSharedKeyCredential(conf.StorageAccountName.String(), conf.StorageAccountKey.String())
		if err != nil {
			return blob.ContainerURL{}, err
		}
	}

	retryOptions := blob.RetryOptions{
		MaxTries: int32(maxRetries),
		Policy:   blob.RetryPolicyExponential,
	}
	if deadline, ok := ctx.Deadline(); ok {
		retryOptions.TryTimeout = time.Until(deadline)
	}

	p := blob.NewPipeline(spt, blob.PipelineOptions{
		Retry:     retryOptions,
		Telemetry: blob.TelemetryOptions{Value: "Tempo"},
	})

	u, err := url.Parse(fmt.Sprintf("https://%s.%s", conf.StorageAccountName, conf.Endpoint))

	// If the endpoint doesn't start with blob.core we can assume Azurite is being used
	// So the endpoint should follow Azurite URL style
	if !strings.HasPrefix(conf.Endpoint, "blob.core") {
		u, err = url.Parse(fmt.Sprintf("http://%s/%s", conf.Endpoint, conf.StorageAccountName))
	}

	if err != nil {
		return blob.ContainerURL{}, err
	}

	service := blob.NewServiceURL(*u, p)

	return service.NewContainerURL(conf.ContainerName), nil
}

func GetContainer(ctx context.Context, conf *Config) (blob.ContainerURL, error) {
	c, err := GetContainerURL(ctx, conf)
	if err != nil {
		return blob.ContainerURL{}, err
	}
	// Getting container properties to check if container exists
	_, err = c.GetProperties(ctx, blob.LeaseAccessConditions{})
	return c, err
}

func GetBlobURL(ctx context.Context, conf *Config, blobName string) (blob.BlockBlobURL, error) {
	c, err := GetContainerURL(ctx, conf)
	if err != nil {
		return blob.BlockBlobURL{}, err
	}
	return c.NewBlockBlobURL(blobName), nil
}

func CreateContainer(ctx context.Context, conf *Config) (blob.ContainerURL, error) {
	c, err := GetContainerURL(ctx, conf)
	if err != nil {
		return blob.ContainerURL{}, err
	}
	_, err = c.Create(
		ctx,
		blob.Metadata{},
		blob.PublicAccessNone)
	return c, err
}
