// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// IncidentNotificationHandle A notification handle that will be notified at incident creation.
type IncidentNotificationHandle struct {
	// The name of the notified handle.
	DisplayName *string `json:"display_name,omitempty"`
	// The email address used for the notification.
	Handle *string `json:"handle,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewIncidentNotificationHandle instantiates a new IncidentNotificationHandle object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewIncidentNotificationHandle() *IncidentNotificationHandle {
	this := IncidentNotificationHandle{}
	return &this
}

// NewIncidentNotificationHandleWithDefaults instantiates a new IncidentNotificationHandle object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewIncidentNotificationHandleWithDefaults() *IncidentNotificationHandle {
	this := IncidentNotificationHandle{}
	return &this
}

// GetDisplayName returns the DisplayName field value if set, zero value otherwise.
func (o *IncidentNotificationHandle) GetDisplayName() string {
	if o == nil || o.DisplayName == nil {
		var ret string
		return ret
	}
	return *o.DisplayName
}

// GetDisplayNameOk returns a tuple with the DisplayName field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IncidentNotificationHandle) GetDisplayNameOk() (*string, bool) {
	if o == nil || o.DisplayName == nil {
		return nil, false
	}
	return o.DisplayName, true
}

// HasDisplayName returns a boolean if a field has been set.
func (o *IncidentNotificationHandle) HasDisplayName() bool {
	return o != nil && o.DisplayName != nil
}

// SetDisplayName gets a reference to the given string and assigns it to the DisplayName field.
func (o *IncidentNotificationHandle) SetDisplayName(v string) {
	o.DisplayName = &v
}

// GetHandle returns the Handle field value if set, zero value otherwise.
func (o *IncidentNotificationHandle) GetHandle() string {
	if o == nil || o.Handle == nil {
		var ret string
		return ret
	}
	return *o.Handle
}

// GetHandleOk returns a tuple with the Handle field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IncidentNotificationHandle) GetHandleOk() (*string, bool) {
	if o == nil || o.Handle == nil {
		return nil, false
	}
	return o.Handle, true
}

// HasHandle returns a boolean if a field has been set.
func (o *IncidentNotificationHandle) HasHandle() bool {
	return o != nil && o.Handle != nil
}

// SetHandle gets a reference to the given string and assigns it to the Handle field.
func (o *IncidentNotificationHandle) SetHandle(v string) {
	o.Handle = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o IncidentNotificationHandle) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.DisplayName != nil {
		toSerialize["display_name"] = o.DisplayName
	}
	if o.Handle != nil {
		toSerialize["handle"] = o.Handle
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *IncidentNotificationHandle) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		DisplayName *string `json:"display_name,omitempty"`
		Handle      *string `json:"handle,omitempty"`
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
	o.DisplayName = all.DisplayName
	o.Handle = all.Handle
	return nil
}
