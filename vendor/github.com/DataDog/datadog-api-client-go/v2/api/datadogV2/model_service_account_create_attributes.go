// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// ServiceAccountCreateAttributes Attributes of the created user.
type ServiceAccountCreateAttributes struct {
	// The email of the user.
	Email string `json:"email"`
	// The name of the user.
	Name *string `json:"name,omitempty"`
	// Whether the user is a service account. Must be true.
	ServiceAccount bool `json:"service_account"`
	// The title of the user.
	Title *string `json:"title,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewServiceAccountCreateAttributes instantiates a new ServiceAccountCreateAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewServiceAccountCreateAttributes(email string, serviceAccount bool) *ServiceAccountCreateAttributes {
	this := ServiceAccountCreateAttributes{}
	this.Email = email
	this.ServiceAccount = serviceAccount
	return &this
}

// NewServiceAccountCreateAttributesWithDefaults instantiates a new ServiceAccountCreateAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewServiceAccountCreateAttributesWithDefaults() *ServiceAccountCreateAttributes {
	this := ServiceAccountCreateAttributes{}
	return &this
}

// GetEmail returns the Email field value.
func (o *ServiceAccountCreateAttributes) GetEmail() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Email
}

// GetEmailOk returns a tuple with the Email field value
// and a boolean to check if the value has been set.
func (o *ServiceAccountCreateAttributes) GetEmailOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Email, true
}

// SetEmail sets field value.
func (o *ServiceAccountCreateAttributes) SetEmail(v string) {
	o.Email = v
}

// GetName returns the Name field value if set, zero value otherwise.
func (o *ServiceAccountCreateAttributes) GetName() string {
	if o == nil || o.Name == nil {
		var ret string
		return ret
	}
	return *o.Name
}

// GetNameOk returns a tuple with the Name field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ServiceAccountCreateAttributes) GetNameOk() (*string, bool) {
	if o == nil || o.Name == nil {
		return nil, false
	}
	return o.Name, true
}

// HasName returns a boolean if a field has been set.
func (o *ServiceAccountCreateAttributes) HasName() bool {
	return o != nil && o.Name != nil
}

// SetName gets a reference to the given string and assigns it to the Name field.
func (o *ServiceAccountCreateAttributes) SetName(v string) {
	o.Name = &v
}

// GetServiceAccount returns the ServiceAccount field value.
func (o *ServiceAccountCreateAttributes) GetServiceAccount() bool {
	if o == nil {
		var ret bool
		return ret
	}
	return o.ServiceAccount
}

// GetServiceAccountOk returns a tuple with the ServiceAccount field value
// and a boolean to check if the value has been set.
func (o *ServiceAccountCreateAttributes) GetServiceAccountOk() (*bool, bool) {
	if o == nil {
		return nil, false
	}
	return &o.ServiceAccount, true
}

// SetServiceAccount sets field value.
func (o *ServiceAccountCreateAttributes) SetServiceAccount(v bool) {
	o.ServiceAccount = v
}

// GetTitle returns the Title field value if set, zero value otherwise.
func (o *ServiceAccountCreateAttributes) GetTitle() string {
	if o == nil || o.Title == nil {
		var ret string
		return ret
	}
	return *o.Title
}

// GetTitleOk returns a tuple with the Title field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ServiceAccountCreateAttributes) GetTitleOk() (*string, bool) {
	if o == nil || o.Title == nil {
		return nil, false
	}
	return o.Title, true
}

// HasTitle returns a boolean if a field has been set.
func (o *ServiceAccountCreateAttributes) HasTitle() bool {
	return o != nil && o.Title != nil
}

// SetTitle gets a reference to the given string and assigns it to the Title field.
func (o *ServiceAccountCreateAttributes) SetTitle(v string) {
	o.Title = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o ServiceAccountCreateAttributes) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["email"] = o.Email
	if o.Name != nil {
		toSerialize["name"] = o.Name
	}
	toSerialize["service_account"] = o.ServiceAccount
	if o.Title != nil {
		toSerialize["title"] = o.Title
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *ServiceAccountCreateAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Email          *string `json:"email"`
		ServiceAccount *bool   `json:"service_account"`
	}{}
	all := struct {
		Email          string  `json:"email"`
		Name           *string `json:"name,omitempty"`
		ServiceAccount bool    `json:"service_account"`
		Title          *string `json:"title,omitempty"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Email == nil {
		return fmt.Errorf("required field email missing")
	}
	if required.ServiceAccount == nil {
		return fmt.Errorf("required field service_account missing")
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
	o.ServiceAccount = all.ServiceAccount
	o.Title = all.Title
	return nil
}
