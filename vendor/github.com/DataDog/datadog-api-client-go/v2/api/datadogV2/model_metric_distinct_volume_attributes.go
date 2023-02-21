// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// MetricDistinctVolumeAttributes Object containing the definition of a metric's distinct volume.
type MetricDistinctVolumeAttributes struct {
	// Distinct volume for the given metric.
	DistinctVolume *int64 `json:"distinct_volume,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewMetricDistinctVolumeAttributes instantiates a new MetricDistinctVolumeAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewMetricDistinctVolumeAttributes() *MetricDistinctVolumeAttributes {
	this := MetricDistinctVolumeAttributes{}
	return &this
}

// NewMetricDistinctVolumeAttributesWithDefaults instantiates a new MetricDistinctVolumeAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewMetricDistinctVolumeAttributesWithDefaults() *MetricDistinctVolumeAttributes {
	this := MetricDistinctVolumeAttributes{}
	return &this
}

// GetDistinctVolume returns the DistinctVolume field value if set, zero value otherwise.
func (o *MetricDistinctVolumeAttributes) GetDistinctVolume() int64 {
	if o == nil || o.DistinctVolume == nil {
		var ret int64
		return ret
	}
	return *o.DistinctVolume
}

// GetDistinctVolumeOk returns a tuple with the DistinctVolume field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MetricDistinctVolumeAttributes) GetDistinctVolumeOk() (*int64, bool) {
	if o == nil || o.DistinctVolume == nil {
		return nil, false
	}
	return o.DistinctVolume, true
}

// HasDistinctVolume returns a boolean if a field has been set.
func (o *MetricDistinctVolumeAttributes) HasDistinctVolume() bool {
	return o != nil && o.DistinctVolume != nil
}

// SetDistinctVolume gets a reference to the given int64 and assigns it to the DistinctVolume field.
func (o *MetricDistinctVolumeAttributes) SetDistinctVolume(v int64) {
	o.DistinctVolume = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o MetricDistinctVolumeAttributes) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.DistinctVolume != nil {
		toSerialize["distinct_volume"] = o.DistinctVolume
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *MetricDistinctVolumeAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		DistinctVolume *int64 `json:"distinct_volume,omitempty"`
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
	o.DistinctVolume = all.DistinctVolume
	return nil
}
