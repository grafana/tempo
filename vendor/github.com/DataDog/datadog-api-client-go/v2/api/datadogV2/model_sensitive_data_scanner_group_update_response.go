// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// SensitiveDataScannerGroupUpdateResponse Update group response.
type SensitiveDataScannerGroupUpdateResponse struct {
	// Meta payload containing information about the API.
	Meta *SensitiveDataScannerMetaVersionOnly `json:"meta,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewSensitiveDataScannerGroupUpdateResponse instantiates a new SensitiveDataScannerGroupUpdateResponse object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSensitiveDataScannerGroupUpdateResponse() *SensitiveDataScannerGroupUpdateResponse {
	this := SensitiveDataScannerGroupUpdateResponse{}
	return &this
}

// NewSensitiveDataScannerGroupUpdateResponseWithDefaults instantiates a new SensitiveDataScannerGroupUpdateResponse object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSensitiveDataScannerGroupUpdateResponseWithDefaults() *SensitiveDataScannerGroupUpdateResponse {
	this := SensitiveDataScannerGroupUpdateResponse{}
	return &this
}

// GetMeta returns the Meta field value if set, zero value otherwise.
func (o *SensitiveDataScannerGroupUpdateResponse) GetMeta() SensitiveDataScannerMetaVersionOnly {
	if o == nil || o.Meta == nil {
		var ret SensitiveDataScannerMetaVersionOnly
		return ret
	}
	return *o.Meta
}

// GetMetaOk returns a tuple with the Meta field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SensitiveDataScannerGroupUpdateResponse) GetMetaOk() (*SensitiveDataScannerMetaVersionOnly, bool) {
	if o == nil || o.Meta == nil {
		return nil, false
	}
	return o.Meta, true
}

// HasMeta returns a boolean if a field has been set.
func (o *SensitiveDataScannerGroupUpdateResponse) HasMeta() bool {
	return o != nil && o.Meta != nil
}

// SetMeta gets a reference to the given SensitiveDataScannerMetaVersionOnly and assigns it to the Meta field.
func (o *SensitiveDataScannerGroupUpdateResponse) SetMeta(v SensitiveDataScannerMetaVersionOnly) {
	o.Meta = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SensitiveDataScannerGroupUpdateResponse) MarshalJSON() ([]byte, error) {
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
func (o *SensitiveDataScannerGroupUpdateResponse) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Meta *SensitiveDataScannerMetaVersionOnly `json:"meta,omitempty"`
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
