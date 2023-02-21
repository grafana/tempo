// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// UserCreateAttributes Attributes of the created user.
type UserCreateAttributes struct {
	// The email of the user.
	Email string `json:"email"`
	// The name of the user.
	Name *string `json:"name,omitempty"`
	// The title of the user.
	Title *string `json:"title,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewUserCreateAttributes instantiates a new UserCreateAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewUserCreateAttributes(email string) *UserCreateAttributes {
	this := UserCreateAttributes{}
	this.Email = email
	return &this
}

// NewUserCreateAttributesWithDefaults instantiates a new UserCreateAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewUserCreateAttributesWithDefaults() *UserCreateAttributes {
	this := UserCreateAttributes{}
	return &this
}

// GetEmail returns the Email field value.
func (o *UserCreateAttributes) GetEmail() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Email
}

// GetEmailOk returns a tuple with the Email field value
// and a boolean to check if the value has been set.
func (o *UserCreateAttributes) GetEmailOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Email, true
}

// SetEmail sets field value.
func (o *UserCreateAttributes) SetEmail(v string) {
	o.Email = v
}

// GetName returns the Name field value if set, zero value otherwise.
func (o *UserCreateAttributes) GetName() string {
	if o == nil || o.Name == nil {
		var ret string
		return ret
	}
	return *o.Name
}

// GetNameOk returns a tuple with the Name field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UserCreateAttributes) GetNameOk() (*string, bool) {
	if o == nil || o.Name == nil {
		return nil, false
	}
	return o.Name, true
}

// HasName returns a boolean if a field has been set.
func (o *UserCreateAttributes) HasName() bool {
	return o != nil && o.Name != nil
}

// SetName gets a reference to the given string and assigns it to the Name field.
func (o *UserCreateAttributes) SetName(v string) {
	o.Name = &v
}

// GetTitle returns the Title field value if set, zero value otherwise.
func (o *UserCreateAttributes) GetTitle() string {
	if o == nil || o.Title == nil {
		var ret string
		return ret
	}
	return *o.Title
}

// GetTitleOk returns a tuple with the Title field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UserCreateAttributes) GetTitleOk() (*string, bool) {
	if o == nil || o.Title == nil {
		return nil, false
	}
	return o.Title, true
}

// HasTitle returns a boolean if a field has been set.
func (o *UserCreateAttributes) HasTitle() bool {
	return o != nil && o.Title != nil
}

// SetTitle gets a reference to the given string and assigns it to the Title field.
func (o *UserCreateAttributes) SetTitle(v string) {
	o.Title = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o UserCreateAttributes) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["email"] = o.Email
	if o.Name != nil {
		toSerialize["name"] = o.Name
	}
	if o.Title != nil {
		toSerialize["title"] = o.Title
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *UserCreateAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Email *string `json:"email"`
	}{}
	all := struct {
		Email string  `json:"email"`
		Name  *string `json:"name,omitempty"`
		Title *string `json:"title,omitempty"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Email == nil {
		return fmt.Errorf("required field email missing")
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
	o.Email = all.Email
	o.Name = all.Name
	o.Title = all.Title
	return nil
}
