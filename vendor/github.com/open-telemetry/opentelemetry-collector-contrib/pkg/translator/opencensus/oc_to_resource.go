// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package opencensus // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/opencensus"

import (
	"time"

	occommon "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	ocresource "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	"go.opencensus.io/resource/resourcekeys"
	"go.opentelemetry.io/collector/model/pdata"
	conventions "go.opentelemetry.io/collector/model/semconv/v1.6.1"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal/occonventions"
)

var ocLangCodeToLangMap = getOCLangCodeToLangMap()

func getOCLangCodeToLangMap() map[occommon.LibraryInfo_Language]string {
	mappings := make(map[occommon.LibraryInfo_Language]string)
	mappings[1] = conventions.AttributeTelemetrySDKLanguageCPP
	mappings[2] = conventions.AttributeTelemetrySDKLanguageDotnet
	mappings[3] = conventions.AttributeTelemetrySDKLanguageErlang
	mappings[4] = conventions.AttributeTelemetrySDKLanguageGo
	mappings[5] = conventions.AttributeTelemetrySDKLanguageJava
	mappings[6] = conventions.AttributeTelemetrySDKLanguageNodejs
	mappings[7] = conventions.AttributeTelemetrySDKLanguagePHP
	mappings[8] = conventions.AttributeTelemetrySDKLanguagePython
	mappings[9] = conventions.AttributeTelemetrySDKLanguageRuby
	mappings[10] = conventions.AttributeTelemetrySDKLanguageWebjs
	return mappings
}

func ocNodeResourceToInternal(ocNode *occommon.Node, ocResource *ocresource.Resource, dest pdata.Resource) {
	if ocNode == nil && ocResource == nil {
		return
	}

	// Number of special fields in OC that will be translated to Attributes
	const serviceInfoAttrCount = 1     // Number of Node.ServiceInfo fields.
	const nodeIdentifierAttrCount = 3  // Number of Node.Identifier fields.
	const libraryInfoAttrCount = 3     // Number of Node.LibraryInfo fields.
	const specialResourceAttrCount = 1 // Number of Resource fields.

	// Calculate maximum total number of attributes for capacity reservation.
	maxTotalAttrCount := 0
	if ocNode != nil {
		maxTotalAttrCount += len(ocNode.Attributes)
		if ocNode.ServiceInfo != nil {
			maxTotalAttrCount += serviceInfoAttrCount
		}
		if ocNode.Identifier != nil {
			maxTotalAttrCount += nodeIdentifierAttrCount
		}
		if ocNode.LibraryInfo != nil {
			maxTotalAttrCount += libraryInfoAttrCount
		}
	}
	if ocResource != nil {
		maxTotalAttrCount += len(ocResource.Labels)
		if ocResource.Type != "" {
			maxTotalAttrCount += specialResourceAttrCount
		}
	}

	// There are no attributes to be set.
	if maxTotalAttrCount == 0 {
		return
	}

	attrs := dest.Attributes()
	attrs.Clear()
	attrs.EnsureCapacity(maxTotalAttrCount)

	if ocNode != nil {
		// Copy all Attributes.
		for k, v := range ocNode.Attributes {
			attrs.InsertString(k, v)
		}

		// Add all special fields.
		if ocNode.ServiceInfo != nil {
			if ocNode.ServiceInfo.Name != "" {
				attrs.UpsertString(conventions.AttributeServiceName, ocNode.ServiceInfo.Name)
			}
		}
		if ocNode.Identifier != nil {
			if ocNode.Identifier.StartTimestamp != nil {
				attrs.UpsertString(occonventions.AttributeProcessStartTime, ocNode.Identifier.StartTimestamp.AsTime().Format(time.RFC3339Nano))
			}
			if ocNode.Identifier.HostName != "" {
				attrs.UpsertString(conventions.AttributeHostName, ocNode.Identifier.HostName)
			}
			if ocNode.Identifier.Pid != 0 {
				attrs.UpsertInt(conventions.AttributeProcessPID, int64(ocNode.Identifier.Pid))
			}
		}
		if ocNode.LibraryInfo != nil {
			if ocNode.LibraryInfo.CoreLibraryVersion != "" {
				attrs.UpsertString(conventions.AttributeTelemetrySDKVersion, ocNode.LibraryInfo.CoreLibraryVersion)
			}
			if ocNode.LibraryInfo.ExporterVersion != "" {
				attrs.UpsertString(occonventions.AttributeExporterVersion, ocNode.LibraryInfo.ExporterVersion)
			}
			if ocNode.LibraryInfo.Language != occommon.LibraryInfo_LANGUAGE_UNSPECIFIED {
				if str, ok := ocLangCodeToLangMap[ocNode.LibraryInfo.Language]; ok {
					attrs.UpsertString(conventions.AttributeTelemetrySDKLanguage, str)
				}
			}
		}
	}

	if ocResource != nil {
		// Copy resource Labels.
		for k, v := range ocResource.Labels {
			switch k {
			case resourcekeys.CloudKeyZone:
				attrs.InsertString(conventions.AttributeCloudAvailabilityZone, v)
			default:
				attrs.InsertString(k, v)
			}
		}
		// Add special fields.
		if ocResource.Type != "" {
			attrs.UpsertString(occonventions.AttributeResourceType, ocResource.Type)
		}
	}
}
