// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// SecurityMonitoringTriageUser Object representing a given user entity.
type SecurityMonitoringTriageUser struct {
	// The handle for this user account.
	Handle *string `json:"handle,omitempty"`
	// Numerical ID assigned by Datadog to this user account.
	Id *int64 `json:"id,omitempty"`
	// The name for this user account.
	Name *string `json:"name,omitempty"`
	// UUID assigned by Datadog to this user account.
	Uuid string `json:"uuid"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewSecurityMonitoringTriageUser instantiates a new SecurityMonitoringTriageUser object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSecurityMonitoringTriageUser(uuid string) *SecurityMonitoringTriageUser {
	this := SecurityMonitoringTriageUser{}
	this.Uuid = uuid
	return &this
}

// NewSecurityMonitoringTriageUserWithDefaults instantiates a new SecurityMonitoringTriageUser object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSecurityMonitoringTriageUserWithDefaults() *SecurityMonitoringTriageUser {
	this := SecurityMonitoringTriageUser{}
	return &this
}

// GetHandle returns the Handle field value if set, zero value otherwise.
func (o *SecurityMonitoringTriageUser) GetHandle() string {
	if o == nil || o.Handle == nil {
		var ret string
		return ret
	}
	return *o.Handle
}

// GetHandleOk returns a tuple with the Handle field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringTriageUser) GetHandleOk() (*string, bool) {
	if o == nil || o.Handle == nil {
		return nil, false
	}
	return o.Handle, true
}

// HasHandle returns a boolean if a field has been set.
func (o *SecurityMonitoringTriageUser) HasHandle() bool {
	return o != nil && o.Handle != nil
}

// SetHandle gets a reference to the given string and assigns it to the Handle field.
func (o *SecurityMonitoringTriageUser) SetHandle(v string) {
	o.Handle = &v
}

// GetId returns the Id field value if set, zero value otherwise.
func (o *SecurityMonitoringTriageUser) GetId() int64 {
	if o == nil || o.Id == nil {
		var ret int64
		return ret
	}
	return *o.Id
}

// GetIdOk returns a tuple with the Id field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringTriageUser) GetIdOk() (*int64, bool) {
	if o == nil || o.Id == nil {
		return nil, false
	}
	return o.Id, true
}

// HasId returns a boolean if a field has been set.
func (o *SecurityMonitoringTriageUser) HasId() bool {
	return o != nil && o.Id != nil
}

// SetId gets a reference to the given int64 and assigns it to the Id field.
func (o *SecurityMonitoringTriageUser) SetId(v int64) {
	o.Id = &v
}

// GetName returns the Name field value if set, zero value otherwise.
func (o *SecurityMonitoringTriageUser) GetName() string {
	if o == nil || o.Name == nil {
		var ret string
		return ret
	}
	return *o.Name
}

// GetNameOk returns a tuple with the Name field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringTriageUser) GetNameOk() (*string, bool) {
	if o == nil || o.Name == nil {
		return nil, false
	}
	return o.Name, true
}

// HasName returns a boolean if a field has been set.
func (o *SecurityMonitoringTriageUser) HasName() bool {
	return o != nil && o.Name != nil
}

// SetName gets a reference to the given string and assigns it to the Name field.
func (o *SecurityMonitoringTriageUser) SetName(v string) {
	o.Name = &v
}

// GetUuid returns the Uuid field value.
func (o *SecurityMonitoringTriageUser) GetUuid() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Uuid
}

// GetUuidOk returns a tuple with the Uuid field value
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringTriageUser) GetUuidOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Uuid, true
}

// SetUuid sets field value.
func (o *SecurityMonitoringTriageUser) SetUuid(v string) {
	o.Uuid = v
}

// MarshalJSON serializes the struct using spec logic.
func (o SecurityMonitoringTriageUser) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Handle != nil {
		toSerialize["handle"] = o.Handle
	}
	if o.Id != nil {
		toSerialize["id"] = o.Id
	}
	if o.Name != nil {
		toSerialize["name"] = o.Name
	}
	toSerialize["uuid"] = o.Uuid

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SecurityMonitoringTriageUser) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Uuid *string `json:"uuid"`
	}{}
	all := struct {
		Handle *string `json:"handle,omitempty"`
		Id     *int64  `json:"id,omitempty"`
		Name   *string `json:"name,omitempty"`
		Uuid   string  `json:"uuid"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Uuid == nil {
		return fmt.Errorf("required field uuid missing")
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
	o.Handle = all.Handle
	o.Id = all.Id
	o.Name = all.Name
	o.Uuid = all.Uuid
	return nil
}
