// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV1

import (
	"encoding/json"
)

// SearchSLOQuery A metric-based SLO. **Required if type is `metric`**. Note that Datadog only allows the sum by aggregator
// to be used because this will sum up all request counts instead of averaging them, or taking the max or
// min of all of those requests.
type SearchSLOQuery struct {
	// A Datadog metric query for total (valid) events.
	Denominator *string `json:"denominator,omitempty"`
	// Metric names used in the query's numerator and denominator.
	// This field will return null and will be implemented in the next version of this endpoint.
	Metrics []string `json:"metrics,omitempty"`
	// A Datadog metric query for good events.
	Numerator *string `json:"numerator,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewSearchSLOQuery instantiates a new SearchSLOQuery object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSearchSLOQuery() *SearchSLOQuery {
	this := SearchSLOQuery{}
	return &this
}

// NewSearchSLOQueryWithDefaults instantiates a new SearchSLOQuery object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSearchSLOQueryWithDefaults() *SearchSLOQuery {
	this := SearchSLOQuery{}
	return &this
}

// GetDenominator returns the Denominator field value if set, zero value otherwise.
func (o *SearchSLOQuery) GetDenominator() string {
	if o == nil || o.Denominator == nil {
		var ret string
		return ret
	}
	return *o.Denominator
}

// GetDenominatorOk returns a tuple with the Denominator field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SearchSLOQuery) GetDenominatorOk() (*string, bool) {
	if o == nil || o.Denominator == nil {
		return nil, false
	}
	return o.Denominator, true
}

// HasDenominator returns a boolean if a field has been set.
func (o *SearchSLOQuery) HasDenominator() bool {
	return o != nil && o.Denominator != nil
}

// SetDenominator gets a reference to the given string and assigns it to the Denominator field.
func (o *SearchSLOQuery) SetDenominator(v string) {
	o.Denominator = &v
}

// GetMetrics returns the Metrics field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *SearchSLOQuery) GetMetrics() []string {
	if o == nil {
		var ret []string
		return ret
	}
	return o.Metrics
}

// GetMetricsOk returns a tuple with the Metrics field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *SearchSLOQuery) GetMetricsOk() (*[]string, bool) {
	if o == nil || o.Metrics == nil {
		return nil, false
	}
	return &o.Metrics, true
}

// HasMetrics returns a boolean if a field has been set.
func (o *SearchSLOQuery) HasMetrics() bool {
	return o != nil && o.Metrics != nil
}

// SetMetrics gets a reference to the given []string and assigns it to the Metrics field.
func (o *SearchSLOQuery) SetMetrics(v []string) {
	o.Metrics = v
}

// GetNumerator returns the Numerator field value if set, zero value otherwise.
func (o *SearchSLOQuery) GetNumerator() string {
	if o == nil || o.Numerator == nil {
		var ret string
		return ret
	}
	return *o.Numerator
}

// GetNumeratorOk returns a tuple with the Numerator field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SearchSLOQuery) GetNumeratorOk() (*string, bool) {
	if o == nil || o.Numerator == nil {
		return nil, false
	}
	return o.Numerator, true
}

// HasNumerator returns a boolean if a field has been set.
func (o *SearchSLOQuery) HasNumerator() bool {
	return o != nil && o.Numerator != nil
}

// SetNumerator gets a reference to the given string and assigns it to the Numerator field.
func (o *SearchSLOQuery) SetNumerator(v string) {
	o.Numerator = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SearchSLOQuery) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Denominator != nil {
		toSerialize["denominator"] = o.Denominator
	}
	if o.Metrics != nil {
		toSerialize["metrics"] = o.Metrics
	}
	if o.Numerator != nil {
		toSerialize["numerator"] = o.Numerator
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SearchSLOQuery) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Denominator *string  `json:"denominator,omitempty"`
		Metrics     []string `json:"metrics,omitempty"`
		Numerator   *string  `json:"numerator,omitempty"`
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
	o.Denominator = all.Denominator
	o.Metrics = all.Metrics
	o.Numerator = all.Numerator
	return nil
}

// NullableSearchSLOQuery handles when a null is used for SearchSLOQuery.
type NullableSearchSLOQuery struct {
	value *SearchSLOQuery
	isSet bool
}

// Get returns the associated value.
func (v NullableSearchSLOQuery) Get() *SearchSLOQuery {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableSearchSLOQuery) Set(val *SearchSLOQuery) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableSearchSLOQuery) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag/
func (v *NullableSearchSLOQuery) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableSearchSLOQuery initializes the struct as if Set has been called.
func NewNullableSearchSLOQuery(val *SearchSLOQuery) *NullableSearchSLOQuery {
	return &NullableSearchSLOQuery{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableSearchSLOQuery) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableSearchSLOQuery) UnmarshalJSON(src []byte) error {
	v.isSet = true

	// this object is nullable so check if the payload is null or empty string
	if string(src) == "" || string(src) == "{}" {
		return nil
	}

	return json.Unmarshal(src, &v.value)
}
