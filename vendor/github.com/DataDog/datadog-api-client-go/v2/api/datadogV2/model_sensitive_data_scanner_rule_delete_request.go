// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// SensitiveDataScannerRuleDeleteRequest Delete rule request.
type SensitiveDataScannerRuleDeleteRequest struct {
	// Meta payload containing information about the API.
	Meta SensitiveDataScannerMetaVersionOnly `json:"meta"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewSensitiveDataScannerRuleDeleteRequest instantiates a new SensitiveDataScannerRuleDeleteRequest object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSensitiveDataScannerRuleDeleteRequest(meta SensitiveDataScannerMetaVersionOnly) *SensitiveDataScannerRuleDeleteRequest {
	this := SensitiveDataScannerRuleDeleteRequest{}
	this.Meta = meta
	return &this
}

// NewSensitiveDataScannerRuleDeleteRequestWithDefaults instantiates a new SensitiveDataScannerRuleDeleteRequest object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSensitiveDataScannerRuleDeleteRequestWithDefaults() *SensitiveDataScannerRuleDeleteRequest {
	this := SensitiveDataScannerRuleDeleteRequest{}
	return &this
}

// GetMeta returns the Meta field value.
func (o *SensitiveDataScannerRuleDeleteRequest) GetMeta() SensitiveDataScannerMetaVersionOnly {
	if o == nil {
		var ret SensitiveDataScannerMetaVersionOnly
		return ret
	}
	return o.Meta
}

// GetMetaOk returns a tuple with the Meta field value
// and a boolean to check if the value has been set.
func (o *SensitiveDataScannerRuleDeleteRequest) GetMetaOk() (*SensitiveDataScannerMetaVersionOnly, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Meta, true
}

// SetMeta sets field value.
func (o *SensitiveDataScannerRuleDeleteRequest) SetMeta(v SensitiveDataScannerMetaVersionOnly) {
	o.Meta = v
}

// MarshalJSON serializes the struct using spec logic.
func (o SensitiveDataScannerRuleDeleteRequest) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["meta"] = o.Meta

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SensitiveDataScannerRuleDeleteRequest) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Meta *SensitiveDataScannerMetaVersionOnly `json:"meta"`
	}{}
	all := struct {
		Meta SensitiveDataScannerMetaVersionOnly `json:"meta"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
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
