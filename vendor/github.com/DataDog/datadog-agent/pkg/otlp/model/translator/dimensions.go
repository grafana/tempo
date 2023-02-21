// Copyright  The OpenTelemetry Authors
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

package translator

import (
	"fmt"
	"sort"
	"strings"

	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/DataDog/datadog-agent/pkg/otlp/model/internal/utils"
)

const (
	dimensionSeparator = string(byte(0))
)

// Dimensions of a metric that identify a timeseries uniquely.
// This is similar to the concept of 'context' in DogStatsD/check metrics.
// NOTE: Keep this in sync with the TestDimensions struct.
type Dimensions struct {
	name     string
	tags     []string
	host     string
	originID string
}

// Name of the metric.
func (d *Dimensions) Name() string {
	return d.name
}

// Tags of the metric (read-only).
func (d *Dimensions) Tags() []string {
	return d.tags
}

// Host of the metric (may be empty).
func (d *Dimensions) Host() string {
	return d.host
}

// OriginID of the metric (may be empty).
func (d *Dimensions) OriginID() string {
	return d.originID
}

// getTags maps an attributeMap into a slice of Datadog tags
func getTags(labels pcommon.Map) []string {
	tags := make([]string, 0, labels.Len())
	labels.Range(func(key string, value pcommon.Value) bool {
		v := value.AsString()
		tags = append(tags, utils.FormatKeyValueTag(key, v))
		return true
	})
	return tags
}

// AddTags to metrics dimensions.
func (d *Dimensions) AddTags(tags ...string) *Dimensions {
	// defensively copy the tags
	newTags := make([]string, 0, len(tags)+len(d.tags))
	newTags = append(newTags, tags...)
	newTags = append(newTags, d.tags...)
	return &Dimensions{
		name:     d.name,
		tags:     newTags,
		host:     d.host,
		originID: d.originID,
	}
}

// WithAttributeMap creates a new metricDimensions struct with additional tags from attributes.
func (d *Dimensions) WithAttributeMap(labels pcommon.Map) *Dimensions {
	return d.AddTags(getTags(labels)...)
}

// WithSuffix creates a new dimensions struct with an extra name suffix.
func (d *Dimensions) WithSuffix(suffix string) *Dimensions {
	return &Dimensions{
		name:     fmt.Sprintf("%s.%s", d.name, suffix),
		host:     d.host,
		tags:     d.tags,
		originID: d.originID,
	}
}

// Uses a logic similar to what is done in the span processor to build metric keys:
// https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/b2327211df976e0a57ef0425493448988772a16b/processor/spanmetricsprocessor/processor.go#L353-L387
// TODO: make this a public util function?
func concatDimensionValue(metricKeyBuilder *strings.Builder, value string) {
	metricKeyBuilder.WriteString(value)
	metricKeyBuilder.WriteString(dimensionSeparator)
}

// String maps dimensions to a string to use as an identifier.
// The tags order does not matter.
func (d *Dimensions) String() string {
	var metricKeyBuilder strings.Builder

	dimensions := make([]string, len(d.tags))
	copy(dimensions, d.tags)

	dimensions = append(dimensions, fmt.Sprintf("name:%s", d.name))
	dimensions = append(dimensions, fmt.Sprintf("host:%s", d.host))
	dimensions = append(dimensions, fmt.Sprintf("originID:%s", d.originID))
	sort.Strings(dimensions)

	for _, dim := range dimensions {
		concatDimensionValue(&metricKeyBuilder, dim)
	}
	return metricKeyBuilder.String()
}
