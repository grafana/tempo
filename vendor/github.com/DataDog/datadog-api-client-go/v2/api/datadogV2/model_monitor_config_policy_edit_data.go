// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// MonitorConfigPolicyEditData A monitor configuration policy data.
type MonitorConfigPolicyEditData struct {
	// Policy and policy type for a monitor configuration policy.
	Attributes MonitorConfigPolicyAttributeEditRequest `json:"attributes"`
	// ID of this monitor configuration policy.
	Id string `json:"id"`
	// Monitor configuration policy resource type.
	Type MonitorConfigPolicyResourceType `json:"type"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewMonitorConfigPolicyEditData instantiates a new MonitorConfigPolicyEditData object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewMonitorConfigPolicyEditData(attributes MonitorConfigPolicyAttributeEditRequest, id string, typeVar MonitorConfigPolicyResourceType) *MonitorConfigPolicyEditData {
	this := MonitorConfigPolicyEditData{}
	this.Attributes = attributes
	this.Id = id
	this.Type = typeVar
	return &this
}

// NewMonitorConfigPolicyEditDataWithDefaults instantiates a new MonitorConfigPolicyEditData object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewMonitorConfigPolicyEditDataWithDefaults() *MonitorConfigPolicyEditData {
	this := MonitorConfigPolicyEditData{}
	var typeVar MonitorConfigPolicyResourceType = MONITORCONFIGPOLICYRESOURCETYPE_MONITOR_CONFIG_POLICY
	this.Type = typeVar
	return &this
}

// GetAttributes returns the Attributes field value.
func (o *MonitorConfigPolicyEditData) GetAttributes() MonitorConfigPolicyAttributeEditRequest {
	if o == nil {
		var ret MonitorConfigPolicyAttributeEditRequest
		return ret
	}
	return o.Attributes
}

// GetAttributesOk returns a tuple with the Attributes field value
// and a boolean to check if the value has been set.
func (o *MonitorConfigPolicyEditData) GetAttributesOk() (*MonitorConfigPolicyAttributeEditRequest, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Attributes, true
}

// SetAttributes sets field value.
func (o *MonitorConfigPolicyEditData) SetAttributes(v MonitorConfigPolicyAttributeEditRequest) {
	o.Attributes = v
}

// GetId returns the Id field value.
func (o *MonitorConfigPolicyEditData) GetId() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Id
}

// GetIdOk returns a tuple with the Id field value
// and a boolean to check if the value has been set.
func (o *MonitorConfigPolicyEditData) GetIdOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Id, true
}

// SetId sets field value.
func (o *MonitorConfigPolicyEditData) SetId(v string) {
	o.Id = v
}

// GetType returns the Type field value.
func (o *MonitorConfigPolicyEditData) GetType() MonitorConfigPolicyResourceType {
	if o == nil {
		var ret MonitorConfigPolicyResourceType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *MonitorConfigPolicyEditData) GetTypeOk() (*MonitorConfigPolicyResourceType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *MonitorConfigPolicyEditData) SetType(v MonitorConfigPolicyResourceType) {
	o.Type = v
}

// MarshalJSON serializes the struct using spec logic.
func (o MonitorConfigPolicyEditData) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["attributes"] = o.Attributes
	toSerialize["id"] = o.Id
	toSerialize["type"] = o.Type

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *MonitorConfigPolicyEditData) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Attributes *MonitorConfigPolicyAttributeEditRequest `json:"attributes"`
		Id         *string                                  `json:"id"`
		Type       *MonitorConfigPolicyResourceType         `json:"type"`
	}{}
	all := struct {
		Attributes MonitorConfigPolicyAttributeEditRequest `json:"attributes"`
		Id         string                                  `json:"id"`
		Type       MonitorConfigPolicyResourceType         `json:"type"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Attributes == nil {
		return fmt.Errorf("required field attributes missing")
	}
	if required.Id == nil {
		return fmt.Errorf("required field id missing")
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
	o.Id = all.Id
	o.Type = all.Type
	return nil
}
