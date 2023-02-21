// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// MetricTagConfigurationCreateAttributes Object containing the definition of a metric tag configuration to be created.
type MetricTagConfigurationCreateAttributes struct {
	// A list of queryable aggregation combinations for a count, rate, or gauge metric.
	// By default, count and rate metrics require the (time: sum, space: sum) aggregation and
	// Gauge metrics require the (time: avg, space: avg) aggregation.
	// Additional time & space combinations are also available:
	//
	// - time: avg, space: avg
	// - time: avg, space: max
	// - time: avg, space: min
	// - time: avg, space: sum
	// - time: count, space: sum
	// - time: max, space: max
	// - time: min, space: min
	// - time: sum, space: avg
	// - time: sum, space: sum
	//
	// Can only be applied to metrics that have a `metric_type` of `count`, `rate`, or `gauge`.
	Aggregations []MetricCustomAggregation `json:"aggregations,omitempty"`
	// Toggle to include/exclude percentiles for a distribution metric.
	// Defaults to false. Can only be applied to metrics that have a `metric_type` of `distribution`.
	IncludePercentiles *bool `json:"include_percentiles,omitempty"`
	// The metric's type.
	MetricType MetricTagConfigurationMetricTypes `json:"metric_type"`
	// A list of tag keys that will be queryable for your metric.
	Tags []string `json:"tags"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewMetricTagConfigurationCreateAttributes instantiates a new MetricTagConfigurationCreateAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewMetricTagConfigurationCreateAttributes(metricType MetricTagConfigurationMetricTypes, tags []string) *MetricTagConfigurationCreateAttributes {
	this := MetricTagConfigurationCreateAttributes{}
	this.MetricType = metricType
	this.Tags = tags
	return &this
}

// NewMetricTagConfigurationCreateAttributesWithDefaults instantiates a new MetricTagConfigurationCreateAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewMetricTagConfigurationCreateAttributesWithDefaults() *MetricTagConfigurationCreateAttributes {
	this := MetricTagConfigurationCreateAttributes{}
	var metricType MetricTagConfigurationMetricTypes = METRICTAGCONFIGURATIONMETRICTYPES_GAUGE
	this.MetricType = metricType
	return &this
}

// GetAggregations returns the Aggregations field value if set, zero value otherwise.
func (o *MetricTagConfigurationCreateAttributes) GetAggregations() []MetricCustomAggregation {
	if o == nil || o.Aggregations == nil {
		var ret []MetricCustomAggregation
		return ret
	}
	return o.Aggregations
}

// GetAggregationsOk returns a tuple with the Aggregations field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MetricTagConfigurationCreateAttributes) GetAggregationsOk() (*[]MetricCustomAggregation, bool) {
	if o == nil || o.Aggregations == nil {
		return nil, false
	}
	return &o.Aggregations, true
}

// HasAggregations returns a boolean if a field has been set.
func (o *MetricTagConfigurationCreateAttributes) HasAggregations() bool {
	return o != nil && o.Aggregations != nil
}

// SetAggregations gets a reference to the given []MetricCustomAggregation and assigns it to the Aggregations field.
func (o *MetricTagConfigurationCreateAttributes) SetAggregations(v []MetricCustomAggregation) {
	o.Aggregations = v
}

// GetIncludePercentiles returns the IncludePercentiles field value if set, zero value otherwise.
func (o *MetricTagConfigurationCreateAttributes) GetIncludePercentiles() bool {
	if o == nil || o.IncludePercentiles == nil {
		var ret bool
		return ret
	}
	return *o.IncludePercentiles
}

// GetIncludePercentilesOk returns a tuple with the IncludePercentiles field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MetricTagConfigurationCreateAttributes) GetIncludePercentilesOk() (*bool, bool) {
	if o == nil || o.IncludePercentiles == nil {
		return nil, false
	}
	return o.IncludePercentiles, true
}

// HasIncludePercentiles returns a boolean if a field has been set.
func (o *MetricTagConfigurationCreateAttributes) HasIncludePercentiles() bool {
	return o != nil && o.IncludePercentiles != nil
}

// SetIncludePercentiles gets a reference to the given bool and assigns it to the IncludePercentiles field.
func (o *MetricTagConfigurationCreateAttributes) SetIncludePercentiles(v bool) {
	o.IncludePercentiles = &v
}

// GetMetricType returns the MetricType field value.
func (o *MetricTagConfigurationCreateAttributes) GetMetricType() MetricTagConfigurationMetricTypes {
	if o == nil {
		var ret MetricTagConfigurationMetricTypes
		return ret
	}
	return o.MetricType
}

// GetMetricTypeOk returns a tuple with the MetricType field value
// and a boolean to check if the value has been set.
func (o *MetricTagConfigurationCreateAttributes) GetMetricTypeOk() (*MetricTagConfigurationMetricTypes, bool) {
	if o == nil {
		return nil, false
	}
	return &o.MetricType, true
}

// SetMetricType sets field value.
func (o *MetricTagConfigurationCreateAttributes) SetMetricType(v MetricTagConfigurationMetricTypes) {
	o.MetricType = v
}

// GetTags returns the Tags field value.
func (o *MetricTagConfigurationCreateAttributes) GetTags() []string {
	if o == nil {
		var ret []string
		return ret
	}
	return o.Tags
}

// GetTagsOk returns a tuple with the Tags field value
// and a boolean to check if the value has been set.
func (o *MetricTagConfigurationCreateAttributes) GetTagsOk() (*[]string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Tags, true
}

// SetTags sets field value.
func (o *MetricTagConfigurationCreateAttributes) SetTags(v []string) {
	o.Tags = v
}

// MarshalJSON serializes the struct using spec logic.
func (o MetricTagConfigurationCreateAttributes) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Aggregations != nil {
		toSerialize["aggregations"] = o.Aggregations
	}
	if o.IncludePercentiles != nil {
		toSerialize["include_percentiles"] = o.IncludePercentiles
	}
	toSerialize["metric_type"] = o.MetricType
	toSerialize["tags"] = o.Tags

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *MetricTagConfigurationCreateAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		MetricType *MetricTagConfigurationMetricTypes `json:"metric_type"`
		Tags       *[]string                          `json:"tags"`
	}{}
	all := struct {
		Aggregations       []MetricCustomAggregation         `json:"aggregations,omitempty"`
		IncludePercentiles *bool                             `json:"include_percentiles,omitempty"`
		MetricType         MetricTagConfigurationMetricTypes `json:"metric_type"`
		Tags               []string                          `json:"tags"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.MetricType == nil {
		return fmt.Errorf("required field metric_type missing")
	}
	if required.Tags == nil {
		return fmt.Errorf("required field tags missing")
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
	if v := all.MetricType; !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.Aggregations = all.Aggregations
	o.IncludePercentiles = all.IncludePercentiles
	o.MetricType = all.MetricType
	o.Tags = all.Tags
	return nil
}
