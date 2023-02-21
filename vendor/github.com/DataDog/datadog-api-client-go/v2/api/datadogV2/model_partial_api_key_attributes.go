// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// PartialAPIKeyAttributes Attributes of a partial API key.
type PartialAPIKeyAttributes struct {
	// Creation date of the API key.
	CreatedAt *string `json:"created_at,omitempty"`
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

// NewPartialAPIKeyAttributes instantiates a new PartialAPIKeyAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewPartialAPIKeyAttributes() *PartialAPIKeyAttributes {
	this := PartialAPIKeyAttributes{}
	return &this
}

// NewPartialAPIKeyAttributesWithDefaults instantiates a new PartialAPIKeyAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewPartialAPIKeyAttributesWithDefaults() *PartialAPIKeyAttributes {
	this := PartialAPIKeyAttributes{}
	return &this
}

// GetCreatedAt returns the CreatedAt field value if set, zero value otherwise.
func (o *PartialAPIKeyAttributes) GetCreatedAt() string {
	if o == nil || o.CreatedAt == nil {
		var ret string
		return ret
	}
	return *o.CreatedAt
}

// GetCreatedAtOk returns a tuple with the CreatedAt field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *PartialAPIKeyAttributes) GetCreatedAtOk() (*string, bool) {
	if o == nil || o.CreatedAt == nil {
		return nil, false
	}
	return o.CreatedAt, true
}

// HasCreatedAt returns a boolean if a field has been set.
func (o *PartialAPIKeyAttributes) HasCreatedAt() bool {
	return o != nil && o.CreatedAt != nil
}

// SetCreatedAt gets a reference to the given string and assigns it to the CreatedAt field.
func (o *PartialAPIKeyAttributes) SetCreatedAt(v string) {
	o.CreatedAt = &v
}

// GetLast4 returns the Last4 field value if set, zero value otherwise.
func (o *PartialAPIKeyAttributes) GetLast4() string {
	if o == nil || o.Last4 == nil {
		var ret string
		return ret
	}
	return *o.Last4
}

// GetLast4Ok returns a tuple with the Last4 field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *PartialAPIKeyAttributes) GetLast4Ok() (*string, bool) {
	if o == nil || o.Last4 == nil {
		return nil, false
	}
	return o.Last4, true
}

// HasLast4 returns a boolean if a field has been set.
func (o *PartialAPIKeyAttributes) HasLast4() bool {
	return o != nil && o.Last4 != nil
}

// SetLast4 gets a reference to the given string and assigns it to the Last4 field.
func (o *PartialAPIKeyAttributes) SetLast4(v string) {
	o.Last4 = &v
}

// GetModifiedAt returns the ModifiedAt field value if set, zero value otherwise.
func (o *PartialAPIKeyAttributes) GetModifiedAt() string {
	if o == nil || o.ModifiedAt == nil {
		var ret string
		return ret
	}
	return *o.ModifiedAt
}

// GetModifiedAtOk returns a tuple with the ModifiedAt field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *PartialAPIKeyAttributes) GetModifiedAtOk() (*string, bool) {
	if o == nil || o.ModifiedAt == nil {
		return nil, false
	}
	return o.ModifiedAt, true
}

// HasModifiedAt returns a boolean if a field has been set.
func (o *PartialAPIKeyAttributes) HasModifiedAt() bool {
	return o != nil && o.ModifiedAt != nil
}

// SetModifiedAt gets a reference to the given string and assigns it to the ModifiedAt field.
func (o *PartialAPIKeyAttributes) SetModifiedAt(v string) {
	o.ModifiedAt = &v
}

// GetName returns the Name field value if set, zero value otherwise.
func (o *PartialAPIKeyAttributes) GetName() string {
	if o == nil || o.Name == nil {
		var ret string
		return ret
	}
	return *o.Name
}

// GetNameOk returns a tuple with the Name field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *PartialAPIKeyAttributes) GetNameOk() (*string, bool) {
	if o == nil || o.Name == nil {
		return nil, false
	}
	return o.Name, true
}

// HasName returns a boolean if a field has been set.
func (o *PartialAPIKeyAttributes) HasName() bool {
	return o != nil && o.Name != nil
}

// SetName gets a reference to the given string and assigns it to the Name field.
func (o *PartialAPIKeyAttributes) SetName(v string) {
	o.Name = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o PartialAPIKeyAttributes) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.CreatedAt != nil {
		toSerialize["created_at"] = o.CreatedAt
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
func (o *PartialAPIKeyAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		CreatedAt  *string `json:"created_at,omitempty"`
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
	o.Last4 = all.Last4
	o.ModifiedAt = all.ModifiedAt
	o.Name = all.Name
	return nil
}
