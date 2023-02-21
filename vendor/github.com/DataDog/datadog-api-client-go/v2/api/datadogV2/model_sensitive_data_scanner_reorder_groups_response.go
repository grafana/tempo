// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// SensitiveDataScannerReorderGroupsResponse Group reorder response.
type SensitiveDataScannerReorderGroupsResponse struct {
	// Meta response containing information about the API.
	Meta *SensitiveDataScannerMeta `json:"meta,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewSensitiveDataScannerReorderGroupsResponse instantiates a new SensitiveDataScannerReorderGroupsResponse object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSensitiveDataScannerReorderGroupsResponse() *SensitiveDataScannerReorderGroupsResponse {
	this := SensitiveDataScannerReorderGroupsResponse{}
	return &this
}

// NewSensitiveDataScannerReorderGroupsResponseWithDefaults instantiates a new SensitiveDataScannerReorderGroupsResponse object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSensitiveDataScannerReorderGroupsResponseWithDefaults() *SensitiveDataScannerReorderGroupsResponse {
	this := SensitiveDataScannerReorderGroupsResponse{}
	return &this
}

// GetMeta returns the Meta field value if set, zero value otherwise.
func (o *SensitiveDataScannerReorderGroupsResponse) GetMeta() SensitiveDataScannerMeta {
	if o == nil || o.Meta == nil {
		var ret SensitiveDataScannerMeta
		return ret
	}
	return *o.Meta
}

// GetMetaOk returns a tuple with the Meta field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SensitiveDataScannerReorderGroupsResponse) GetMetaOk() (*SensitiveDataScannerMeta, bool) {
	if o == nil || o.Meta == nil {
		return nil, false
	}
	return o.Meta, true
}

// HasMeta returns a boolean if a field has been set.
func (o *SensitiveDataScannerReorderGroupsResponse) HasMeta() bool {
	return o != nil && o.Meta != nil
}

// SetMeta gets a reference to the given SensitiveDataScannerMeta and assigns it to the Meta field.
func (o *SensitiveDataScannerReorderGroupsResponse) SetMeta(v SensitiveDataScannerMeta) {
	o.Meta = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SensitiveDataScannerReorderGroupsResponse) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Meta != nil {
		toSerialize["meta"] = o.Meta
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SensitiveDataScannerReorderGroupsResponse) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Meta *SensitiveDataScannerMeta `json:"meta,omitempty"`
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
	if all.Meta != nil && all.Meta.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Meta = all.Meta
	return nil
}
