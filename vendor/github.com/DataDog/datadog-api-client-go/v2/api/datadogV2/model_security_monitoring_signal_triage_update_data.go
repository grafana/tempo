// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// SecurityMonitoringSignalTriageUpdateData Data containing the updated triage attributes of the signal.
type SecurityMonitoringSignalTriageUpdateData struct {
	// Attributes describing a triage state update operation over a security signal.
	Attributes *SecurityMonitoringSignalTriageAttributes `json:"attributes,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewSecurityMonitoringSignalTriageUpdateData instantiates a new SecurityMonitoringSignalTriageUpdateData object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSecurityMonitoringSignalTriageUpdateData() *SecurityMonitoringSignalTriageUpdateData {
	this := SecurityMonitoringSignalTriageUpdateData{}
	return &this
}

// NewSecurityMonitoringSignalTriageUpdateDataWithDefaults instantiates a new SecurityMonitoringSignalTriageUpdateData object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSecurityMonitoringSignalTriageUpdateDataWithDefaults() *SecurityMonitoringSignalTriageUpdateData {
	this := SecurityMonitoringSignalTriageUpdateData{}
	return &this
}

// GetAttributes returns the Attributes field value if set, zero value otherwise.
func (o *SecurityMonitoringSignalTriageUpdateData) GetAttributes() SecurityMonitoringSignalTriageAttributes {
	if o == nil || o.Attributes == nil {
		var ret SecurityMonitoringSignalTriageAttributes
		return ret
	}
	return *o.Attributes
}

// GetAttributesOk returns a tuple with the Attributes field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringSignalTriageUpdateData) GetAttributesOk() (*SecurityMonitoringSignalTriageAttributes, bool) {
	if o == nil || o.Attributes == nil {
		return nil, false
	}
	return o.Attributes, true
}

// HasAttributes returns a boolean if a field has been set.
func (o *SecurityMonitoringSignalTriageUpdateData) HasAttributes() bool {
	return o != nil && o.Attributes != nil
}

// SetAttributes gets a reference to the given SecurityMonitoringSignalTriageAttributes and assigns it to the Attributes field.
func (o *SecurityMonitoringSignalTriageUpdateData) SetAttributes(v SecurityMonitoringSignalTriageAttributes) {
	o.Attributes = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SecurityMonitoringSignalTriageUpdateData) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Attributes != nil {
		toSerialize["attributes"] = o.Attributes
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SecurityMonitoringSignalTriageUpdateData) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Attributes *SecurityMonitoringSignalTriageAttributes `json:"attributes,omitempty"`
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
	if all.Attributes != nil && all.Attributes.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Attributes = all.Attributes
	return nil
}
