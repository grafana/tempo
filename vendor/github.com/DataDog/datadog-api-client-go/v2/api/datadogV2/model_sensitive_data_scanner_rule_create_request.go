// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// SensitiveDataScannerRuleCreateRequest Create rule request.
type SensitiveDataScannerRuleCreateRequest struct {
	// Data related to the creation of a rule.
	Data SensitiveDataScannerRuleCreate `json:"data"`
	// Meta payload containing information about the API.
	Meta SensitiveDataScannerMetaVersionOnly `json:"meta"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewSensitiveDataScannerRuleCreateRequest instantiates a new SensitiveDataScannerRuleCreateRequest object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSensitiveDataScannerRuleCreateRequest(data SensitiveDataScannerRuleCreate, meta SensitiveDataScannerMetaVersionOnly) *SensitiveDataScannerRuleCreateRequest {
	this := SensitiveDataScannerRuleCreateRequest{}
	this.Data = data
	this.Meta = meta
	return &this
}

// NewSensitiveDataScannerRuleCreateRequestWithDefaults instantiates a new SensitiveDataScannerRuleCreateRequest object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSensitiveDataScannerRuleCreateRequestWithDefaults() *SensitiveDataScannerRuleCreateRequest {
	this := SensitiveDataScannerRuleCreateRequest{}
	return &this
}

// GetData returns the Data field value.
func (o *SensitiveDataScannerRuleCreateRequest) GetData() SensitiveDataScannerRuleCreate {
	if o == nil {
		var ret SensitiveDataScannerRuleCreate
		return ret
	}
	return o.Data
}

// GetDataOk returns a tuple with the Data field value
// and a boolean to check if the value has been set.
func (o *SensitiveDataScannerRuleCreateRequest) GetDataOk() (*SensitiveDataScannerRuleCreate, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Data, true
}

// SetData sets field value.
func (o *SensitiveDataScannerRuleCreateRequest) SetData(v SensitiveDataScannerRuleCreate) {
	o.Data = v
}

// GetMeta returns the Meta field value.
func (o *SensitiveDataScannerRuleCreateRequest) GetMeta() SensitiveDataScannerMetaVersionOnly {
	if o == nil {
		var ret SensitiveDataScannerMetaVersionOnly
		return ret
	}
	return o.Meta
}

// GetMetaOk returns a tuple with the Meta field value
// and a boolean to check if the value has been set.
func (o *SensitiveDataScannerRuleCreateRequest) GetMetaOk() (*SensitiveDataScannerMetaVersionOnly, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Meta, true
}

// SetMeta sets field value.
func (o *SensitiveDataScannerRuleCreateRequest) SetMeta(v SensitiveDataScannerMetaVersionOnly) {
	o.Meta = v
}

// MarshalJSON serializes the struct using spec logic.
func (o SensitiveDataScannerRuleCreateRequest) MarshalJSON() ([]byte, error) {
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
func (o *SensitiveDataScannerRuleCreateRequest) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Data *SensitiveDataScannerRuleCreate      `json:"data"`
		Meta *SensitiveDataScannerMetaVersionOnly `json:"meta"`
	}{}
	all := struct {
		Data SensitiveDataScannerRuleCreate      `json:"data"`
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
