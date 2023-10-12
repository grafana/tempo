// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package zipkin // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/zipkin/internal/zipkin"

import (
	"regexp"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

// These constants are the attribute keys used when translating from zipkin
// format to the internal collector data format.
const (
	StartTimeAbsent      = "otel.zipkin.absentField.startTime"
	TagServiceNameSource = "otlp.service.name.source"
)

var attrValDescriptions = []*attrValDescript{
	constructAttrValDescript("^$", pcommon.ValueTypeEmpty),
	constructAttrValDescript(`^-?\d+$`, pcommon.ValueTypeInt),
	constructAttrValDescript(`^-?\d+\.\d+$`, pcommon.ValueTypeDouble),
	constructAttrValDescript(`^(true|false)$`, pcommon.ValueTypeBool),
	constructAttrValDescript(`^\{"\w+":.+\}$`, pcommon.ValueTypeMap),
	constructAttrValDescript(`^\[.*\]$`, pcommon.ValueTypeSlice),
}

type attrValDescript struct {
	regex    *regexp.Regexp
	attrType pcommon.ValueType
}

func constructAttrValDescript(regex string, attrType pcommon.ValueType) *attrValDescript {
	regexc := regexp.MustCompile(regex)
	return &attrValDescript{
		regex:    regexc,
		attrType: attrType,
	}
}

// DetermineValueType returns the native OTLP attribute type the string translates to.
func DetermineValueType(value string) pcommon.ValueType {
	for _, desc := range attrValDescriptions {
		if desc.regex.MatchString(value) {
			return desc.attrType
		}
	}
	return pcommon.ValueTypeStr
}
