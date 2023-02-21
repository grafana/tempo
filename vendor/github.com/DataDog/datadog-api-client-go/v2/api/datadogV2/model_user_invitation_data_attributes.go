// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"time"
)

// UserInvitationDataAttributes Attributes of a user invitation.
type UserInvitationDataAttributes struct {
	// Creation time of the user invitation.
	CreatedAt *time.Time `json:"created_at,omitempty"`
	// Time of invitation expiration.
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	// Type of invitation.
	InviteType *string `json:"invite_type,omitempty"`
	// UUID of the user invitation.
	Uuid *string `json:"uuid,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewUserInvitationDataAttributes instantiates a new UserInvitationDataAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewUserInvitationDataAttributes() *UserInvitationDataAttributes {
	this := UserInvitationDataAttributes{}
	return &this
}

// NewUserInvitationDataAttributesWithDefaults instantiates a new UserInvitationDataAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewUserInvitationDataAttributesWithDefaults() *UserInvitationDataAttributes {
	this := UserInvitationDataAttributes{}
	return &this
}

// GetCreatedAt returns the CreatedAt field value if set, zero value otherwise.
func (o *UserInvitationDataAttributes) GetCreatedAt() time.Time {
	if o == nil || o.CreatedAt == nil {
		var ret time.Time
		return ret
	}
	return *o.CreatedAt
}

// GetCreatedAtOk returns a tuple with the CreatedAt field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UserInvitationDataAttributes) GetCreatedAtOk() (*time.Time, bool) {
	if o == nil || o.CreatedAt == nil {
		return nil, false
	}
	return o.CreatedAt, true
}

// HasCreatedAt returns a boolean if a field has been set.
func (o *UserInvitationDataAttributes) HasCreatedAt() bool {
	return o != nil && o.CreatedAt != nil
}

// SetCreatedAt gets a reference to the given time.Time and assigns it to the CreatedAt field.
func (o *UserInvitationDataAttributes) SetCreatedAt(v time.Time) {
	o.CreatedAt = &v
}

// GetExpiresAt returns the ExpiresAt field value if set, zero value otherwise.
func (o *UserInvitationDataAttributes) GetExpiresAt() time.Time {
	if o == nil || o.ExpiresAt == nil {
		var ret time.Time
		return ret
	}
	return *o.ExpiresAt
}

// GetExpiresAtOk returns a tuple with the ExpiresAt field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UserInvitationDataAttributes) GetExpiresAtOk() (*time.Time, bool) {
	if o == nil || o.ExpiresAt == nil {
		return nil, false
	}
	return o.ExpiresAt, true
}

// HasExpiresAt returns a boolean if a field has been set.
func (o *UserInvitationDataAttributes) HasExpiresAt() bool {
	return o != nil && o.ExpiresAt != nil
}

// SetExpiresAt gets a reference to the given time.Time and assigns it to the ExpiresAt field.
func (o *UserInvitationDataAttributes) SetExpiresAt(v time.Time) {
	o.ExpiresAt = &v
}

// GetInviteType returns the InviteType field value if set, zero value otherwise.
func (o *UserInvitationDataAttributes) GetInviteType() string {
	if o == nil || o.InviteType == nil {
		var ret string
		return ret
	}
	return *o.InviteType
}

// GetInviteTypeOk returns a tuple with the InviteType field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UserInvitationDataAttributes) GetInviteTypeOk() (*string, bool) {
	if o == nil || o.InviteType == nil {
		return nil, false
	}
	return o.InviteType, true
}

// HasInviteType returns a boolean if a field has been set.
func (o *UserInvitationDataAttributes) HasInviteType() bool {
	return o != nil && o.InviteType != nil
}

// SetInviteType gets a reference to the given string and assigns it to the InviteType field.
func (o *UserInvitationDataAttributes) SetInviteType(v string) {
	o.InviteType = &v
}

// GetUuid returns the Uuid field value if set, zero value otherwise.
func (o *UserInvitationDataAttributes) GetUuid() string {
	if o == nil || o.Uuid == nil {
		var ret string
		return ret
	}
	return *o.Uuid
}

// GetUuidOk returns a tuple with the Uuid field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UserInvitationDataAttributes) GetUuidOk() (*string, bool) {
	if o == nil || o.Uuid == nil {
		return nil, false
	}
	return o.Uuid, true
}

// HasUuid returns a boolean if a field has been set.
func (o *UserInvitationDataAttributes) HasUuid() bool {
	return o != nil && o.Uuid != nil
}

// SetUuid gets a reference to the given string and assigns it to the Uuid field.
func (o *UserInvitationDataAttributes) SetUuid(v string) {
	o.Uuid = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o UserInvitationDataAttributes) MarshalJSON() ([]byte, error) {
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
	if o.ExpiresAt != nil {
		if o.ExpiresAt.Nanosecond() == 0 {
			toSerialize["expires_at"] = o.ExpiresAt.Format("2006-01-02T15:04:05Z07:00")
		} else {
			toSerialize["expires_at"] = o.ExpiresAt.Format("2006-01-02T15:04:05.000Z07:00")
		}
	}
	if o.InviteType != nil {
		toSerialize["invite_type"] = o.InviteType
	}
	if o.Uuid != nil {
		toSerialize["uuid"] = o.Uuid
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *UserInvitationDataAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		CreatedAt  *time.Time `json:"created_at,omitempty"`
		ExpiresAt  *time.Time `json:"expires_at,omitempty"`
		InviteType *string    `json:"invite_type,omitempty"`
		Uuid       *string    `json:"uuid,omitempty"`
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
	o.ExpiresAt = all.ExpiresAt
	o.InviteType = all.InviteType
	o.Uuid = all.Uuid
	return nil
}
