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
	conventions "go.opentelemetry.io/otel/semconv/v1.6.1"
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
		labelKeyPresent: string(conventions.ContainerNameKey),
		resourceType:    resourcekeys.ContainerType,
	},
	{
		// See https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/resource/semantic_conventions/k8s.md#pod
		labelKeyPresent: string(conventions.K8SPodNameKey),
		// NOTE: OpenCensus is using "k8s" rather than "k8s.pod" for Pod
		resourceType: resourcekeys.K8SType,
	},
	{
		// See https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/resource/semantic_conventions/host.md
		labelKeyPresent: string(conventions.HostNameKey),
		resourceType:    resourcekeys.HostType,
	},
	{
		// See https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/resource/semantic_conventions/cloud.md
		labelKeyPresent: string(conventions.CloudProviderKey),
		resourceType:    resourcekeys.CloudType,
	},
}

var langToOCLangCodeMap = getSDKLangToOCLangCodeMap()

func getSDKLangToOCLangCodeMap() map[string]int32 {
	mappings := make(map[string]int32)
	mappings[conventions.TelemetrySDKLanguageCPP.Value.AsString()] = 1
	mappings[conventions.TelemetrySDKLanguageDotnet.Value.AsString()] = 2
	mappings[conventions.TelemetrySDKLanguageErlang.Value.AsString()] = 3
	mappings[conventions.TelemetrySDKLanguageGo.Value.AsString()] = 4
	mappings[conventions.TelemetrySDKLanguageJava.Value.AsString()] = 5
	mappings[conventions.TelemetrySDKLanguageNodejs.Value.AsString()] = 6
	mappings[conventions.TelemetrySDKLanguagePHP.Value.AsString()] = 7
	mappings[conventions.TelemetrySDKLanguagePython.Value.AsString()] = 8
	mappings[conventions.TelemetrySDKLanguageRuby.Value.AsString()] = 9
	mappings[conventions.TelemetrySDKLanguageWebjs.Value.AsString()] = 10
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
		case string(conventions.CloudAvailabilityZoneKey):
			labels[resourcekeys.CloudKeyZone] = val
		case occonventions.AttributeResourceType:
			ocResource.Type = val
		case string(conventions.ServiceNameKey):
			getServiceInfo(ocNode).Name = val
		case occonventions.AttributeProcessStartTime:
			t, err := time.Parse(time.RFC3339Nano, val)
			if err != nil {
				continue
			}
			ts := timestamppb.New(t)
			getProcessIdentifier(ocNode).StartTimestamp = ts
		case string(conventions.HostNameKey):
			getProcessIdentifier(ocNode).HostName = val
		case string(conventions.ProcessPIDKey):
			pid, err := strconv.ParseUint(val, 10, 32)
			if err == nil {
				getProcessIdentifier(ocNode).Pid = uint32(pid)
			}
		case string(conventions.TelemetrySDKVersionKey):
			getLibraryInfo(ocNode).CoreLibraryVersion = val
		case occonventions.AttributeExporterVersion:
			getLibraryInfo(ocNode).ExporterVersion = val
		case string(conventions.TelemetrySDKLanguageKey):
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
