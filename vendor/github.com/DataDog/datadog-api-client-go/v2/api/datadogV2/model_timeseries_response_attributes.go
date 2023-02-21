// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// TimeseriesResponseAttributes The object describing a timeseries response.
type TimeseriesResponseAttributes struct {
	// Array of response series. The index here corresponds to the index in the `formulas` or `queries` array from the request.
	Series []TimeseriesResponseSeries `json:"series,omitempty"`
	// Array of times, 1-1 match with individual values arrays.
	Times []int64 `json:"times,omitempty"`
	// Array of value-arrays. The index here corresponds to the index in the `formulas` or `queries` array from the request.
	Values [][]*float64 `json:"values,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewTimeseriesResponseAttributes instantiates a new TimeseriesResponseAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewTimeseriesResponseAttributes() *TimeseriesResponseAttributes {
	this := TimeseriesResponseAttributes{}
	return &this
}

// NewTimeseriesResponseAttributesWithDefaults instantiates a new TimeseriesResponseAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewTimeseriesResponseAttributesWithDefaults() *TimeseriesResponseAttributes {
	this := TimeseriesResponseAttributes{}
	return &this
}

// GetSeries returns the Series field value if set, zero value otherwise.
func (o *TimeseriesResponseAttributes) GetSeries() []TimeseriesResponseSeries {
	if o == nil || o.Series == nil {
		var ret []TimeseriesResponseSeries
		return ret
	}
	return o.Series
}

// GetSeriesOk returns a tuple with the Series field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *TimeseriesResponseAttributes) GetSeriesOk() (*[]TimeseriesResponseSeries, bool) {
	if o == nil || o.Series == nil {
		return nil, false
	}
	return &o.Series, true
}

// HasSeries returns a boolean if a field has been set.
func (o *TimeseriesResponseAttributes) HasSeries() bool {
	return o != nil && o.Series != nil
}

// SetSeries gets a reference to the given []TimeseriesResponseSeries and assigns it to the Series field.
func (o *TimeseriesResponseAttributes) SetSeries(v []TimeseriesResponseSeries) {
	o.Series = v
}

// GetTimes returns the Times field value if set, zero value otherwise.
func (o *TimeseriesResponseAttributes) GetTimes() []int64 {
	if o == nil || o.Times == nil {
		var ret []int64
		return ret
	}
	return o.Times
}

// GetTimesOk returns a tuple with the Times field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *TimeseriesResponseAttributes) GetTimesOk() (*[]int64, bool) {
	if o == nil || o.Times == nil {
		return nil, false
	}
	return &o.Times, true
}

// HasTimes returns a boolean if a field has been set.
func (o *TimeseriesResponseAttributes) HasTimes() bool {
	return o != nil && o.Times != nil
}

// SetTimes gets a reference to the given []int64 and assigns it to the Times field.
func (o *TimeseriesResponseAttributes) SetTimes(v []int64) {
	o.Times = v
}

// GetValues returns the Values field value if set, zero value otherwise.
func (o *TimeseriesResponseAttributes) GetValues() [][]*float64 {
	if o == nil || o.Values == nil {
		var ret [][]*float64
		return ret
	}
	return o.Values
}

// GetValuesOk returns a tuple with the Values field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *TimeseriesResponseAttributes) GetValuesOk() (*[][]*float64, bool) {
	if o == nil || o.Values == nil {
		return nil, false
	}
	return &o.Values, true
}

// HasValues returns a boolean if a field has been set.
func (o *TimeseriesResponseAttributes) HasValues() bool {
	return o != nil && o.Values != nil
}

// SetValues gets a reference to the given [][]*float64 and assigns it to the Values field.
func (o *TimeseriesResponseAttributes) SetValues(v [][]*float64) {
	o.Values = v
}

// MarshalJSON serializes the struct using spec logic.
func (o TimeseriesResponseAttributes) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Series != nil {
		toSerialize["series"] = o.Series
	}
	if o.Times != nil {
		toSerialize["times"] = o.Times
	}
	if o.Values != nil {
		toSerialize["values"] = o.Values
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *TimeseriesResponseAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Series []TimeseriesResponseSeries `json:"series,omitempty"`
		Times  []int64                    `json:"times,omitempty"`
		Values [][]*float64               `json:"values,omitempty"`
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
	o.Series = all.Series
	o.Times = all.Times
	o.Values = all.Values
	return nil
}
