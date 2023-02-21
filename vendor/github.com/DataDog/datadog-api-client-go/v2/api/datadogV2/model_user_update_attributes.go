// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// UserUpdateAttributes Attributes of the edited user.
type UserUpdateAttributes struct {
	// If the user is enabled or disabled.
	Disabled *bool `json:"disabled,omitempty"`
	// The email of the user.
	Email *string `json:"email,omitempty"`
	// The name of the user.
	Name *string `json:"name,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewUserUpdateAttributes instantiates a new UserUpdateAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewUserUpdateAttributes() *UserUpdateAttributes {
	this := UserUpdateAttributes{}
	return &this
}

// NewUserUpdateAttributesWithDefaults instantiates a new UserUpdateAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewUserUpdateAttributesWithDefaults() *UserUpdateAttributes {
	this := UserUpdateAttributes{}
	return &this
}

// GetDisabled returns the Disabled field value if set, zero value otherwise.
func (o *UserUpdateAttributes) GetDisabled() bool {
	if o == nil || o.Disabled == nil {
		var ret bool
		return ret
	}
	return *o.Disabled
}

// GetDisabledOk returns a tuple with the Disabled field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UserUpdateAttributes) GetDisabledOk() (*bool, bool) {
	if o == nil || o.Disabled == nil {
		return nil, false
	}
	return o.Disabled, true
}

// HasDisabled returns a boolean if a field has been set.
func (o *UserUpdateAttributes) HasDisabled() bool {
	return o != nil && o.Disabled != nil
}

// SetDisabled gets a reference to the given bool and assigns it to the Disabled field.
func (o *UserUpdateAttributes) SetDisabled(v bool) {
	o.Disabled = &v
}

// GetEmail returns the Email field value if set, zero value otherwise.
func (o *UserUpdateAttributes) GetEmail() string {
	if o == nil || o.Email == nil {
		var ret string
		return ret
	}
	return *o.Email
}

// GetEmailOk returns a tuple with the Email field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UserUpdateAttributes) GetEmailOk() (*string, bool) {
	if o == nil || o.Email == nil {
		return nil, false
	}
	return o.Email, true
}

// HasEmail returns a boolean if a field has been set.
func (o *UserUpdateAttributes) HasEmail() bool {
	return o != nil && o.Email != nil
}

// SetEmail gets a reference to the given string and assigns it to the Email field.
func (o *UserUpdateAttributes) SetEmail(v string) {
	o.Email = &v
}

// GetName returns the Name field value if set, zero value otherwise.
func (o *UserUpdateAttributes) GetName() string {
	if o == nil || o.Name == nil {
		var ret string
		return ret
	}
	return *o.Name
}

// GetNameOk returns a tuple with the Name field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UserUpdateAttributes) GetNameOk() (*string, bool) {
	if o == nil || o.Name == nil {
		return nil, false
	}
	return o.Name, true
}

// HasName returns a boolean if a field has been set.
func (o *UserUpdateAttributes) HasName() bool {
	return o != nil && o.Name != nil
}

// SetName gets a reference to the given string and assigns it to the Name field.
func (o *UserUpdateAttributes) SetName(v string) {
	o.Name = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o UserUpdateAttributes) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Disabled != nil {
		toSerialize["disabled"] = o.Disabled
	}
	if o.Email != nil {
		toSerialize["email"] = o.Email
	}
	if o.Name != nil {
		toSerialize["name"] = o.Name
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *UserUpdateAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Disabled *bool   `json:"disabled,omitempty"`
		Email    *string `json:"email,omitempty"`
		Name     *string `json:"name,omitempty"`
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
	o.Disabled = all.Disabled
	o.Email = all.Email
	o.Name = all.Name
	return nil
}
