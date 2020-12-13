package azure

import (
	"context"
	"fmt"
	"net/url"
	"time"

	blob "github.com/Azure/azure-storage-blob-go/azblob"
)

const maxRetries = 3

func GetContainerURL(ctx context.Context, conf *Config) (blob.ContainerURL, error) {
	c, err := blob.NewSharedKeyCredential(conf.StorageAccountName, conf.StorageAccountKey)
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

	p := blob.NewPipeline(c, blob.PipelineOptions{
		Retry:     retryOptions,
		Telemetry: blob.TelemetryOptions{Value: "Tempo"},
	})

	u, err := url.Parse(fmt.Sprintf("https://%s.%s", conf.StorageAccountName, conf.Endpoint))

	if conf.DevelopmentMode {
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
