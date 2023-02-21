// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// FullApplicationKeyAttributes Attributes of a full application key.
type FullApplicationKeyAttributes struct {
	// Creation date of the application key.
	CreatedAt *string `json:"created_at,omitempty"`
	// The application key.
	Key *string `json:"key,omitempty"`
	// The last four characters of the application key.
	Last4 *string `json:"last4,omitempty"`
	// Name of the application key.
	Name *string `json:"name,omitempty"`
	// Array of scopes to grant the application key. This feature is in private beta, please contact Datadog support to enable scopes for your application keys.
	Scopes []string `json:"scopes,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewFullApplicationKeyAttributes instantiates a new FullApplicationKeyAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewFullApplicationKeyAttributes() *FullApplicationKeyAttributes {
	this := FullApplicationKeyAttributes{}
	return &this
}

// NewFullApplicationKeyAttributesWithDefaults instantiates a new FullApplicationKeyAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewFullApplicationKeyAttributesWithDefaults() *FullApplicationKeyAttributes {
	this := FullApplicationKeyAttributes{}
	return &this
}

// GetCreatedAt returns the CreatedAt field value if set, zero value otherwise.
func (o *FullApplicationKeyAttributes) GetCreatedAt() string {
	if o == nil || o.CreatedAt == nil {
		var ret string
		return ret
	}
	return *o.CreatedAt
}

// GetCreatedAtOk returns a tuple with the CreatedAt field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *FullApplicationKeyAttributes) GetCreatedAtOk() (*string, bool) {
	if o == nil || o.CreatedAt == nil {
		return nil, false
	}
	return o.CreatedAt, true
}

// HasCreatedAt returns a boolean if a field has been set.
func (o *FullApplicationKeyAttributes) HasCreatedAt() bool {
	return o != nil && o.CreatedAt != nil
}

// SetCreatedAt gets a reference to the given string and assigns it to the CreatedAt field.
func (o *FullApplicationKeyAttributes) SetCreatedAt(v string) {
	o.CreatedAt = &v
}

// GetKey returns the Key field value if set, zero value otherwise.
func (o *FullApplicationKeyAttributes) GetKey() string {
	if o == nil || o.Key == nil {
		var ret string
		return ret
	}
	return *o.Key
}

// GetKeyOk returns a tuple with the Key field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *FullApplicationKeyAttributes) GetKeyOk() (*string, bool) {
	if o == nil || o.Key == nil {
		return nil, false
	}
	return o.Key, true
}

// HasKey returns a boolean if a field has been set.
func (o *FullApplicationKeyAttributes) HasKey() bool {
	return o != nil && o.Key != nil
}

// SetKey gets a reference to the given string and assigns it to the Key field.
func (o *FullApplicationKeyAttributes) SetKey(v string) {
	o.Key = &v
}

// GetLast4 returns the Last4 field value if set, zero value otherwise.
func (o *FullApplicationKeyAttributes) GetLast4() string {
	if o == nil || o.Last4 == nil {
		var ret string
		return ret
	}
	return *o.Last4
}

// GetLast4Ok returns a tuple with the Last4 field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *FullApplicationKeyAttributes) GetLast4Ok() (*string, bool) {
	if o == nil || o.Last4 == nil {
		return nil, false
	}
	return o.Last4, true
}

// HasLast4 returns a boolean if a field has been set.
func (o *FullApplicationKeyAttributes) HasLast4() bool {
	return o != nil && o.Last4 != nil
}

// SetLast4 gets a reference to the given string and assigns it to the Last4 field.
func (o *FullApplicationKeyAttributes) SetLast4(v string) {
	o.Last4 = &v
}

// GetName returns the Name field value if set, zero value otherwise.
func (o *FullApplicationKeyAttributes) GetName() string {
	if o == nil || o.Name == nil {
		var ret string
		return ret
	}
	return *o.Name
}

// GetNameOk returns a tuple with the Name field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *FullApplicationKeyAttributes) GetNameOk() (*string, bool) {
	if o == nil || o.Name == nil {
		return nil, false
	}
	return o.Name, true
}

// HasName returns a boolean if a field has been set.
func (o *FullApplicationKeyAttributes) HasName() bool {
	return o != nil && o.Name != nil
}

// SetName gets a reference to the given string and assigns it to the Name field.
func (o *FullApplicationKeyAttributes) SetName(v string) {
	o.Name = &v
}

// GetScopes returns the Scopes field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *FullApplicationKeyAttributes) GetScopes() []string {
	if o == nil {
		var ret []string
		return ret
	}
	return o.Scopes
}

// GetScopesOk returns a tuple with the Scopes field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *FullApplicationKeyAttributes) GetScopesOk() (*[]string, bool) {
	if o == nil || o.Scopes == nil {
		return nil, false
	}
	return &o.Scopes, true
}

// HasScopes returns a boolean if a field has been set.
func (o *FullApplicationKeyAttributes) HasScopes() bool {
	return o != nil && o.Scopes != nil
}

// SetScopes gets a reference to the given []string and assigns it to the Scopes field.
func (o *FullApplicationKeyAttributes) SetScopes(v []string) {
	o.Scopes = v
}

// MarshalJSON serializes the struct using spec logic.
func (o FullApplicationKeyAttributes) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.CreatedAt != nil {
		toSerialize["created_at"] = o.CreatedAt
	}
	if o.Key != nil {
		toSerialize["key"] = o.Key
	}
	if o.Last4 != nil {
		toSerialize["last4"] = o.Last4
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
func (o *FullApplicationKeyAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		CreatedAt *string  `json:"created_at,omitempty"`
		Key       *string  `json:"key,omitempty"`
		Last4     *string  `json:"last4,omitempty"`
		Name      *string  `json:"name,omitempty"`
		Scopes    []string `json:"scopes,omitempty"`
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
	o.CreatedAt = all.CreatedAt
	o.Key = all.Key
	o.Last4 = all.Last4
	o.Name = all.Name
	o.Scopes = all.Scopes
	return nil
}
