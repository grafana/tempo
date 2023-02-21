// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"time"
)

// MetricEstimateAttributes Object containing the definition of a metric estimate attribute.
type MetricEstimateAttributes struct {
	// Estimate type based on the queried configuration. By default, `count_or_gauge` is returned. `distribution` is returned for distribution metrics without percentiles enabled. Lastly, `percentile` is returned if `filter[pct]=true` is queried with a distribution metric.
	EstimateType *MetricEstimateType `json:"estimate_type,omitempty"`
	// Timestamp when the cardinality estimate was requested.
	EstimatedAt *time.Time `json:"estimated_at,omitempty"`
	// Estimated cardinality of the metric based on the queried configuration.
	EstimatedOutputSeries *int64 `json:"estimated_output_series,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewMetricEstimateAttributes instantiates a new MetricEstimateAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewMetricEstimateAttributes() *MetricEstimateAttributes {
	this := MetricEstimateAttributes{}
	var estimateType MetricEstimateType = METRICESTIMATETYPE_COUNT_OR_GAUGE
	this.EstimateType = &estimateType
	return &this
}

// NewMetricEstimateAttributesWithDefaults instantiates a new MetricEstimateAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewMetricEstimateAttributesWithDefaults() *MetricEstimateAttributes {
	this := MetricEstimateAttributes{}
	var estimateType MetricEstimateType = METRICESTIMATETYPE_COUNT_OR_GAUGE
	this.EstimateType = &estimateType
	return &this
}

// GetEstimateType returns the EstimateType field value if set, zero value otherwise.
func (o *MetricEstimateAttributes) GetEstimateType() MetricEstimateType {
	if o == nil || o.EstimateType == nil {
		var ret MetricEstimateType
		return ret
	}
	return *o.EstimateType
}

// GetEstimateTypeOk returns a tuple with the EstimateType field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MetricEstimateAttributes) GetEstimateTypeOk() (*MetricEstimateType, bool) {
	if o == nil || o.EstimateType == nil {
		return nil, false
	}
	return o.EstimateType, true
}

// HasEstimateType returns a boolean if a field has been set.
func (o *MetricEstimateAttributes) HasEstimateType() bool {
	return o != nil && o.EstimateType != nil
}

// SetEstimateType gets a reference to the given MetricEstimateType and assigns it to the EstimateType field.
func (o *MetricEstimateAttributes) SetEstimateType(v MetricEstimateType) {
	o.EstimateType = &v
}

// GetEstimatedAt returns the EstimatedAt field value if set, zero value otherwise.
func (o *MetricEstimateAttributes) GetEstimatedAt() time.Time {
	if o == nil || o.EstimatedAt == nil {
		var ret time.Time
		return ret
	}
	return *o.EstimatedAt
}

// GetEstimatedAtOk returns a tuple with the EstimatedAt field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MetricEstimateAttributes) GetEstimatedAtOk() (*time.Time, bool) {
	if o == nil || o.EstimatedAt == nil {
		return nil, false
	}
	return o.EstimatedAt, true
}

// HasEstimatedAt returns a boolean if a field has been set.
func (o *MetricEstimateAttributes) HasEstimatedAt() bool {
	return o != nil && o.EstimatedAt != nil
}

// SetEstimatedAt gets a reference to the given time.Time and assigns it to the EstimatedAt field.
func (o *MetricEstimateAttributes) SetEstimatedAt(v time.Time) {
	o.EstimatedAt = &v
}

// GetEstimatedOutputSeries returns the EstimatedOutputSeries field value if set, zero value otherwise.
func (o *MetricEstimateAttributes) GetEstimatedOutputSeries() int64 {
	if o == nil || o.EstimatedOutputSeries == nil {
		var ret int64
		return ret
	}
	return *o.EstimatedOutputSeries
}

// GetEstimatedOutputSeriesOk returns a tuple with the EstimatedOutputSeries field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MetricEstimateAttributes) GetEstimatedOutputSeriesOk() (*int64, bool) {
	if o == nil || o.EstimatedOutputSeries == nil {
		return nil, false
	}
	return o.EstimatedOutputSeries, true
}

// HasEstimatedOutputSeries returns a boolean if a field has been set.
func (o *MetricEstimateAttributes) HasEstimatedOutputSeries() bool {
	return o != nil && o.EstimatedOutputSeries != nil
}

// SetEstimatedOutputSeries gets a reference to the given int64 and assigns it to the EstimatedOutputSeries field.
func (o *MetricEstimateAttributes) SetEstimatedOutputSeries(v int64) {
	o.EstimatedOutputSeries = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o MetricEstimateAttributes) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.EstimateType != nil {
		toSerialize["estimate_type"] = o.EstimateType
	}
	if o.EstimatedAt != nil {
		if o.EstimatedAt.Nanosecond() == 0 {
			toSerialize["estimated_at"] = o.EstimatedAt.Format("2006-01-02T15:04:05Z07:00")
		} else {
			toSerialize["estimated_at"] = o.EstimatedAt.Format("2006-01-02T15:04:05.000Z07:00")
		}
	}
	if o.EstimatedOutputSeries != nil {
		toSerialize["estimated_output_series"] = o.EstimatedOutputSeries
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *MetricEstimateAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		EstimateType          *MetricEstimateType `json:"estimate_type,omitempty"`
		EstimatedAt           *time.Time          `json:"estimated_at,omitempty"`
		EstimatedOutputSeries *int64              `json:"estimated_output_series,omitempty"`
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
	if v := all.EstimateType; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.EstimateType = all.EstimateType
	o.EstimatedAt = all.EstimatedAt
	o.EstimatedOutputSeries = all.EstimatedOutputSeries
	return nil
}
