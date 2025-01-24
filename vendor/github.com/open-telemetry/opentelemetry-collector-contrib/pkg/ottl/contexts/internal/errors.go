// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal"

import (
	"fmt"
	"strings"
)

const (
	DefaultErrorMessage = "segment %q from path %q is not a valid path nor a valid OTTL keyword for the %v context - review %v to see all valid paths"

	ResourceContextRef      = "https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/pkg/ottl/contexts/ottlresource"
	InstrumentationScopeRef = "https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/pkg/ottl/contexts/ottlscope"
	SpanRef                 = "https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/pkg/ottl/contexts/ottlspan"
	SpanEventRef            = "https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/pkg/ottl/contexts/ottlspanevent"
	MetricRef               = "https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/pkg/ottl/contexts/ottlmetric"
	DataPointRef            = "https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/pkg/ottl/contexts/ottldatapoint"
	LogRef                  = "https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/pkg/ottl/contexts/ottllog"
)

func FormatDefaultErrorMessage(pathSegment, fullPath, context, ref string) error {
	return fmt.Errorf(DefaultErrorMessage, pathSegment, fullPath, context, ref)
}

func FormatCacheErrorMessage(lowerContext, pathContext, fullPath string) error {
	pathSuggestion := strings.Replace(fullPath, pathContext+".", lowerContext+".", 1)
	return fmt.Errorf(`access to cache must be performed using the same statement's context, please replace "%s" with "%s"`, fullPath, pathSuggestion)
}
