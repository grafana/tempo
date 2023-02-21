// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
)

// IncidentSearchResponseNumericFacetDataAggregates Aggregate information for numeric incident data.
type IncidentSearchResponseNumericFacetDataAggregates struct {
	// Maximum value of the numeric aggregates.
	Max datadog.NullableFloat64 `json:"max,omitempty"`
	// Minimum value of the numeric aggregates.
	Min datadog.NullableFloat64 `json:"min,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewIncidentSearchResponseNumericFacetDataAggregates instantiates a new IncidentSearchResponseNumericFacetDataAggregates object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewIncidentSearchResponseNumericFacetDataAggregates() *IncidentSearchResponseNumericFacetDataAggregates {
	this := IncidentSearchResponseNumericFacetDataAggregates{}
	return &this
}

// NewIncidentSearchResponseNumericFacetDataAggregatesWithDefaults instantiates a new IncidentSearchResponseNumericFacetDataAggregates object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewIncidentSearchResponseNumericFacetDataAggregatesWithDefaults() *IncidentSearchResponseNumericFacetDataAggregates {
	this := IncidentSearchResponseNumericFacetDataAggregates{}
	return &this
}

// GetMax returns the Max field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *IncidentSearchResponseNumericFacetDataAggregates) GetMax() float64 {
	if o == nil || o.Max.Get() == nil {
		var ret float64
		return ret
	}
	return *o.Max.Get()
}

// GetMaxOk returns a tuple with the Max field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *IncidentSearchResponseNumericFacetDataAggregates) GetMaxOk() (*float64, bool) {
	if o == nil {
		return nil, false
	}
	return o.Max.Get(), o.Max.IsSet()
}

// HasMax returns a boolean if a field has been set.
func (o *IncidentSearchResponseNumericFacetDataAggregates) HasMax() bool {
	return o != nil && o.Max.IsSet()
}

// SetMax gets a reference to the given datadog.NullableFloat64 and assigns it to the Max field.
func (o *IncidentSearchResponseNumericFacetDataAggregates) SetMax(v float64) {
	o.Max.Set(&v)
}

// SetMaxNil sets the value for Max to be an explicit nil.
func (o *IncidentSearchResponseNumericFacetDataAggregates) SetMaxNil() {
	o.Max.Set(nil)
}

// UnsetMax ensures that no value is present for Max, not even an explicit nil.
func (o *IncidentSearchResponseNumericFacetDataAggregates) UnsetMax() {
	o.Max.Unset()
}

// GetMin returns the Min field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *IncidentSearchResponseNumericFacetDataAggregates) GetMin() float64 {
	if o == nil || o.Min.Get() == nil {
		var ret float64
		return ret
	}
	return *o.Min.Get()
}

// GetMinOk returns a tuple with the Min field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *IncidentSearchResponseNumericFacetDataAggregates) GetMinOk() (*float64, bool) {
	if o == nil {
		return nil, false
	}
	return o.Min.Get(), o.Min.IsSet()
}

// HasMin returns a boolean if a field has been set.
func (o *IncidentSearchResponseNumericFacetDataAggregates) HasMin() bool {
	return o != nil && o.Min.IsSet()
}

// SetMin gets a reference to the given datadog.NullableFloat64 and assigns it to the Min field.
func (o *IncidentSearchResponseNumericFacetDataAggregates) SetMin(v float64) {
	o.Min.Set(&v)
}

// SetMinNil sets the value for Min to be an explicit nil.
func (o *IncidentSearchResponseNumericFacetDataAggregates) SetMinNil() {
	o.Min.Set(nil)
}

// UnsetMin ensures that no value is present for Min, not even an explicit nil.
func (o *IncidentSearchResponseNumericFacetDataAggregates) UnsetMin() {
	o.Min.Unset()
}

// MarshalJSON serializes the struct using spec logic.
func (o IncidentSearchResponseNumericFacetDataAggregates) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Max.IsSet() {
		toSerialize["max"] = o.Max.Get()
	}
	if o.Min.IsSet() {
		toSerialize["min"] = o.Min.Get()
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *IncidentSearchResponseNumericFacetDataAggregates) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Max datadog.NullableFloat64 `json:"max,omitempty"`
		Min datadog.NullableFloat64 `json:"min,omitempty"`
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
	o.Max = all.Max
	o.Min = all.Min
	return nil
}
