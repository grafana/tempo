// Copyright  OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package attributes

import (
	"go.opentelemetry.io/collector/pdata/pcommon"
	conventions "go.opentelemetry.io/collector/semconv/v1.6.1"

	"github.com/DataDog/datadog-agent/pkg/otlp/model/attributes/azure"
	"github.com/DataDog/datadog-agent/pkg/otlp/model/attributes/ec2"
	"github.com/DataDog/datadog-agent/pkg/otlp/model/attributes/gcp"
	"github.com/DataDog/datadog-agent/pkg/otlp/model/source"
)

const (
	// AttributeDatadogHostname the datadog host name attribute
	AttributeDatadogHostname = "datadog.host.name"
	// AttributeK8sNodeName the datadog k8s node name attribute
	AttributeK8sNodeName = "k8s.node.name"
)

func getClusterName(attrs pcommon.Map) (string, bool) {
	if k8sClusterName, ok := attrs.Get(conventions.AttributeK8SClusterName); ok {
		return k8sClusterName.Str(), true
	}

	cloudProvider, ok := attrs.Get(conventions.AttributeCloudProvider)
	if ok && cloudProvider.Str() == conventions.AttributeCloudProviderAzure {
		return azure.ClusterNameFromAttributes(attrs)
	} else if ok && cloudProvider.Str() == conventions.AttributeCloudProviderAWS {
		return ec2.ClusterNameFromAttributes(attrs)
	}

	return "", false
}

// HostnameFromAttributes tries to get a valid hostname from attributes by checking, in order:
//
//  1. a custom Datadog hostname provided by the "datadog.host.name" attribute
//
//  2. the Kubernetes node name (and cluster name if available),
//
//  3. cloud provider specific hostname for AWS or GCP
//
//  4. the container ID,
//
//  5. the cloud provider host ID and
//
//  6. the host.name attribute.
//
//     It returns a boolean value indicated if any name was found
func hostnameFromAttributes(attrs pcommon.Map, usePreviewRules bool) (string, bool) {
	// Check if the host is localhost or 0.0.0.0, if so discard it.
	// We don't do the more strict validation done for metadata,
	// to avoid breaking users existing invalid-but-accepted hostnames.
	var invalidHosts = map[string]struct{}{
		"0.0.0.0":                 {},
		"127.0.0.1":               {},
		"localhost":               {},
		"localhost.localdomain":   {},
		"localhost6.localdomain6": {},
		"ip6-localhost":           {},
	}

	candidateHost, ok := unsanitizedHostnameFromAttributes(attrs, usePreviewRules)
	if _, invalid := invalidHosts[candidateHost]; invalid {
		return "", false
	}
	return candidateHost, ok
}

func k8sHostnameFromAttributes(attrs pcommon.Map) (string, bool) {
	node, ok := attrs.Get(AttributeK8sNodeName)
	if !ok {
		return "", false
	}

	if cluster, ok := getClusterName(attrs); ok {
		return node.Str() + "-" + cluster, true
	}
	return node.Str(), true
}

func unsanitizedHostnameFromAttributes(attrs pcommon.Map, usePreviewRules bool) (string, bool) {
	// Custom hostname: useful for overriding in k8s/cloud envs
	if customHostname, ok := attrs.Get(AttributeDatadogHostname); ok {
		return customHostname.Str(), true
	}

	if launchType, ok := attrs.Get(conventions.AttributeAWSECSLaunchtype); ok && launchType.Str() == conventions.AttributeAWSECSLaunchtypeFargate {
		// If on AWS ECS Fargate, we don't have a hostname
		return "", false
	}

	// Kubernetes: node-cluster if cluster name is available, else node
	k8sName, k8sOk := k8sHostnameFromAttributes(attrs)

	// If not using the preview rules, return the Kubernetes node name
	// before cloud provider names to preserve the current behavior.
	if !usePreviewRules && k8sOk {
		return k8sName, true
	}

	cloudProvider, ok := attrs.Get(conventions.AttributeCloudProvider)
	if ok && cloudProvider.Str() == conventions.AttributeCloudProviderAWS {
		return ec2.HostnameFromAttributes(attrs, usePreviewRules)
	} else if ok && cloudProvider.Str() == conventions.AttributeCloudProviderGCP {
		return gcp.HostnameFromAttributes(attrs, usePreviewRules)
	} else if ok && cloudProvider.Str() == conventions.AttributeCloudProviderAzure {
		return azure.HostnameFromAttributes(attrs, usePreviewRules)
	}

	// If using the preview rules, the cloud provider names take precedence.
	// This is to report the same hostname as Datadog cloud integrations.
	if usePreviewRules && k8sOk {
		return k8sName, true
	}

	// host id from cloud provider
	if hostID, ok := attrs.Get(conventions.AttributeHostID); ok {
		return hostID.Str(), true
	}

	// hostname from cloud provider or OS
	if hostName, ok := attrs.Get(conventions.AttributeHostName); ok {
		return hostName.Str(), true
	}

	if !usePreviewRules {
		// container id (e.g. from Docker)
		if containerID, ok := attrs.Get(conventions.AttributeContainerID); ok {
			return containerID.Str(), true
		}
	}

	return "", false
}

// SourceFromAttributes gets a telemetry signal source from its attributes.
func SourceFromAttributes(attrs pcommon.Map, usePreviewRules bool) (source.Source, bool) {
	if launchType, ok := attrs.Get(conventions.AttributeAWSECSLaunchtype); ok && launchType.Str() == conventions.AttributeAWSECSLaunchtypeFargate {
		if taskARN, ok := attrs.Get(conventions.AttributeAWSECSTaskARN); ok {
			return source.Source{Kind: source.AWSECSFargateKind, Identifier: taskARN.Str()}, true
		}
	}

	if host, ok := hostnameFromAttributes(attrs, usePreviewRules); ok {
		return source.Source{Kind: source.HostnameKind, Identifier: host}, true
	}

	return source.Source{}, false
}
