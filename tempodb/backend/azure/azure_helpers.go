package azure

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Azure/azure-pipeline-go/pipeline"
	blob "github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/cristalhq/hedgedhttp"
)

const (
	maxRetries         = 1
	uptoHedgedRequests = 2
)

func GetContainerURL(ctx context.Context, cfg *Config, hedge bool) (blob.ContainerURL, error) {
	c, err := blob.NewSharedKeyCredential(cfg.StorageAccountName.String(), cfg.StorageAccountKey.String())
	if err != nil {
		return blob.ContainerURL{}, err
	}

	retryOptions := blob.RetryOptions{
		MaxTries: int32(maxRetries),
		Policy:   blob.RetryPolicyExponential,
	}
	if deadline, ok := ctx.Deadline(); ok {
		retryOptions.TryTimeout = time.Until(deadline)
	}

	var httpSender pipeline.Factory
	if hedge && cfg.HedgeRequestsAt != 0 {
		httpSender = pipeline.FactoryFunc(func(next pipeline.Policy, po *pipeline.PolicyOptions) pipeline.PolicyFunc {
			return func(ctx context.Context, request pipeline.Request) (pipeline.Response, error) {

				customTransport := http.DefaultTransport.(*http.Transport).Clone()

				transport := newInstrumentedTransport(customTransport)
				transport = hedgedhttp.NewRoundTripper(cfg.HedgeRequestsAt, uptoHedgedRequests, transport)

				client := http.Client{Transport: transport}

				// Send the request over the network
				resp, err := client.Do(request.WithContext(ctx))

				return pipeline.NewHTTPResponse(resp), err
			}
		})
	}

	p := blob.NewPipeline(c, blob.PipelineOptions{
		Retry:      retryOptions,
		Telemetry:  blob.TelemetryOptions{Value: "Tempo"},
		HTTPSender: httpSender,
	})

	u, err := url.Parse(fmt.Sprintf("https://%s.%s", cfg.StorageAccountName, cfg.Endpoint))

	// If the endpoint doesn't start with blob.core we can assume Azurite is being used
	// So the endpoint should follow Azurite URL style
	if !strings.HasPrefix(cfg.Endpoint, "blob.core") {
		u, err = url.Parse(fmt.Sprintf("http://%s/%s", cfg.Endpoint, cfg.StorageAccountName))
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
	// Getting container properties to check if container exists
	_, err = c.GetProperties(ctx, blob.LeaseAccessConditions{})
	return c, err
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
