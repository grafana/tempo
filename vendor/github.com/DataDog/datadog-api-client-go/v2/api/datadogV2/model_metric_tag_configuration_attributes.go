// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"time"
)

// MetricTagConfigurationAttributes Object containing the definition of a metric tag configuration attributes.
type MetricTagConfigurationAttributes struct {
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
	// Timestamp when the tag configuration was created.
	CreatedAt *time.Time `json:"created_at,omitempty"`
	// Toggle to include or exclude percentile aggregations for distribution metrics.
	// Only present when the `metric_type` is `distribution`.
	IncludePercentiles *bool `json:"include_percentiles,omitempty"`
	// The metric's type.
	MetricType *MetricTagConfigurationMetricTypes `json:"metric_type,omitempty"`
	// Timestamp when the tag configuration was last modified.
	ModifiedAt *time.Time `json:"modified_at,omitempty"`
	// List of tag keys on which to group.
	Tags []string `json:"tags,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewMetricTagConfigurationAttributes instantiates a new MetricTagConfigurationAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewMetricTagConfigurationAttributes() *MetricTagConfigurationAttributes {
	this := MetricTagConfigurationAttributes{}
	var metricType MetricTagConfigurationMetricTypes = METRICTAGCONFIGURATIONMETRICTYPES_GAUGE
	this.MetricType = &metricType
	return &this
}

// NewMetricTagConfigurationAttributesWithDefaults instantiates a new MetricTagConfigurationAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewMetricTagConfigurationAttributesWithDefaults() *MetricTagConfigurationAttributes {
	this := MetricTagConfigurationAttributes{}
	var metricType MetricTagConfigurationMetricTypes = METRICTAGCONFIGURATIONMETRICTYPES_GAUGE
	this.MetricType = &metricType
	return &this
}

// GetAggregations returns the Aggregations field value if set, zero value otherwise.
func (o *MetricTagConfigurationAttributes) GetAggregations() []MetricCustomAggregation {
	if o == nil || o.Aggregations == nil {
		var ret []MetricCustomAggregation
		return ret
	}
	return o.Aggregations
}

// GetAggregationsOk returns a tuple with the Aggregations field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MetricTagConfigurationAttributes) GetAggregationsOk() (*[]MetricCustomAggregation, bool) {
	if o == nil || o.Aggregations == nil {
		return nil, false
	}
	return &o.Aggregations, true
}

// HasAggregations returns a boolean if a field has been set.
func (o *MetricTagConfigurationAttributes) HasAggregations() bool {
	return o != nil && o.Aggregations != nil
}

// SetAggregations gets a reference to the given []MetricCustomAggregation and assigns it to the Aggregations field.
func (o *MetricTagConfigurationAttributes) SetAggregations(v []MetricCustomAggregation) {
	o.Aggregations = v
}

// GetCreatedAt returns the CreatedAt field value if set, zero value otherwise.
func (o *MetricTagConfigurationAttributes) GetCreatedAt() time.Time {
	if o == nil || o.CreatedAt == nil {
		var ret time.Time
		return ret
	}
	return *o.CreatedAt
}

// GetCreatedAtOk returns a tuple with the CreatedAt field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MetricTagConfigurationAttributes) GetCreatedAtOk() (*time.Time, bool) {
	if o == nil || o.CreatedAt == nil {
		return nil, false
	}
	return o.CreatedAt, true
}

// HasCreatedAt returns a boolean if a field has been set.
func (o *MetricTagConfigurationAttributes) HasCreatedAt() bool {
	return o != nil && o.CreatedAt != nil
}

// SetCreatedAt gets a reference to the given time.Time and assigns it to the CreatedAt field.
func (o *MetricTagConfigurationAttributes) SetCreatedAt(v time.Time) {
	o.CreatedAt = &v
}

// GetIncludePercentiles returns the IncludePercentiles field value if set, zero value otherwise.
func (o *MetricTagConfigurationAttributes) GetIncludePercentiles() bool {
	if o == nil || o.IncludePercentiles == nil {
		var ret bool
		return ret
	}
	return *o.IncludePercentiles
}

// GetIncludePercentilesOk returns a tuple with the IncludePercentiles field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MetricTagConfigurationAttributes) GetIncludePercentilesOk() (*bool, bool) {
	if o == nil || o.IncludePercentiles == nil {
		return nil, false
	}
	return o.IncludePercentiles, true
}

// HasIncludePercentiles returns a boolean if a field has been set.
func (o *MetricTagConfigurationAttributes) HasIncludePercentiles() bool {
	return o != nil && o.IncludePercentiles != nil
}

// SetIncludePercentiles gets a reference to the given bool and assigns it to the IncludePercentiles field.
func (o *MetricTagConfigurationAttributes) SetIncludePercentiles(v bool) {
	o.IncludePercentiles = &v
}

// GetMetricType returns the MetricType field value if set, zero value otherwise.
func (o *MetricTagConfigurationAttributes) GetMetricType() MetricTagConfigurationMetricTypes {
	if o == nil || o.MetricType == nil {
		var ret MetricTagConfigurationMetricTypes
		return ret
	}
	return *o.MetricType
}

// GetMetricTypeOk returns a tuple with the MetricType field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MetricTagConfigurationAttributes) GetMetricTypeOk() (*MetricTagConfigurationMetricTypes, bool) {
	if o == nil || o.MetricType == nil {
		return nil, false
	}
	return o.MetricType, true
}

// HasMetricType returns a boolean if a field has been set.
func (o *MetricTagConfigurationAttributes) HasMetricType() bool {
	return o != nil && o.MetricType != nil
}

// SetMetricType gets a reference to the given MetricTagConfigurationMetricTypes and assigns it to the MetricType field.
func (o *MetricTagConfigurationAttributes) SetMetricType(v MetricTagConfigurationMetricTypes) {
	o.MetricType = &v
}

// GetModifiedAt returns the ModifiedAt field value if set, zero value otherwise.
func (o *MetricTagConfigurationAttributes) GetModifiedAt() time.Time {
	if o == nil || o.ModifiedAt == nil {
		var ret time.Time
		return ret
	}
	return *o.ModifiedAt
}

// GetModifiedAtOk returns a tuple with the ModifiedAt field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MetricTagConfigurationAttributes) GetModifiedAtOk() (*time.Time, bool) {
	if o == nil || o.ModifiedAt == nil {
		return nil, false
	}
	return o.ModifiedAt, true
}

// HasModifiedAt returns a boolean if a field has been set.
func (o *MetricTagConfigurationAttributes) HasModifiedAt() bool {
	return o != nil && o.ModifiedAt != nil
}

// SetModifiedAt gets a reference to the given time.Time and assigns it to the ModifiedAt field.
func (o *MetricTagConfigurationAttributes) SetModifiedAt(v time.Time) {
	o.ModifiedAt = &v
}

// GetTags returns the Tags field value if set, zero value otherwise.
func (o *MetricTagConfigurationAttributes) GetTags() []string {
	if o == nil || o.Tags == nil {
		var ret []string
		return ret
	}
	return o.Tags
}

// GetTagsOk returns a tuple with the Tags field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MetricTagConfigurationAttributes) GetTagsOk() (*[]string, bool) {
	if o == nil || o.Tags == nil {
		return nil, false
	}
	return &o.Tags, true
}

// HasTags returns a boolean if a field has been set.
func (o *MetricTagConfigurationAttributes) HasTags() bool {
	return o != nil && o.Tags != nil
}

// SetTags gets a reference to the given []string and assigns it to the Tags field.
func (o *MetricTagConfigurationAttributes) SetTags(v []string) {
	o.Tags = v
}

// MarshalJSON serializes the struct using spec logic.
func (o MetricTagConfigurationAttributes) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Aggregations != nil {
		toSerialize["aggregations"] = o.Aggregations
	}
	if o.CreatedAt != nil {
		if o.CreatedAt.Nanosecond() == 0 {
			toSerialize["created_at"] = o.CreatedAt.Format("2006-01-02T15:04:05Z07:00")
		} else {
			toSerialize["created_at"] = o.CreatedAt.Format("2006-01-02T15:04:05.000Z07:00")
		}
	}
	if o.IncludePercentiles != nil {
		toSerialize["include_percentiles"] = o.IncludePercentiles
	}
	if o.MetricType != nil {
		toSerialize["metric_type"] = o.MetricType
	}
	if o.ModifiedAt != nil {
		if o.ModifiedAt.Nanosecond() == 0 {
			toSerialize["modified_at"] = o.ModifiedAt.Format("2006-01-02T15:04:05Z07:00")
		} else {
			toSerialize["modified_at"] = o.ModifiedAt.Format("2006-01-02T15:04:05.000Z07:00")
		}
	}
	if o.Tags != nil {
		toSerialize["tags"] = o.Tags
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *MetricTagConfigurationAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Aggregations       []MetricCustomAggregation          `json:"aggregations,omitempty"`
		CreatedAt          *time.Time                         `json:"created_at,omitempty"`
		IncludePercentiles *bool                              `json:"include_percentiles,omitempty"`
		MetricType         *MetricTagConfigurationMetricTypes `json:"metric_type,omitempty"`
		ModifiedAt         *time.Time                         `json:"modified_at,omitempty"`
		Tags               []string                           `json:"tags,omitempty"`
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
	if v := all.MetricType; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.Aggregations = all.Aggregations
	o.CreatedAt = all.CreatedAt
	o.IncludePercentiles = all.IncludePercentiles
	o.MetricType = all.MetricType
	o.ModifiedAt = all.ModifiedAt
	o.Tags = all.Tags
	return nil
}
