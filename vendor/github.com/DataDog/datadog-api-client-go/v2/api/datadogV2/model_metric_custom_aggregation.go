// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// MetricCustomAggregation A time and space aggregation combination for use in query.
type MetricCustomAggregation struct {
	// A space aggregation for use in query.
	Space MetricCustomSpaceAggregation `json:"space"`
	// A time aggregation for use in query.
	Time MetricCustomTimeAggregation `json:"time"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewMetricCustomAggregation instantiates a new MetricCustomAggregation object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewMetricCustomAggregation(space MetricCustomSpaceAggregation, time MetricCustomTimeAggregation) *MetricCustomAggregation {
	this := MetricCustomAggregation{}
	this.Space = space
	this.Time = time
	return &this
}

// NewMetricCustomAggregationWithDefaults instantiates a new MetricCustomAggregation object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewMetricCustomAggregationWithDefaults() *MetricCustomAggregation {
	this := MetricCustomAggregation{}
	return &this
}

// GetSpace returns the Space field value.
func (o *MetricCustomAggregation) GetSpace() MetricCustomSpaceAggregation {
	if o == nil {
		var ret MetricCustomSpaceAggregation
		return ret
	}
	return o.Space
}

// GetSpaceOk returns a tuple with the Space field value
// and a boolean to check if the value has been set.
func (o *MetricCustomAggregation) GetSpaceOk() (*MetricCustomSpaceAggregation, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Space, true
}

// SetSpace sets field value.
func (o *MetricCustomAggregation) SetSpace(v MetricCustomSpaceAggregation) {
	o.Space = v
}

// GetTime returns the Time field value.
func (o *MetricCustomAggregation) GetTime() MetricCustomTimeAggregation {
	if o == nil {
		var ret MetricCustomTimeAggregation
		return ret
	}
	return o.Time
}

// GetTimeOk returns a tuple with the Time field value
// and a boolean to check if the value has been set.
func (o *MetricCustomAggregation) GetTimeOk() (*MetricCustomTimeAggregation, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Time, true
}

// SetTime sets field value.
func (o *MetricCustomAggregation) SetTime(v MetricCustomTimeAggregation) {
	o.Time = v
}

// MarshalJSON serializes the struct using spec logic.
func (o MetricCustomAggregation) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["space"] = o.Space
	toSerialize["time"] = o.Time

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *MetricCustomAggregation) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Space *MetricCustomSpaceAggregation `json:"space"`
		Time  *MetricCustomTimeAggregation  `json:"time"`
	}{}
	all := struct {
		Space MetricCustomSpaceAggregation `json:"space"`
		Time  MetricCustomTimeAggregation  `json:"time"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Space == nil {
		return fmt.Errorf("required field space missing")
	}
	if required.Time == nil {
		return fmt.Errorf("required field time missing")
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
	if v := all.Space; !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	if v := all.Time; !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.Space = all.Space
	o.Time = all.Time
	return nil
}
