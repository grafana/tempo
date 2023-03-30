// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"time"
)

// IPAllowlistEntryAttributes Attributes of the IP allowlist entry.
type IPAllowlistEntryAttributes struct {
	// The CIDR block describing the IP range of the entry.
	CidrBlock *string `json:"cidr_block,omitempty"`
	// Creation time of the entry.
	CreatedAt *time.Time `json:"created_at,omitempty"`
	// Time of last entry modification.
	ModifiedAt *time.Time `json:"modified_at,omitempty"`
	// A note describing the IP allowlist entry.
	Note *string `json:"note,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewIPAllowlistEntryAttributes instantiates a new IPAllowlistEntryAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewIPAllowlistEntryAttributes() *IPAllowlistEntryAttributes {
	this := IPAllowlistEntryAttributes{}
	return &this
}

// NewIPAllowlistEntryAttributesWithDefaults instantiates a new IPAllowlistEntryAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewIPAllowlistEntryAttributesWithDefaults() *IPAllowlistEntryAttributes {
	this := IPAllowlistEntryAttributes{}
	return &this
}

// GetCidrBlock returns the CidrBlock field value if set, zero value otherwise.
func (o *IPAllowlistEntryAttributes) GetCidrBlock() string {
	if o == nil || o.CidrBlock == nil {
		var ret string
		return ret
	}
	return *o.CidrBlock
}

// GetCidrBlockOk returns a tuple with the CidrBlock field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IPAllowlistEntryAttributes) GetCidrBlockOk() (*string, bool) {
	if o == nil || o.CidrBlock == nil {
		return nil, false
	}
	return o.CidrBlock, true
}

// HasCidrBlock returns a boolean if a field has been set.
func (o *IPAllowlistEntryAttributes) HasCidrBlock() bool {
	return o != nil && o.CidrBlock != nil
}

// SetCidrBlock gets a reference to the given string and assigns it to the CidrBlock field.
func (o *IPAllowlistEntryAttributes) SetCidrBlock(v string) {
	o.CidrBlock = &v
}

// GetCreatedAt returns the CreatedAt field value if set, zero value otherwise.
func (o *IPAllowlistEntryAttributes) GetCreatedAt() time.Time {
	if o == nil || o.CreatedAt == nil {
		var ret time.Time
		return ret
	}
	return *o.CreatedAt
}

// GetCreatedAtOk returns a tuple with the CreatedAt field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IPAllowlistEntryAttributes) GetCreatedAtOk() (*time.Time, bool) {
	if o == nil || o.CreatedAt == nil {
		return nil, false
	}
	return o.CreatedAt, true
}

// HasCreatedAt returns a boolean if a field has been set.
func (o *IPAllowlistEntryAttributes) HasCreatedAt() bool {
	return o != nil && o.CreatedAt != nil
}

// SetCreatedAt gets a reference to the given time.Time and assigns it to the CreatedAt field.
func (o *IPAllowlistEntryAttributes) SetCreatedAt(v time.Time) {
	o.CreatedAt = &v
}

// GetModifiedAt returns the ModifiedAt field value if set, zero value otherwise.
func (o *IPAllowlistEntryAttributes) GetModifiedAt() time.Time {
	if o == nil || o.ModifiedAt == nil {
		var ret time.Time
		return ret
	}
	return *o.ModifiedAt
}

// GetModifiedAtOk returns a tuple with the ModifiedAt field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IPAllowlistEntryAttributes) GetModifiedAtOk() (*time.Time, bool) {
	if o == nil || o.ModifiedAt == nil {
		return nil, false
	}
	return o.ModifiedAt, true
}

// HasModifiedAt returns a boolean if a field has been set.
func (o *IPAllowlistEntryAttributes) HasModifiedAt() bool {
	return o != nil && o.ModifiedAt != nil
}

// SetModifiedAt gets a reference to the given time.Time and assigns it to the ModifiedAt field.
func (o *IPAllowlistEntryAttributes) SetModifiedAt(v time.Time) {
	o.ModifiedAt = &v
}

// GetNote returns the Note field value if set, zero value otherwise.
func (o *IPAllowlistEntryAttributes) GetNote() string {
	if o == nil || o.Note == nil {
		var ret string
		return ret
	}
	return *o.Note
}

// GetNoteOk returns a tuple with the Note field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IPAllowlistEntryAttributes) GetNoteOk() (*string, bool) {
	if o == nil || o.Note == nil {
		return nil, false
	}
	return o.Note, true
}

// HasNote returns a boolean if a field has been set.
func (o *IPAllowlistEntryAttributes) HasNote() bool {
	return o != nil && o.Note != nil
}

// SetNote gets a reference to the given string and assigns it to the Note field.
func (o *IPAllowlistEntryAttributes) SetNote(v string) {
	o.Note = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o IPAllowlistEntryAttributes) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.CidrBlock != nil {
		toSerialize["cidr_block"] = o.CidrBlock
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
	if o.Note != nil {
		toSerialize["note"] = o.Note
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *IPAllowlistEntryAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		CidrBlock  *string    `json:"cidr_block,omitempty"`
		CreatedAt  *time.Time `json:"created_at,omitempty"`
		ModifiedAt *time.Time `json:"modified_at,omitempty"`
		Note       *string    `json:"note,omitempty"`
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
	o.CidrBlock = all.CidrBlock
	o.CreatedAt = all.CreatedAt
	o.ModifiedAt = all.ModifiedAt
	o.Note = all.Note
	return nil
}
