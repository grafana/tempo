// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package opencensus // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/opencensus"

import (
	"strconv"
	"time"

	occommon "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	ocresource "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	"go.opencensus.io/resource/resourcekeys"
	"go.opentelemetry.io/collector/pdata/pcommon"
	conventions "go.opentelemetry.io/collector/semconv/v1.6.1"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal/occonventions"
)

type ocInferredResourceType struct {
	// label presence to check against
	labelKeyPresent string
	// inferred resource type
	resourceType string
}

// mapping of label presence to inferred OC resource type
// NOTE: defined in the priority order (first match wins)
var labelPresenceToResourceType = []ocInferredResourceType{
	{
		// See https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/resource/semantic_conventions/container.md
		labelKeyPresent: conventions.AttributeContainerName,
		resourceType:    resourcekeys.ContainerType,
	},
	{
		// See https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/resource/semantic_conventions/k8s.md#pod
		labelKeyPresent: conventions.AttributeK8SPodName,
		// NOTE: OpenCensus is using "k8s" rather than "k8s.pod" for Pod
		resourceType: resourcekeys.K8SType,
	},
	{
		// See https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/resource/semantic_conventions/host.md
		labelKeyPresent: conventions.AttributeHostName,
		resourceType:    resourcekeys.HostType,
	},
	{
		// See https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/resource/semantic_conventions/cloud.md
		labelKeyPresent: conventions.AttributeCloudProvider,
		resourceType:    resourcekeys.CloudType,
	},
}

var langToOCLangCodeMap = getSDKLangToOCLangCodeMap()

func getSDKLangToOCLangCodeMap() map[string]int32 {
	mappings := make(map[string]int32)
	mappings[conventions.AttributeTelemetrySDKLanguageCPP] = 1
	mappings[conventions.AttributeTelemetrySDKLanguageDotnet] = 2
	mappings[conventions.AttributeTelemetrySDKLanguageErlang] = 3
	mappings[conventions.AttributeTelemetrySDKLanguageGo] = 4
	mappings[conventions.AttributeTelemetrySDKLanguageJava] = 5
	mappings[conventions.AttributeTelemetrySDKLanguageNodejs] = 6
	mappings[conventions.AttributeTelemetrySDKLanguagePHP] = 7
	mappings[conventions.AttributeTelemetrySDKLanguagePython] = 8
	mappings[conventions.AttributeTelemetrySDKLanguageRuby] = 9
	mappings[conventions.AttributeTelemetrySDKLanguageWebjs] = 10
	return mappings
}

func internalResourceToOC(resource pcommon.Resource) (*occommon.Node, *ocresource.Resource) {
	attrs := resource.Attributes()
	if attrs.Len() == 0 {
		return nil, nil
	}

	ocNode := &occommon.Node{}
	ocResource := &ocresource.Resource{}
	labels := make(map[string]string, attrs.Len())
	for k, v := range attrs.All() {
		val := v.AsString()

		switch k {
		case conventions.AttributeCloudAvailabilityZone:
			labels[resourcekeys.CloudKeyZone] = val
		case occonventions.AttributeResourceType:
			ocResource.Type = val
		case conventions.AttributeServiceName:
			getServiceInfo(ocNode).Name = val
		case occonventions.AttributeProcessStartTime:
			t, err := time.Parse(time.RFC3339Nano, val)
			if err != nil {
				continue
			}
			ts := timestamppb.New(t)
			getProcessIdentifier(ocNode).StartTimestamp = ts
		case conventions.AttributeHostName:
			getProcessIdentifier(ocNode).HostName = val
		case conventions.AttributeProcessPID:
			pid, err := strconv.ParseUint(val, 10, 32)
			if err == nil {
				getProcessIdentifier(ocNode).Pid = uint32(pid)
			}
		case conventions.AttributeTelemetrySDKVersion:
			getLibraryInfo(ocNode).CoreLibraryVersion = val
		case occonventions.AttributeExporterVersion:
			getLibraryInfo(ocNode).ExporterVersion = val
		case conventions.AttributeTelemetrySDKLanguage:
			if code, ok := langToOCLangCodeMap[val]; ok {
				getLibraryInfo(ocNode).Language = occommon.LibraryInfo_Language(code)
			}
		default:
			// Not a special attribute, put it into resource labels
			labels[k] = val
		}
	}
	ocResource.Labels = labels

	// If resource type is missing, try to infer it
	// based on the presence of resource labels (semantic conventions)
	if ocResource.Type == "" {
		if resType, ok := inferResourceType(ocResource.Labels); ok {
			ocResource.Type = resType
		}
	}

	return ocNode, ocResource
}

func getProcessIdentifier(ocNode *occommon.Node) *occommon.ProcessIdentifier {
	if ocNode.Identifier == nil {
		ocNode.Identifier = &occommon.ProcessIdentifier{}
	}
	return ocNode.Identifier
}

func getLibraryInfo(ocNode *occommon.Node) *occommon.LibraryInfo {
	if ocNode.LibraryInfo == nil {
		ocNode.LibraryInfo = &occommon.LibraryInfo{}
	}
	return ocNode.LibraryInfo
}

func getServiceInfo(ocNode *occommon.Node) *occommon.ServiceInfo {
	if ocNode.ServiceInfo == nil {
		ocNode.ServiceInfo = &occommon.ServiceInfo{}
	}
	return ocNode.ServiceInfo
}

func inferResourceType(labels map[string]string) (string, bool) {
	if labels == nil {
		return "", false
	}

	for _, mapping := range labelPresenceToResourceType {
		if _, ok := labels[mapping.labelKeyPresent]; ok {
			return mapping.resourceType, true
		}
	}

	return "", false
}
