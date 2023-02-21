// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// SensitiveDataScannerConfigRequest Group reorder request.
type SensitiveDataScannerConfigRequest struct {
	// Data related to the reordering of scanning groups.
	Data SensitiveDataScannerReorderConfig `json:"data"`
	// Meta payload containing information about the API.
	Meta SensitiveDataScannerMetaVersionOnly `json:"meta"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewSensitiveDataScannerConfigRequest instantiates a new SensitiveDataScannerConfigRequest object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSensitiveDataScannerConfigRequest(data SensitiveDataScannerReorderConfig, meta SensitiveDataScannerMetaVersionOnly) *SensitiveDataScannerConfigRequest {
	this := SensitiveDataScannerConfigRequest{}
	this.Data = data
	this.Meta = meta
	return &this
}

// NewSensitiveDataScannerConfigRequestWithDefaults instantiates a new SensitiveDataScannerConfigRequest object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSensitiveDataScannerConfigRequestWithDefaults() *SensitiveDataScannerConfigRequest {
	this := SensitiveDataScannerConfigRequest{}
	return &this
}

// GetData returns the Data field value.
func (o *SensitiveDataScannerConfigRequest) GetData() SensitiveDataScannerReorderConfig {
	if o == nil {
		var ret SensitiveDataScannerReorderConfig
		return ret
	}
	return o.Data
}

// GetDataOk returns a tuple with the Data field value
// and a boolean to check if the value has been set.
func (o *SensitiveDataScannerConfigRequest) GetDataOk() (*SensitiveDataScannerReorderConfig, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Data, true
}

// SetData sets field value.
func (o *SensitiveDataScannerConfigRequest) SetData(v SensitiveDataScannerReorderConfig) {
	o.Data = v
}

// GetMeta returns the Meta field value.
func (o *SensitiveDataScannerConfigRequest) GetMeta() SensitiveDataScannerMetaVersionOnly {
	if o == nil {
		var ret SensitiveDataScannerMetaVersionOnly
		return ret
	}
	return o.Meta
}

// GetMetaOk returns a tuple with the Meta field value
// and a boolean to check if the value has been set.
func (o *SensitiveDataScannerConfigRequest) GetMetaOk() (*SensitiveDataScannerMetaVersionOnly, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Meta, true
}

// SetMeta sets field value.
func (o *SensitiveDataScannerConfigRequest) SetMeta(v SensitiveDataScannerMetaVersionOnly) {
	o.Meta = v
}

// MarshalJSON serializes the struct using spec logic.
func (o SensitiveDataScannerConfigRequest) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["data"] = o.Data
	toSerialize["meta"] = o.Meta

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SensitiveDataScannerConfigRequest) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Data *SensitiveDataScannerReorderConfig   `json:"data"`
		Meta *SensitiveDataScannerMetaVersionOnly `json:"meta"`
	}{}
	all := struct {
		Data SensitiveDataScannerReorderConfig   `json:"data"`
		Meta SensitiveDataScannerMetaVersionOnly `json:"meta"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Data == nil {
		return fmt.Errorf("required field data missing")
	}
	if required.Meta == nil {
		return fmt.Errorf("required field meta missing")
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
	if all.Data.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Data = all.Data
	if all.Meta.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Meta = all.Meta
	return nil
}
