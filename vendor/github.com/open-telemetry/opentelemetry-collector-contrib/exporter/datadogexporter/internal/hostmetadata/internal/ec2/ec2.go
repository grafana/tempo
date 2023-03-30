// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package ec2 contains the AWS EC2 hostname provider
package ec2 // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/datadogexporter/internal/hostmetadata/internal/ec2"

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/DataDog/opentelemetry-mapping-go/pkg/otlp/attributes/source"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/datadogexporter/internal/hostmetadata/provider"
	ec2provider "github.com/open-telemetry/opentelemetry-collector-contrib/internal/metadataproviders/aws/ec2"
)

var (
	defaultPrefixes = [3]string{"ip-", "domu", "ec2amaz-"}
)

type HostInfo struct {
	InstanceID  string
	EC2Hostname string
	EC2Tags     []string
}

// isDefaultHostname checks if a hostname is an EC2 default
func isDefaultHostname(hostname string) bool {
	for _, val := range defaultPrefixes {
		if strings.HasPrefix(hostname, val) {
			return true
		}
	}

	return false
}

// GetHostInfo gets the hostname info from EC2 metadata
func GetHostInfo(logger *zap.Logger) (hostInfo *HostInfo) {
	sess, err := session.NewSession()
	hostInfo = &HostInfo{}

	if err != nil {
		logger.Warn("Failed to build AWS session", zap.Error(err))
		return
	}

	meta := ec2metadata.New(sess)

	if !meta.Available() {
		logger.Debug("EC2 Metadata not available")
		return
	}

	if idDoc, err := meta.GetInstanceIdentityDocument(); err == nil {
		hostInfo.InstanceID = idDoc.InstanceID
	} else {
		logger.Warn("Failed to get EC2 instance id document", zap.Error(err))
	}

	if ec2Hostname, err := meta.GetMetadata("hostname"); err == nil {
		hostInfo.EC2Hostname = ec2Hostname
	} else {
		logger.Warn("Failed to get EC2 hostname", zap.Error(err))
	}

	return
}

func (hi *HostInfo) GetHostname(logger *zap.Logger) string {
	if isDefaultHostname(hi.EC2Hostname) {
		return hi.InstanceID
	}

	return hi.EC2Hostname
}

var _ source.Provider = (*Provider)(nil)
var _ provider.ClusterNameProvider = (*Provider)(nil)

type Provider struct {
	once     sync.Once
	hostInfo HostInfo

	detector ec2provider.Provider
	logger   *zap.Logger
}

func NewProvider(logger *zap.Logger) (*Provider, error) {
	sess, err := session.NewSession()
	if err != nil {
		return nil, err
	}
	return &Provider{
		logger:   logger,
		detector: ec2provider.NewProvider(sess),
	}, nil
}

func (p *Provider) fillHostInfo() {
	p.once.Do(func() { p.hostInfo = *GetHostInfo(p.logger) })
}

func (p *Provider) Source(ctx context.Context) (source.Source, error) {
	p.fillHostInfo()
	if p.hostInfo.InstanceID == "" {
		return source.Source{}, fmt.Errorf("instance ID is unavailable")
	}

	return source.Source{Kind: source.HostnameKind, Identifier: p.hostInfo.InstanceID}, nil
}

// instanceTags gets the EC2 tags for the current instance.
func (p *Provider) instanceTags(ctx context.Context) (*ec2.DescribeTagsOutput, error) {
	// Get EC2 metadata to find the region and instance ID
	meta, err := p.detector.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get metadata: %w", err)
	}

	// Get the EC2 tags for the instance id.
	// Similar to:
	// - https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/39dbc1ac8/processor/resourcedetectionprocessor/internal/aws/ec2/ec2.go#L118-L151
	// - https://github.com/DataDog/datadog-agent/blob/1b4afdd6a03e8fabcc169b924931b2bb8935dab9/pkg/util/ec2/ec2_tags.go#L104-L134
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(meta.Region),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to build AWS session: %w", err)
	}

	svc := ec2.New(sess)
	return svc.DescribeTagsWithContext(ctx,
		&ec2.DescribeTagsInput{
			Filters: []*ec2.Filter{{
				Name: aws.String("resource-id"),
				Values: []*string{
					aws.String(meta.InstanceID),
				},
			}},
		})
}

// clusterNameFromTags gets the AWS EC2 Cluster name from the tags on an EC2 instance.
func clusterNameFromTags(ec2Tags *ec2.DescribeTagsOutput) (string, error) {
	// Similar to:
	// - https://github.com/DataDog/datadog-agent/blob/1b4afdd6a03/pkg/util/ec2/ec2.go#L256-L271
	const clusterNameTagPrefix = "kubernetes.io/cluster/"
	for _, tag := range ec2Tags.Tags {
		if strings.HasPrefix(*tag.Key, clusterNameTagPrefix) {
			if len(*tag.Key) == len(clusterNameTagPrefix) {
				return "", fmt.Errorf("missing cluster name in %q tag", *tag.Key)
			}
			return strings.Split(*tag.Key, "/")[2], nil
		}
	}

	return "", fmt.Errorf("no tag found with prefix %q", clusterNameTagPrefix)
}

// ClusterName gets the cluster name from an AWS EC2 machine.
func (p *Provider) ClusterName(ctx context.Context) (string, error) {
	ec2Tags, err := p.instanceTags(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get EC2 instance tags: %w", err)
	}
	return clusterNameFromTags(ec2Tags)
}

func (p *Provider) HostInfo() *HostInfo {
	p.fillHostInfo()
	return &p.hostInfo
}
