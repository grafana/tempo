// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// SecurityMonitoringSignalAssigneeUpdateAttributes Attributes describing the new assignee of a security signal.
type SecurityMonitoringSignalAssigneeUpdateAttributes struct {
	// Object representing a given user entity.
	Assignee SecurityMonitoringTriageUser `json:"assignee"`
	// Version of the updated signal. If server side version is higher, update will be rejected.
	Version *int64 `json:"version,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewSecurityMonitoringSignalAssigneeUpdateAttributes instantiates a new SecurityMonitoringSignalAssigneeUpdateAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSecurityMonitoringSignalAssigneeUpdateAttributes(assignee SecurityMonitoringTriageUser) *SecurityMonitoringSignalAssigneeUpdateAttributes {
	this := SecurityMonitoringSignalAssigneeUpdateAttributes{}
	this.Assignee = assignee
	return &this
}

// NewSecurityMonitoringSignalAssigneeUpdateAttributesWithDefaults instantiates a new SecurityMonitoringSignalAssigneeUpdateAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSecurityMonitoringSignalAssigneeUpdateAttributesWithDefaults() *SecurityMonitoringSignalAssigneeUpdateAttributes {
	this := SecurityMonitoringSignalAssigneeUpdateAttributes{}
	return &this
}

// GetAssignee returns the Assignee field value.
func (o *SecurityMonitoringSignalAssigneeUpdateAttributes) GetAssignee() SecurityMonitoringTriageUser {
	if o == nil {
		var ret SecurityMonitoringTriageUser
		return ret
	}
	return o.Assignee
}

// GetAssigneeOk returns a tuple with the Assignee field value
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringSignalAssigneeUpdateAttributes) GetAssigneeOk() (*SecurityMonitoringTriageUser, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Assignee, true
}

// SetAssignee sets field value.
func (o *SecurityMonitoringSignalAssigneeUpdateAttributes) SetAssignee(v SecurityMonitoringTriageUser) {
	o.Assignee = v
}

// GetVersion returns the Version field value if set, zero value otherwise.
func (o *SecurityMonitoringSignalAssigneeUpdateAttributes) GetVersion() int64 {
	if o == nil || o.Version == nil {
		var ret int64
		return ret
	}
	return *o.Version
}

// GetVersionOk returns a tuple with the Version field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringSignalAssigneeUpdateAttributes) GetVersionOk() (*int64, bool) {
	if o == nil || o.Version == nil {
		return nil, false
	}
	return o.Version, true
}

// HasVersion returns a boolean if a field has been set.
func (o *SecurityMonitoringSignalAssigneeUpdateAttributes) HasVersion() bool {
	return o != nil && o.Version != nil
}

// SetVersion gets a reference to the given int64 and assigns it to the Version field.
func (o *SecurityMonitoringSignalAssigneeUpdateAttributes) SetVersion(v int64) {
	o.Version = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SecurityMonitoringSignalAssigneeUpdateAttributes) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["assignee"] = o.Assignee
	if o.Version != nil {
		toSerialize["version"] = o.Version
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SecurityMonitoringSignalAssigneeUpdateAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Assignee *SecurityMonitoringTriageUser `json:"assignee"`
	}{}
	all := struct {
		Assignee SecurityMonitoringTriageUser `json:"assignee"`
		Version  *int64                       `json:"version,omitempty"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Assignee == nil {
		return fmt.Errorf("required field assignee missing")
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
	if all.Assignee.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Assignee = all.Assignee
	o.Version = all.Version
	return nil
}
