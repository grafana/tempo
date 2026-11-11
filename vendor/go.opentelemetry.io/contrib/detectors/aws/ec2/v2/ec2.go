// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package ec2 provides a resource detector for EC2 instances using aws-sdk-go-v2.
package ec2

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"

	aws "github.com/aws/aws-sdk-go-v2/aws"
	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"github.com/aws/smithy-go/logging"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

var (
	loadDefaultConfig = awsconfig.LoadDefaultConfig
	newIMDSClient     = func(cfg aws.Config) client {
		return imds.NewFromConfig(cfg)
	}
)

// client implements methods to capture EC2 environment metadata information.
type client interface {
	GetInstanceIdentityDocument(ctx context.Context, params *imds.GetInstanceIdentityDocumentInput, optFns ...func(*imds.Options)) (*imds.GetInstanceIdentityDocumentOutput, error)
	GetMetadata(ctx context.Context, params *imds.GetMetadataInput, optFns ...func(*imds.Options)) (*imds.GetMetadataOutput, error)
}

// resource detector collects resource information from EC2 environment.
type resourceDetector struct {
	cfg config

	mu sync.Mutex
	c  client
}

// compile time assertion that imds.Client implements client.
var _ client = (*imds.Client)(nil)

// compile time assertion that resourceDetector implements the resource.Detector interface.
var _ resource.Detector = (*resourceDetector)(nil)

// NewResourceDetector returns a resource detector that will detect AWS EC2 resources.
func NewResourceDetector() resource.Detector {
	return &resourceDetector{}
}

// NewResourceDetectorWithOptions returns a resource detector that will detect
// AWS EC2 resources. The options configure the AWS SDK client.
func NewResourceDetectorWithOptions(opts ...Option) resource.Detector {
	var c config
	for _, opt := range opts {
		opt.apply(&c)
	}
	return &resourceDetector{cfg: c}
}

// WithAWSLogger passes the AWS SDK [logging.Logger] to the AWS SDK client.
// Note this is the AWS SDK's logger, not an OpenTelemetry logger.
func WithAWSLogger(logger logging.Logger) Option {
	return optionFunc(func(c *config) {
		c.logger = logger
	})
}

// Option configures an AWS SDK client.
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (f optionFunc) apply(c *config) { f(c) }

type config struct {
	logger logging.Logger
}

func (detector *resourceDetector) getClient(ctx context.Context) (client, error) {
	detector.mu.Lock()
	defer detector.mu.Unlock()
	if detector.c != nil {
		return detector.c, nil
	}

	c, err := newClient(ctx, detector.cfg)
	if err != nil {
		return nil, err
	}
	detector.c = c
	return c, nil
}

// Detect detects associated resources when running in AWS environment.
func (detector *resourceDetector) Detect(ctx context.Context) (*resource.Resource, error) {
	client, err := detector.getClient(ctx)
	if err != nil {
		return nil, err
	}

	// Available method removed in aws-sdk-go-v2, return empty resource if client returns error
	doc, err := client.GetInstanceIdentityDocument(ctx, nil)
	if err != nil {
		return resource.Empty(), nil
	}

	attributes := []attribute.KeyValue{
		semconv.CloudProviderAWS,
		semconv.CloudPlatformAWSEC2,
		semconv.CloudRegion(doc.Region),
		semconv.CloudAvailabilityZone(doc.AvailabilityZone),
		semconv.CloudAccountID(doc.AccountID),
		semconv.HostID(doc.InstanceID),
		semconv.HostImageID(doc.ImageID),
		semconv.HostType(doc.InstanceType),
	}

	m := &metadata{client: client}
	m.add(ctx, semconv.HostNameKey, "hostname")

	attributes = append(attributes, m.attributes...)

	if len(m.errs) > 0 {
		err = fmt.Errorf("%w: %s", resource.ErrPartialResource, m.errs)
	}

	return resource.NewWithAttributes(semconv.SchemaURL, attributes...), err
}

func newClient(ctx context.Context, c config) (client, error) {
	var optFns []func(*awsconfig.LoadOptions) error
	if c.logger != nil {
		optFns = append(optFns, awsconfig.WithLogger(c.logger))
	}

	cfg, err := loadDefaultConfig(ctx, optFns...)
	if err != nil {
		return nil, err
	}

	return newIMDSClient(cfg), nil
}

type metadata struct {
	client     client
	errs       []error
	attributes []attribute.KeyValue
}

func (m *metadata) add(ctx context.Context, k attribute.Key, n string) {
	metadataInput := &imds.GetMetadataInput{Path: n}
	md, err := m.client.GetMetadata(ctx, metadataInput)
	if err != nil {
		m.recordError(n, err)
		return
	}
	data, err := io.ReadAll(md.Content)
	if err != nil {
		m.recordError(n, err)
		return
	}
	m.attributes = append(m.attributes, k.String(string(data)))
}

func (m *metadata) recordError(path string, err error) {
	var rf *awshttp.ResponseError
	ok := errors.As(err, &rf)
	if !ok {
		m.errs = append(m.errs, fmt.Errorf("%q: %w", path, err))
		return
	}

	if rf.HTTPStatusCode() == http.StatusNotFound {
		return
	}

	m.errs = append(m.errs, fmt.Errorf("%q: %d %s", path, rf.HTTPStatusCode(), rf.Error()))
}
