// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package azurevm

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

const defaultAzureVMMetadataEndpoint = "http://169.254.169.254/metadata/instance/compute?api-version=2021-12-13&format=json"

// Azure-specific resource attribute keys that are not (yet) part of the
// semantic conventions package.
const (
	azureVMNameKey         = attribute.Key("azure.vm.name")
	azureVMSizeKey         = attribute.Key("azure.vm.size")
	azureVMScaleSetNameKey = attribute.Key("azure.vm.scaleset.name")
	azureTagPrefix         = "azure.tag."
)

type vmMetadata struct {
	VMId              *string      `json:"vmId"`
	Location          *string      `json:"location"`
	ResourceId        *string      `json:"resourceId"`
	Name              *string      `json:"name"`
	VMSize            *string      `json:"vmSize"`
	OsType            *string      `json:"osType"`
	Version           *string      `json:"version"`
	SubscriptionId    *string      `json:"subscriptionId"`
	ResourceGroupName *string      `json:"resourceGroupName"`
	VMScaleSetName    *string      `json:"vmScaleSetName"`
	Zone              *string      `json:"zone"`
	OsProfile         *osProfile   `json:"osProfile"`
	TagsList          []computeTag `json:"tagsList"`
}

type osProfile struct {
	ComputerName *string `json:"computerName"`
}

type computeTag struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type config struct {
	filter       attribute.Filter
	tagKeyFilter func(key string) bool
}

// Option configures a [ResourceDetector].
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (f optionFunc) apply(c *config) { f(c) }

// WithAttributeFilter sets a filter that controls which detected attributes are
// included in the returned resource. Only attributes for which filter returns
// true are included. By default all attributes are included.
func WithAttributeFilter(filter attribute.Filter) Option {
	return optionFunc(func(c *config) { c.filter = filter })
}

// WithTagKeyFilter emits an azure.tag.<name> attribute for every VM tag whose
// key satisfies filter. The filter receives the raw Azure tag name, without the
// "azure.tag." prefix. Without this option no VM tags are emitted. For regexp
// matching, pass re.MatchString.
//
// Tag attributes are subject to [WithAttributeFilter] like any other attribute.
func WithTagKeyFilter(filter func(key string) bool) Option {
	return optionFunc(func(c *config) { c.tagKeyFilter = filter })
}

// ResourceDetector collects resource information of Azure VMs.
type ResourceDetector struct {
	endpoint string
	cfg      config
}

// Compile time assertion that ResourceDetector implements the resource.Detector interface.
var _ resource.Detector = (*ResourceDetector)(nil)

// New returns a [ResourceDetector] that will detect Azure VM resources.
//
// It is equivalent to [NewResourceDetector] called with no options.
func New() *ResourceDetector {
	return NewResourceDetector()
}

// NewResourceDetector returns a [ResourceDetector] that will detect Azure VM
// resources.
func NewResourceDetector(opts ...Option) *ResourceDetector {
	var cfg config
	for _, opt := range opts {
		opt.apply(&cfg)
	}
	return &ResourceDetector{endpoint: defaultAzureVMMetadataEndpoint, cfg: cfg}
}

// Detect detects associated resources when running on an Azure VM.
func (detector *ResourceDetector) Detect(ctx context.Context) (*resource.Resource, error) {
	jsonMetadata, runningInAzure, err := detector.getJSONMetadata(ctx)
	if err != nil {
		if !runningInAzure {
			return resource.Empty(), nil
		}

		return nil, err
	}

	var metadata vmMetadata
	err = json.Unmarshal(jsonMetadata, &metadata)
	if err != nil {
		return nil, err
	}

	attributes := []attribute.KeyValue{
		semconv.CloudProviderAzure,
		semconv.CloudPlatformAzureVM,
	}

	if metadata.VMId != nil {
		attributes = append(attributes, semconv.HostID(*metadata.VMId))
	}
	if metadata.Location != nil {
		attributes = append(attributes, semconv.CloudRegion(*metadata.Location))
	}
	if metadata.ResourceId != nil {
		attributes = append(attributes, semconv.CloudResourceID(*metadata.ResourceId))
	}
	// Prefer osProfile.computerName for host.name, falling back to the VM name
	// if it is unavailable (e.g., VMs created from specialized disks).
	if metadata.OsProfile != nil && metadata.OsProfile.ComputerName != nil && *metadata.OsProfile.ComputerName != "" {
		attributes = append(attributes, semconv.HostName(*metadata.OsProfile.ComputerName))
	} else if metadata.Name != nil {
		attributes = append(attributes, semconv.HostName(*metadata.Name))
	}
	if metadata.VMSize != nil {
		attributes = append(attributes, semconv.HostType(*metadata.VMSize))
	}
	if metadata.OsType != nil {
		attributes = append(attributes, semconv.OSTypeKey.String(*metadata.OsType))
	}
	if metadata.Version != nil {
		attributes = append(attributes, semconv.OSVersion(*metadata.Version))
	}
	if metadata.SubscriptionId != nil {
		attributes = append(attributes, semconv.CloudAccountID(*metadata.SubscriptionId))
	}
	if metadata.Zone != nil && *metadata.Zone != "" {
		attributes = append(attributes, semconv.CloudAvailabilityZone(*metadata.Zone))
	}
	if metadata.Name != nil {
		attributes = append(attributes, azureVMNameKey.String(*metadata.Name))
	}
	if metadata.VMSize != nil {
		attributes = append(attributes, azureVMSizeKey.String(*metadata.VMSize))
	}
	if metadata.VMScaleSetName != nil && *metadata.VMScaleSetName != "" {
		attributes = append(attributes, azureVMScaleSetNameKey.String(*metadata.VMScaleSetName))
	}
	if metadata.ResourceGroupName != nil {
		attributes = append(attributes, semconv.AzureResourceGroupName(*metadata.ResourceGroupName))
	}

	if detector.cfg.tagKeyFilter != nil {
		for _, tag := range metadata.TagsList {
			if detector.cfg.tagKeyFilter(tag.Name) {
				attributes = append(attributes, attribute.String(azureTagPrefix+tag.Name, tag.Value))
			}
		}
	}

	if detector.cfg.filter != nil {
		filtered := attributes[:0]
		for _, kv := range attributes {
			if detector.cfg.filter(kv) {
				filtered = append(filtered, kv)
			}
		}
		attributes = filtered
	}

	return resource.NewWithAttributes(semconv.SchemaURL, attributes...), nil
}

func (detector *ResourceDetector) getJSONMetadata(ctx context.Context) ([]byte, bool, error) {
	pTransport := &http.Transport{Proxy: nil}

	client := http.Client{Transport: pTransport}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, detector.endpoint, http.NoBody)
	if err != nil {
		return nil, false, err
	}

	req.Header.Add("Metadata", "True")

	resp, err := client.Do(req)
	if err != nil {
		return nil, false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		bytes, err := io.ReadAll(resp.Body)
		return bytes, true, err
	}

	runningInAzure := resp.StatusCode < 400 || resp.StatusCode > 499

	return nil, runningInAzure, errors.New(http.StatusText(resp.StatusCode))
}
