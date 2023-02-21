// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// LogsMetricUpdateCompute The compute rule to compute the log-based metric.
type LogsMetricUpdateCompute struct {
	// Toggle to include or exclude percentile aggregations for distribution metrics.
	// Only present when the `aggregation_type` is `distribution`.
	IncludePercentiles *bool `json:"include_percentiles,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewLogsMetricUpdateCompute instantiates a new LogsMetricUpdateCompute object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewLogsMetricUpdateCompute() *LogsMetricUpdateCompute {
	this := LogsMetricUpdateCompute{}
	return &this
}

// NewLogsMetricUpdateComputeWithDefaults instantiates a new LogsMetricUpdateCompute object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewLogsMetricUpdateComputeWithDefaults() *LogsMetricUpdateCompute {
	this := LogsMetricUpdateCompute{}
	return &this
}

// GetIncludePercentiles returns the IncludePercentiles field value if set, zero value otherwise.
func (o *LogsMetricUpdateCompute) GetIncludePercentiles() bool {
	if o == nil || o.IncludePercentiles == nil {
		var ret bool
		return ret
	}
	return *o.IncludePercentiles
}

// GetIncludePercentilesOk returns a tuple with the IncludePercentiles field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsMetricUpdateCompute) GetIncludePercentilesOk() (*bool, bool) {
	if o == nil || o.IncludePercentiles == nil {
		return nil, false
	}
	return o.IncludePercentiles, true
}

// HasIncludePercentiles returns a boolean if a field has been set.
func (o *LogsMetricUpdateCompute) HasIncludePercentiles() bool {
	return o != nil && o.IncludePercentiles != nil
}

// SetIncludePercentiles gets a reference to the given bool and assigns it to the IncludePercentiles field.
func (o *LogsMetricUpdateCompute) SetIncludePercentiles(v bool) {
	o.IncludePercentiles = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o LogsMetricUpdateCompute) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.IncludePercentiles != nil {
		toSerialize["include_percentiles"] = o.IncludePercentiles
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *LogsMetricUpdateCompute) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		IncludePercentiles *bool `json:"include_percentiles,omitempty"`
	}{}
	err = json.Unmarshal(bytes, &all)
	if err != nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.IncludePercentiles = all.IncludePercentiles
	return nil
}
