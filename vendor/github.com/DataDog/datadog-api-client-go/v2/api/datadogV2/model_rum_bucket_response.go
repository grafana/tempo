// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// RUMBucketResponse Bucket values.
type RUMBucketResponse struct {
	// The key-value pairs for each group-by.
	By map[string]string `json:"by,omitempty"`
	// A map of the metric name to value for regular compute, or a list of values for a timeseries.
	Computes map[string]RUMAggregateBucketValue `json:"computes,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewRUMBucketResponse instantiates a new RUMBucketResponse object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewRUMBucketResponse() *RUMBucketResponse {
	this := RUMBucketResponse{}
	return &this
}

// NewRUMBucketResponseWithDefaults instantiates a new RUMBucketResponse object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewRUMBucketResponseWithDefaults() *RUMBucketResponse {
	this := RUMBucketResponse{}
	return &this
}

// GetBy returns the By field value if set, zero value otherwise.
func (o *RUMBucketResponse) GetBy() map[string]string {
	if o == nil || o.By == nil {
		var ret map[string]string
		return ret
	}
	return o.By
}

// GetByOk returns a tuple with the By field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *RUMBucketResponse) GetByOk() (*map[string]string, bool) {
	if o == nil || o.By == nil {
		return nil, false
	}
	return &o.By, true
}

// HasBy returns a boolean if a field has been set.
func (o *RUMBucketResponse) HasBy() bool {
	return o != nil && o.By != nil
}

// SetBy gets a reference to the given map[string]string and assigns it to the By field.
func (o *RUMBucketResponse) SetBy(v map[string]string) {
	o.By = v
}

// GetComputes returns the Computes field value if set, zero value otherwise.
func (o *RUMBucketResponse) GetComputes() map[string]RUMAggregateBucketValue {
	if o == nil || o.Computes == nil {
		var ret map[string]RUMAggregateBucketValue
		return ret
	}
	return o.Computes
}

// GetComputesOk returns a tuple with the Computes field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *RUMBucketResponse) GetComputesOk() (*map[string]RUMAggregateBucketValue, bool) {
	if o == nil || o.Computes == nil {
		return nil, false
	}
	return &o.Computes, true
}

// HasComputes returns a boolean if a field has been set.
func (o *RUMBucketResponse) HasComputes() bool {
	return o != nil && o.Computes != nil
}

// SetComputes gets a reference to the given map[string]RUMAggregateBucketValue and assigns it to the Computes field.
func (o *RUMBucketResponse) SetComputes(v map[string]RUMAggregateBucketValue) {
	o.Computes = v
}

// MarshalJSON serializes the struct using spec logic.
func (o RUMBucketResponse) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.By != nil {
		toSerialize["by"] = o.By
	}
	if o.Computes != nil {
		toSerialize["computes"] = o.Computes
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *RUMBucketResponse) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		By       map[string]string                  `json:"by,omitempty"`
		Computes map[string]RUMAggregateBucketValue `json:"computes,omitempty"`
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
	o.By = all.By
	o.Computes = all.Computes
	return nil
}
