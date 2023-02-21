// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// LogsMetricCompute The compute rule to compute the log-based metric.
type LogsMetricCompute struct {
	// The type of aggregation to use.
	AggregationType LogsMetricComputeAggregationType `json:"aggregation_type"`
	// Toggle to include or exclude percentile aggregations for distribution metrics.
	// Only present when the `aggregation_type` is `distribution`.
	IncludePercentiles *bool `json:"include_percentiles,omitempty"`
	// The path to the value the log-based metric will aggregate on (only used if the aggregation type is a "distribution").
	Path *string `json:"path,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewLogsMetricCompute instantiates a new LogsMetricCompute object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewLogsMetricCompute(aggregationType LogsMetricComputeAggregationType) *LogsMetricCompute {
	this := LogsMetricCompute{}
	this.AggregationType = aggregationType
	return &this
}

// NewLogsMetricComputeWithDefaults instantiates a new LogsMetricCompute object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewLogsMetricComputeWithDefaults() *LogsMetricCompute {
	this := LogsMetricCompute{}
	return &this
}

// GetAggregationType returns the AggregationType field value.
func (o *LogsMetricCompute) GetAggregationType() LogsMetricComputeAggregationType {
	if o == nil {
		var ret LogsMetricComputeAggregationType
		return ret
	}
	return o.AggregationType
}

// GetAggregationTypeOk returns a tuple with the AggregationType field value
// and a boolean to check if the value has been set.
func (o *LogsMetricCompute) GetAggregationTypeOk() (*LogsMetricComputeAggregationType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.AggregationType, true
}

// SetAggregationType sets field value.
func (o *LogsMetricCompute) SetAggregationType(v LogsMetricComputeAggregationType) {
	o.AggregationType = v
}

// GetIncludePercentiles returns the IncludePercentiles field value if set, zero value otherwise.
func (o *LogsMetricCompute) GetIncludePercentiles() bool {
	if o == nil || o.IncludePercentiles == nil {
		var ret bool
		return ret
	}
	return *o.IncludePercentiles
}

// GetIncludePercentilesOk returns a tuple with the IncludePercentiles field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsMetricCompute) GetIncludePercentilesOk() (*bool, bool) {
	if o == nil || o.IncludePercentiles == nil {
		return nil, false
	}
	return o.IncludePercentiles, true
}

// HasIncludePercentiles returns a boolean if a field has been set.
func (o *LogsMetricCompute) HasIncludePercentiles() bool {
	return o != nil && o.IncludePercentiles != nil
}

// SetIncludePercentiles gets a reference to the given bool and assigns it to the IncludePercentiles field.
func (o *LogsMetricCompute) SetIncludePercentiles(v bool) {
	o.IncludePercentiles = &v
}

// GetPath returns the Path field value if set, zero value otherwise.
func (o *LogsMetricCompute) GetPath() string {
	if o == nil || o.Path == nil {
		var ret string
		return ret
	}
	return *o.Path
}

// GetPathOk returns a tuple with the Path field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsMetricCompute) GetPathOk() (*string, bool) {
	if o == nil || o.Path == nil {
		return nil, false
	}
	return o.Path, true
}

// HasPath returns a boolean if a field has been set.
func (o *LogsMetricCompute) HasPath() bool {
	return o != nil && o.Path != nil
}

// SetPath gets a reference to the given string and assigns it to the Path field.
func (o *LogsMetricCompute) SetPath(v string) {
	o.Path = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o LogsMetricCompute) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["aggregation_type"] = o.AggregationType
	if o.IncludePercentiles != nil {
		toSerialize["include_percentiles"] = o.IncludePercentiles
	}
	if o.Path != nil {
		toSerialize["path"] = o.Path
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *LogsMetricCompute) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		AggregationType *LogsMetricComputeAggregationType `json:"aggregation_type"`
	}{}
	all := struct {
		AggregationType    LogsMetricComputeAggregationType `json:"aggregation_type"`
		IncludePercentiles *bool                            `json:"include_percentiles,omitempty"`
		Path               *string                          `json:"path,omitempty"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.AggregationType == nil {
		return fmt.Errorf("required field aggregation_type missing")
	}
	err = json.Unmarshal(bytes, &all)
	if err != nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	if v := all.AggregationType; !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.AggregationType = all.AggregationType
	o.IncludePercentiles = all.IncludePercentiles
	o.Path = all.Path
	return nil
}
