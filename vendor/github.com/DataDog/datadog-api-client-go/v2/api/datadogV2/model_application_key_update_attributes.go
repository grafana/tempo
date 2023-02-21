// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// ApplicationKeyUpdateAttributes Attributes used to update an application Key.
type ApplicationKeyUpdateAttributes struct {
	// Name of the application key.
	Name *string `json:"name,omitempty"`
	// Array of scopes to grant the application key. This feature is in private beta, please contact Datadog support to enable scopes for your application keys.
	Scopes []string `json:"scopes,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewApplicationKeyUpdateAttributes instantiates a new ApplicationKeyUpdateAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewApplicationKeyUpdateAttributes() *ApplicationKeyUpdateAttributes {
	this := ApplicationKeyUpdateAttributes{}
	return &this
}

// NewApplicationKeyUpdateAttributesWithDefaults instantiates a new ApplicationKeyUpdateAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewApplicationKeyUpdateAttributesWithDefaults() *ApplicationKeyUpdateAttributes {
	this := ApplicationKeyUpdateAttributes{}
	return &this
}

// GetName returns the Name field value if set, zero value otherwise.
func (o *ApplicationKeyUpdateAttributes) GetName() string {
	if o == nil || o.Name == nil {
		var ret string
		return ret
	}
	return *o.Name
}

// GetNameOk returns a tuple with the Name field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ApplicationKeyUpdateAttributes) GetNameOk() (*string, bool) {
	if o == nil || o.Name == nil {
		return nil, false
	}
	return o.Name, true
}

// HasName returns a boolean if a field has been set.
func (o *ApplicationKeyUpdateAttributes) HasName() bool {
	return o != nil && o.Name != nil
}

// SetName gets a reference to the given string and assigns it to the Name field.
func (o *ApplicationKeyUpdateAttributes) SetName(v string) {
	o.Name = &v
}

// GetScopes returns the Scopes field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *ApplicationKeyUpdateAttributes) GetScopes() []string {
	if o == nil {
		var ret []string
		return ret
	}
	return o.Scopes
}

// GetScopesOk returns a tuple with the Scopes field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *ApplicationKeyUpdateAttributes) GetScopesOk() (*[]string, bool) {
	if o == nil || o.Scopes == nil {
		return nil, false
	}
	return &o.Scopes, true
}

// HasScopes returns a boolean if a field has been set.
func (o *ApplicationKeyUpdateAttributes) HasScopes() bool {
	return o != nil && o.Scopes != nil
}

// SetScopes gets a reference to the given []string and assigns it to the Scopes field.
func (o *ApplicationKeyUpdateAttributes) SetScopes(v []string) {
	o.Scopes = v
}

// MarshalJSON serializes the struct using spec logic.
func (o ApplicationKeyUpdateAttributes) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Name != nil {
		toSerialize["name"] = o.Name
	}
	if o.Scopes != nil {
		toSerialize["scopes"] = o.Scopes
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *ApplicationKeyUpdateAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Name   *string  `json:"name,omitempty"`
		Scopes []string `json:"scopes,omitempty"`
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
	o.Name = all.Name
	o.Scopes = all.Scopes
	return nil
}
