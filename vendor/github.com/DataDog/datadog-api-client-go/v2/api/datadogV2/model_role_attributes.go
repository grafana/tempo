// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"time"
)

// RoleAttributes Attributes of the role.
type RoleAttributes struct {
	// Creation time of the role.
	CreatedAt *time.Time `json:"created_at,omitempty"`
	// Time of last role modification.
	ModifiedAt *time.Time `json:"modified_at,omitempty"`
	// The name of the role. The name is neither unique nor a stable identifier of the role.
	Name *string `json:"name,omitempty"`
	// Number of users with that role.
	UserCount *int64 `json:"user_count,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewRoleAttributes instantiates a new RoleAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewRoleAttributes() *RoleAttributes {
	this := RoleAttributes{}
	return &this
}

// NewRoleAttributesWithDefaults instantiates a new RoleAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewRoleAttributesWithDefaults() *RoleAttributes {
	this := RoleAttributes{}
	return &this
}

// GetCreatedAt returns the CreatedAt field value if set, zero value otherwise.
func (o *RoleAttributes) GetCreatedAt() time.Time {
	if o == nil || o.CreatedAt == nil {
		var ret time.Time
		return ret
	}
	return *o.CreatedAt
}

// GetCreatedAtOk returns a tuple with the CreatedAt field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *RoleAttributes) GetCreatedAtOk() (*time.Time, bool) {
	if o == nil || o.CreatedAt == nil {
		return nil, false
	}
	return o.CreatedAt, true
}

// HasCreatedAt returns a boolean if a field has been set.
func (o *RoleAttributes) HasCreatedAt() bool {
	return o != nil && o.CreatedAt != nil
}

// SetCreatedAt gets a reference to the given time.Time and assigns it to the CreatedAt field.
func (o *RoleAttributes) SetCreatedAt(v time.Time) {
	o.CreatedAt = &v
}

// GetModifiedAt returns the ModifiedAt field value if set, zero value otherwise.
func (o *RoleAttributes) GetModifiedAt() time.Time {
	if o == nil || o.ModifiedAt == nil {
		var ret time.Time
		return ret
	}
	return *o.ModifiedAt
}

// GetModifiedAtOk returns a tuple with the ModifiedAt field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *RoleAttributes) GetModifiedAtOk() (*time.Time, bool) {
	if o == nil || o.ModifiedAt == nil {
		return nil, false
	}
	return o.ModifiedAt, true
}

// HasModifiedAt returns a boolean if a field has been set.
func (o *RoleAttributes) HasModifiedAt() bool {
	return o != nil && o.ModifiedAt != nil
}

// SetModifiedAt gets a reference to the given time.Time and assigns it to the ModifiedAt field.
func (o *RoleAttributes) SetModifiedAt(v time.Time) {
	o.ModifiedAt = &v
}

// GetName returns the Name field value if set, zero value otherwise.
func (o *RoleAttributes) GetName() string {
	if o == nil || o.Name == nil {
		var ret string
		return ret
	}
	return *o.Name
}

// GetNameOk returns a tuple with the Name field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *RoleAttributes) GetNameOk() (*string, bool) {
	if o == nil || o.Name == nil {
		return nil, false
	}
	return o.Name, true
}

// HasName returns a boolean if a field has been set.
func (o *RoleAttributes) HasName() bool {
	return o != nil && o.Name != nil
}

// SetName gets a reference to the given string and assigns it to the Name field.
func (o *RoleAttributes) SetName(v string) {
	o.Name = &v
}

// GetUserCount returns the UserCount field value if set, zero value otherwise.
func (o *RoleAttributes) GetUserCount() int64 {
	if o == nil || o.UserCount == nil {
		var ret int64
		return ret
	}
	return *o.UserCount
}

// GetUserCountOk returns a tuple with the UserCount field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *RoleAttributes) GetUserCountOk() (*int64, bool) {
	if o == nil || o.UserCount == nil {
		return nil, false
	}
	return o.UserCount, true
}

// HasUserCount returns a boolean if a field has been set.
func (o *RoleAttributes) HasUserCount() bool {
	return o != nil && o.UserCount != nil
}

// SetUserCount gets a reference to the given int64 and assigns it to the UserCount field.
func (o *RoleAttributes) SetUserCount(v int64) {
	o.UserCount = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o RoleAttributes) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.CreatedAt != nil {
		if o.CreatedAt.Nanosecond() == 0 {
			toSerialize["created_at"] = o.CreatedAt.Format("2006-01-02T15:04:05Z07:00")
		} else {
			toSerialize["created_at"] = o.CreatedAt.Format("2006-01-02T15:04:05.000Z07:00")
		}
	}
	if o.ModifiedAt != nil {
		if o.ModifiedAt.Nanosecond() == 0 {
			toSerialize["modified_at"] = o.ModifiedAt.Format("2006-01-02T15:04:05Z07:00")
		} else {
			toSerialize["modified_at"] = o.ModifiedAt.Format("2006-01-02T15:04:05.000Z07:00")
		}
	}
	if o.Name != nil {
		toSerialize["name"] = o.Name
	}
	if o.UserCount != nil {
		toSerialize["user_count"] = o.UserCount
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *RoleAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		CreatedAt  *time.Time `json:"created_at,omitempty"`
		ModifiedAt *time.Time `json:"modified_at,omitempty"`
		Name       *string    `json:"name,omitempty"`
		UserCount  *int64     `json:"user_count,omitempty"`
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
	o.ModifiedAt = all.ModifiedAt
	o.Name = all.Name
	o.UserCount = all.UserCount
	return nil
}
