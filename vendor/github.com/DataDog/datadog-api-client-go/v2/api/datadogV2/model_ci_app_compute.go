// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// CIAppCompute A compute rule to compute metrics or timeseries.
type CIAppCompute struct {
	// An aggregation function.
	Aggregation CIAppAggregationFunction `json:"aggregation"`
	// The time buckets' size (only used for type=timeseries)
	// Defaults to a resolution of 150 points.
	Interval *string `json:"interval,omitempty"`
	// The metric to use.
	Metric *string `json:"metric,omitempty"`
	// The type of compute.
	Type *CIAppComputeType `json:"type,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewCIAppCompute instantiates a new CIAppCompute object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewCIAppCompute(aggregation CIAppAggregationFunction) *CIAppCompute {
	this := CIAppCompute{}
	this.Aggregation = aggregation
	var typeVar CIAppComputeType = CIAPPCOMPUTETYPE_TOTAL
	this.Type = &typeVar
	return &this
}

// NewCIAppComputeWithDefaults instantiates a new CIAppCompute object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewCIAppComputeWithDefaults() *CIAppCompute {
	this := CIAppCompute{}
	var typeVar CIAppComputeType = CIAPPCOMPUTETYPE_TOTAL
	this.Type = &typeVar
	return &this
}

// GetAggregation returns the Aggregation field value.
func (o *CIAppCompute) GetAggregation() CIAppAggregationFunction {
	if o == nil {
		var ret CIAppAggregationFunction
		return ret
	}
	return o.Aggregation
}

// GetAggregationOk returns a tuple with the Aggregation field value
// and a boolean to check if the value has been set.
func (o *CIAppCompute) GetAggregationOk() (*CIAppAggregationFunction, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Aggregation, true
}

// SetAggregation sets field value.
func (o *CIAppCompute) SetAggregation(v CIAppAggregationFunction) {
	o.Aggregation = v
}

// GetInterval returns the Interval field value if set, zero value otherwise.
func (o *CIAppCompute) GetInterval() string {
	if o == nil || o.Interval == nil {
		var ret string
		return ret
	}
	return *o.Interval
}

// GetIntervalOk returns a tuple with the Interval field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *CIAppCompute) GetIntervalOk() (*string, bool) {
	if o == nil || o.Interval == nil {
		return nil, false
	}
	return o.Interval, true
}

// HasInterval returns a boolean if a field has been set.
func (o *CIAppCompute) HasInterval() bool {
	return o != nil && o.Interval != nil
}

// SetInterval gets a reference to the given string and assigns it to the Interval field.
func (o *CIAppCompute) SetInterval(v string) {
	o.Interval = &v
}

// GetMetric returns the Metric field value if set, zero value otherwise.
func (o *CIAppCompute) GetMetric() string {
	if o == nil || o.Metric == nil {
		var ret string
		return ret
	}
	return *o.Metric
}

// GetMetricOk returns a tuple with the Metric field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *CIAppCompute) GetMetricOk() (*string, bool) {
	if o == nil || o.Metric == nil {
		return nil, false
	}
	return o.Metric, true
}

// HasMetric returns a boolean if a field has been set.
func (o *CIAppCompute) HasMetric() bool {
	return o != nil && o.Metric != nil
}

// SetMetric gets a reference to the given string and assigns it to the Metric field.
func (o *CIAppCompute) SetMetric(v string) {
	o.Metric = &v
}

// GetType returns the Type field value if set, zero value otherwise.
func (o *CIAppCompute) GetType() CIAppComputeType {
	if o == nil || o.Type == nil {
		var ret CIAppComputeType
		return ret
	}
	return *o.Type
}

// GetTypeOk returns a tuple with the Type field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *CIAppCompute) GetTypeOk() (*CIAppComputeType, bool) {
	if o == nil || o.Type == nil {
		return nil, false
	}
	return o.Type, true
}

// HasType returns a boolean if a field has been set.
func (o *CIAppCompute) HasType() bool {
	return o != nil && o.Type != nil
}

// SetType gets a reference to the given CIAppComputeType and assigns it to the Type field.
func (o *CIAppCompute) SetType(v CIAppComputeType) {
	o.Type = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o CIAppCompute) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["aggregation"] = o.Aggregation
	if o.Interval != nil {
		toSerialize["interval"] = o.Interval
	}
	if o.Metric != nil {
		toSerialize["metric"] = o.Metric
	}
	if o.Type != nil {
		toSerialize["type"] = o.Type
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *CIAppCompute) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Aggregation *CIAppAggregationFunction `json:"aggregation"`
	}{}
	all := struct {
		Aggregation CIAppAggregationFunction `json:"aggregation"`
		Interval    *string                  `json:"interval,omitempty"`
		Metric      *string                  `json:"metric,omitempty"`
		Type        *CIAppComputeType        `json:"type,omitempty"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Aggregation == nil {
		return fmt.Errorf("required field aggregation missing")
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
	if v := all.Aggregation; !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	if v := all.Type; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.Aggregation = all.Aggregation
	o.Interval = all.Interval
	o.Metric = all.Metric
	o.Type = all.Type
	return nil
}
