package azure

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/grafana/tempo/tempodb/backend/instrumentation"

	"github.com/Azure/azure-pipeline-go/pipeline"
	blob "github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/cristalhq/hedgedhttp"
)

const (
	maxRetries = 1
)

var (
	defaultAuthFunctions = authFunctions{
		NewOAuthConfigFunc: adal.NewOAuthConfig,
		NewServicePrincipalTokenFromFederatedTokenFunc: adal.NewServicePrincipalTokenFromFederatedToken,
	}
)

type authFunctions struct {
	NewOAuthConfigFunc                             func(activeDirectoryEndpoint, tenantID string) (*adal.OAuthConfig, error)
	NewServicePrincipalTokenFromFederatedTokenFunc func(oauthConfig adal.OAuthConfig, clientID string, jwt string, resource string, callbacks ...adal.TokenRefreshCallback) (*adal.ServicePrincipalToken, error)
}

func GetContainerURL(ctx context.Context, cfg *Config, hedge bool) (blob.ContainerURL, error) {
	var err error
	var p pipeline.Pipeline

	retryOptions := blob.RetryOptions{
		MaxTries: int32(maxRetries),
		Policy:   blob.RetryPolicyExponential,
	}
	if deadline, ok := ctx.Deadline(); ok {
		retryOptions.TryTimeout = time.Until(deadline)
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
			return blob.ContainerURL{}, err
		}
		instrumentation.PublishHedgedMetrics(stats)
	}

	client := http.Client{Transport: transport}

	httpSender := pipeline.FactoryFunc(func(next pipeline.Policy, po *pipeline.PolicyOptions) pipeline.PolicyFunc {
		return func(ctx context.Context, request pipeline.Request) (pipeline.Response, error) {

			// Send the request over the network
			resp, err := client.Do(request.WithContext(ctx))

			return pipeline.NewHTTPResponse(resp), err
		}
	})

	opts := blob.PipelineOptions{
		Retry:      retryOptions,
		Telemetry:  blob.TelemetryOptions{Value: "Tempo"},
		HTTPSender: httpSender,
	}

	if !cfg.UseFederatedToken && !cfg.UseManagedIdentity && cfg.UserAssignedID == "" {
		credential, err := blob.NewSharedKeyCredential(getStorageAccountName(cfg), getStorageAccountKey(cfg))
		if err != nil {
			return blob.ContainerURL{}, err
		}

		p = blob.NewPipeline(credential, opts)
	} else {
		credential, err := getOAuthToken(cfg)
		if err != nil {
			return blob.ContainerURL{}, err
		}

		p = blob.NewPipeline(*credential, opts)
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
		return blob.ContainerURL{}, err
	}

	service := blob.NewServiceURL(*u, p)

	return service.NewContainerURL(cfg.ContainerName), nil
}

func GetContainer(ctx context.Context, conf *Config, hedge bool) (blob.ContainerURL, error) {
	c, err := GetContainerURL(ctx, conf, hedge)
	if err != nil {
		return blob.ContainerURL{}, err
	}
	return c, nil
}

func GetBlobURL(ctx context.Context, conf *Config, blobName string) (blob.BlockBlobURL, error) {
	c, err := GetContainerURL(ctx, conf, false)
	if err != nil {
		return blob.BlockBlobURL{}, err
	}
	return c.NewBlockBlobURL(blobName), nil
}

func CreateContainer(ctx context.Context, conf *Config) (blob.ContainerURL, error) {
	c, err := GetContainerURL(ctx, conf, false)
	if err != nil {
		return blob.ContainerURL{}, err
	}
	_, err = c.Create(
		ctx,
		blob.Metadata{},
		blob.PublicAccessNone)
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

func getOAuthToken(cfg *Config) (*blob.TokenCredential, error) {
	spt, err := getServicePrincipalToken(cfg)
	if err != nil {
		return nil, err
	}

	// Refresh obtains a fresh token
	err = spt.Refresh()
	if err != nil {
		return nil, err
	}

	tc := blob.NewTokenCredential(spt.Token().AccessToken, func(tc blob.TokenCredential) time.Duration {
		err := spt.Refresh()
		if err != nil {
			// something went wrong, prevent the refresher from being triggered again
			return 0
		}

		// set the new token value
		tc.SetToken(spt.Token().AccessToken)

		// get the next token slightly before the current one expires
		return time.Until(spt.Token().Expires()) - 10*time.Second
	})

	return &tc, nil
}

func getServicePrincipalToken(cfg *Config) (*adal.ServicePrincipalToken, error) {
	endpoint := cfg.Endpoint

	resource := fmt.Sprintf("https://%s.%s", cfg.StorageAccountName, endpoint)

	if cfg.UseFederatedToken {
		token, err := servicePrincipalTokenFromFederatedToken(resource, defaultAuthFunctions)
		if err != nil {
			return nil, err
		}

		var customRefreshFunc adal.TokenRefresh = func(context context.Context, resource string) (*adal.Token, error) {
			newToken, err := servicePrincipalTokenFromFederatedToken(resource, defaultAuthFunctions)
			if err != nil {
				return nil, err
			}

			err = newToken.Refresh()
			if err != nil {
				return nil, err
			}

			token := newToken.Token()

			return &token, nil
		}

		token.SetCustomRefreshFunc(customRefreshFunc)
		return token, err
	}

	msiConfig := auth.MSIConfig{
		Resource: resource,
	}

	if cfg.UserAssignedID != "" {
		msiConfig.ClientID = cfg.UserAssignedID
	}

	return msiConfig.ServicePrincipalToken()
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
