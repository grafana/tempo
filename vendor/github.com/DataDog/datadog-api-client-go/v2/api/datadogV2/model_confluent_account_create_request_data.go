// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// ConfluentAccountCreateRequestData The data body for adding a Confluent account.
type ConfluentAccountCreateRequestData struct {
	// Attributes associated with the account creation request.
	Attributes ConfluentAccountCreateRequestAttributes `json:"attributes"`
	// The JSON:API type for this API. Should always be `confluent-cloud-accounts`.
	Type ConfluentAccountType `json:"type"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewConfluentAccountCreateRequestData instantiates a new ConfluentAccountCreateRequestData object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewConfluentAccountCreateRequestData(attributes ConfluentAccountCreateRequestAttributes, typeVar ConfluentAccountType) *ConfluentAccountCreateRequestData {
	this := ConfluentAccountCreateRequestData{}
	this.Attributes = attributes
	this.Type = typeVar
	return &this
}

// NewConfluentAccountCreateRequestDataWithDefaults instantiates a new ConfluentAccountCreateRequestData object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewConfluentAccountCreateRequestDataWithDefaults() *ConfluentAccountCreateRequestData {
	this := ConfluentAccountCreateRequestData{}
	var typeVar ConfluentAccountType = CONFLUENTACCOUNTTYPE_CONFLUENT_CLOUD_ACCOUNTS
	this.Type = typeVar
	return &this
}

// GetAttributes returns the Attributes field value.
func (o *ConfluentAccountCreateRequestData) GetAttributes() ConfluentAccountCreateRequestAttributes {
	if o == nil {
		var ret ConfluentAccountCreateRequestAttributes
		return ret
	}
	return o.Attributes
}

// GetAttributesOk returns a tuple with the Attributes field value
// and a boolean to check if the value has been set.
func (o *ConfluentAccountCreateRequestData) GetAttributesOk() (*ConfluentAccountCreateRequestAttributes, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Attributes, true
}

// SetAttributes sets field value.
func (o *ConfluentAccountCreateRequestData) SetAttributes(v ConfluentAccountCreateRequestAttributes) {
	o.Attributes = v
}

// GetType returns the Type field value.
func (o *ConfluentAccountCreateRequestData) GetType() ConfluentAccountType {
	if o == nil {
		var ret ConfluentAccountType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *ConfluentAccountCreateRequestData) GetTypeOk() (*ConfluentAccountType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *ConfluentAccountCreateRequestData) SetType(v ConfluentAccountType) {
	o.Type = v
}

// MarshalJSON serializes the struct using spec logic.
func (o ConfluentAccountCreateRequestData) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["attributes"] = o.Attributes
	toSerialize["type"] = o.Type

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *ConfluentAccountCreateRequestData) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Attributes *ConfluentAccountCreateRequestAttributes `json:"attributes"`
		Type       *ConfluentAccountType                    `json:"type"`
	}{}
	all := struct {
		Attributes ConfluentAccountCreateRequestAttributes `json:"attributes"`
		Type       ConfluentAccountType                    `json:"type"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Attributes == nil {
		return fmt.Errorf("required field attributes missing")
	}
	if required.Type == nil {
		return fmt.Errorf("required field type missing")
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
	if v := all.Type; !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	if all.Attributes.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Attributes = all.Attributes
	o.Type = all.Type
	return nil
}
