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

package ec2

import (
	"fmt"
	"strings"

	"go.opentelemetry.io/collector/pdata/pcommon"
	conventions "go.opentelemetry.io/collector/semconv/v1.6.1"
)

var (
	defaultPrefixes  = [3]string{"ip-", "domu", "ec2amaz-"}
	ec2TagPrefix     = "ec2.tag."
	clusterTagPrefix = ec2TagPrefix + "kubernetes.io/cluster/"
)

// HostInfo holds the EC2 host information.
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

// HostnameFromAttributes gets a valid hostname from labels
// if available
func HostnameFromAttributes(attrs pcommon.Map, usePreviewRules bool) (string, bool) {
	hostName, ok := attrs.Get(conventions.AttributeHostName)
	// With hostname preview rules, return the EC2 instance id always.
	if !usePreviewRules && ok && !isDefaultHostname(hostName.Str()) {
		return hostName.Str(), true
	}

	if hostID, ok := attrs.Get(conventions.AttributeHostID); ok {
		return hostID.Str(), true
	}

	return "", false
}

// HostInfoFromAttributes gets EC2 host info from attributes following
// OpenTelemetry semantic conventions
func HostInfoFromAttributes(attrs pcommon.Map) (hostInfo *HostInfo) {
	hostInfo = &HostInfo{}

	if hostID, ok := attrs.Get(conventions.AttributeHostID); ok {
		hostInfo.InstanceID = hostID.Str()
	}

	if hostName, ok := attrs.Get(conventions.AttributeHostName); ok {
		hostInfo.EC2Hostname = hostName.Str()
	}

	attrs.Range(func(k string, v pcommon.Value) bool {
		if strings.HasPrefix(k, ec2TagPrefix) {
			tag := fmt.Sprintf("%s:%s", strings.TrimPrefix(k, ec2TagPrefix), v.Str())
			hostInfo.EC2Tags = append(hostInfo.EC2Tags, tag)
		}
		return true
	})

	return
}

// ClusterNameFromAttributes gets the AWS cluster name from attributes
func ClusterNameFromAttributes(attrs pcommon.Map) (clusterName string, ok bool) {
	// Get cluster name from tag keys
	// https://github.com/DataDog/datadog-agent/blob/1c94b11/pkg/util/ec2/ec2.go#L238
	attrs.Range(func(k string, _ pcommon.Value) bool {
		if strings.HasPrefix(k, clusterTagPrefix) {
			clusterName = strings.Split(k, "/")[2]
			ok = true
		}
		return true
	})
	return
}
