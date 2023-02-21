// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// FullAPIKeyAttributes Attributes of a full API key.
type FullAPIKeyAttributes struct {
	// Creation date of the API key.
	CreatedAt *string `json:"created_at,omitempty"`
	// The API key.
	Key *string `json:"key,omitempty"`
	// The last four characters of the API key.
	Last4 *string `json:"last4,omitempty"`
	// Date the API key was last modified.
	ModifiedAt *string `json:"modified_at,omitempty"`
	// Name of the API key.
	Name *string `json:"name,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewFullAPIKeyAttributes instantiates a new FullAPIKeyAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewFullAPIKeyAttributes() *FullAPIKeyAttributes {
	this := FullAPIKeyAttributes{}
	return &this
}

// NewFullAPIKeyAttributesWithDefaults instantiates a new FullAPIKeyAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewFullAPIKeyAttributesWithDefaults() *FullAPIKeyAttributes {
	this := FullAPIKeyAttributes{}
	return &this
}

// GetCreatedAt returns the CreatedAt field value if set, zero value otherwise.
func (o *FullAPIKeyAttributes) GetCreatedAt() string {
	if o == nil || o.CreatedAt == nil {
		var ret string
		return ret
	}
	return *o.CreatedAt
}

// GetCreatedAtOk returns a tuple with the CreatedAt field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *FullAPIKeyAttributes) GetCreatedAtOk() (*string, bool) {
	if o == nil || o.CreatedAt == nil {
		return nil, false
	}
	return o.CreatedAt, true
}

// HasCreatedAt returns a boolean if a field has been set.
func (o *FullAPIKeyAttributes) HasCreatedAt() bool {
	return o != nil && o.CreatedAt != nil
}

// SetCreatedAt gets a reference to the given string and assigns it to the CreatedAt field.
func (o *FullAPIKeyAttributes) SetCreatedAt(v string) {
	o.CreatedAt = &v
}

// GetKey returns the Key field value if set, zero value otherwise.
func (o *FullAPIKeyAttributes) GetKey() string {
	if o == nil || o.Key == nil {
		var ret string
		return ret
	}
	return *o.Key
}

// GetKeyOk returns a tuple with the Key field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *FullAPIKeyAttributes) GetKeyOk() (*string, bool) {
	if o == nil || o.Key == nil {
		return nil, false
	}
	return o.Key, true
}

// HasKey returns a boolean if a field has been set.
func (o *FullAPIKeyAttributes) HasKey() bool {
	return o != nil && o.Key != nil
}

// SetKey gets a reference to the given string and assigns it to the Key field.
func (o *FullAPIKeyAttributes) SetKey(v string) {
	o.Key = &v
}

// GetLast4 returns the Last4 field value if set, zero value otherwise.
func (o *FullAPIKeyAttributes) GetLast4() string {
	if o == nil || o.Last4 == nil {
		var ret string
		return ret
	}
	return *o.Last4
}

// GetLast4Ok returns a tuple with the Last4 field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *FullAPIKeyAttributes) GetLast4Ok() (*string, bool) {
	if o == nil || o.Last4 == nil {
		return nil, false
	}
	return o.Last4, true
}

// HasLast4 returns a boolean if a field has been set.
func (o *FullAPIKeyAttributes) HasLast4() bool {
	return o != nil && o.Last4 != nil
}

// SetLast4 gets a reference to the given string and assigns it to the Last4 field.
func (o *FullAPIKeyAttributes) SetLast4(v string) {
	o.Last4 = &v
}

// GetModifiedAt returns the ModifiedAt field value if set, zero value otherwise.
func (o *FullAPIKeyAttributes) GetModifiedAt() string {
	if o == nil || o.ModifiedAt == nil {
		var ret string
		return ret
	}
	return *o.ModifiedAt
}

// GetModifiedAtOk returns a tuple with the ModifiedAt field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *FullAPIKeyAttributes) GetModifiedAtOk() (*string, bool) {
	if o == nil || o.ModifiedAt == nil {
		return nil, false
	}
	return o.ModifiedAt, true
}

// HasModifiedAt returns a boolean if a field has been set.
func (o *FullAPIKeyAttributes) HasModifiedAt() bool {
	return o != nil && o.ModifiedAt != nil
}

// SetModifiedAt gets a reference to the given string and assigns it to the ModifiedAt field.
func (o *FullAPIKeyAttributes) SetModifiedAt(v string) {
	o.ModifiedAt = &v
}

// GetName returns the Name field value if set, zero value otherwise.
func (o *FullAPIKeyAttributes) GetName() string {
	if o == nil || o.Name == nil {
		var ret string
		return ret
	}
	return *o.Name
}

// GetNameOk returns a tuple with the Name field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *FullAPIKeyAttributes) GetNameOk() (*string, bool) {
	if o == nil || o.Name == nil {
		return nil, false
	}
	return o.Name, true
}

// HasName returns a boolean if a field has been set.
func (o *FullAPIKeyAttributes) HasName() bool {
	return o != nil && o.Name != nil
}

// SetName gets a reference to the given string and assigns it to the Name field.
func (o *FullAPIKeyAttributes) SetName(v string) {
	o.Name = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o FullAPIKeyAttributes) MarshalJSON() ([]byte, error) {
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
	if o.ModifiedAt != nil {
		toSerialize["modified_at"] = o.ModifiedAt
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
func (o *FullAPIKeyAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		CreatedAt  *string `json:"created_at,omitempty"`
		Key        *string `json:"key,omitempty"`
		Last4      *string `json:"last4,omitempty"`
		ModifiedAt *string `json:"modified_at,omitempty"`
		Name       *string `json:"name,omitempty"`
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
	o.ModifiedAt = all.ModifiedAt
	o.Name = all.Name
	return nil
}
