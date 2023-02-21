// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"time"
)

// OrganizationAttributes Attributes of the organization.
type OrganizationAttributes struct {
	// Creation time of the organization.
	CreatedAt *time.Time `json:"created_at,omitempty"`
	// Description of the organization.
	Description *string `json:"description,omitempty"`
	// Whether or not the organization is disabled.
	Disabled *bool `json:"disabled,omitempty"`
	// Time of last organization modification.
	ModifiedAt *time.Time `json:"modified_at,omitempty"`
	// Name of the organization.
	Name *string `json:"name,omitempty"`
	// Public ID of the organization.
	PublicId *string `json:"public_id,omitempty"`
	// Sharing type of the organization.
	Sharing *string `json:"sharing,omitempty"`
	// URL of the site that this organization exists at.
	Url *string `json:"url,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewOrganizationAttributes instantiates a new OrganizationAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewOrganizationAttributes() *OrganizationAttributes {
	this := OrganizationAttributes{}
	return &this
}

// NewOrganizationAttributesWithDefaults instantiates a new OrganizationAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewOrganizationAttributesWithDefaults() *OrganizationAttributes {
	this := OrganizationAttributes{}
	return &this
}

// GetCreatedAt returns the CreatedAt field value if set, zero value otherwise.
func (o *OrganizationAttributes) GetCreatedAt() time.Time {
	if o == nil || o.CreatedAt == nil {
		var ret time.Time
		return ret
	}
	return *o.CreatedAt
}

// GetCreatedAtOk returns a tuple with the CreatedAt field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *OrganizationAttributes) GetCreatedAtOk() (*time.Time, bool) {
	if o == nil || o.CreatedAt == nil {
		return nil, false
	}
	return o.CreatedAt, true
}

// HasCreatedAt returns a boolean if a field has been set.
func (o *OrganizationAttributes) HasCreatedAt() bool {
	return o != nil && o.CreatedAt != nil
}

// SetCreatedAt gets a reference to the given time.Time and assigns it to the CreatedAt field.
func (o *OrganizationAttributes) SetCreatedAt(v time.Time) {
	o.CreatedAt = &v
}

// GetDescription returns the Description field value if set, zero value otherwise.
func (o *OrganizationAttributes) GetDescription() string {
	if o == nil || o.Description == nil {
		var ret string
		return ret
	}
	return *o.Description
}

// GetDescriptionOk returns a tuple with the Description field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *OrganizationAttributes) GetDescriptionOk() (*string, bool) {
	if o == nil || o.Description == nil {
		return nil, false
	}
	return o.Description, true
}

// HasDescription returns a boolean if a field has been set.
func (o *OrganizationAttributes) HasDescription() bool {
	return o != nil && o.Description != nil
}

// SetDescription gets a reference to the given string and assigns it to the Description field.
func (o *OrganizationAttributes) SetDescription(v string) {
	o.Description = &v
}

// GetDisabled returns the Disabled field value if set, zero value otherwise.
func (o *OrganizationAttributes) GetDisabled() bool {
	if o == nil || o.Disabled == nil {
		var ret bool
		return ret
	}
	return *o.Disabled
}

// GetDisabledOk returns a tuple with the Disabled field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *OrganizationAttributes) GetDisabledOk() (*bool, bool) {
	if o == nil || o.Disabled == nil {
		return nil, false
	}
	return o.Disabled, true
}

// HasDisabled returns a boolean if a field has been set.
func (o *OrganizationAttributes) HasDisabled() bool {
	return o != nil && o.Disabled != nil
}

// SetDisabled gets a reference to the given bool and assigns it to the Disabled field.
func (o *OrganizationAttributes) SetDisabled(v bool) {
	o.Disabled = &v
}

// GetModifiedAt returns the ModifiedAt field value if set, zero value otherwise.
func (o *OrganizationAttributes) GetModifiedAt() time.Time {
	if o == nil || o.ModifiedAt == nil {
		var ret time.Time
		return ret
	}
	return *o.ModifiedAt
}

// GetModifiedAtOk returns a tuple with the ModifiedAt field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *OrganizationAttributes) GetModifiedAtOk() (*time.Time, bool) {
	if o == nil || o.ModifiedAt == nil {
		return nil, false
	}
	return o.ModifiedAt, true
}

// HasModifiedAt returns a boolean if a field has been set.
func (o *OrganizationAttributes) HasModifiedAt() bool {
	return o != nil && o.ModifiedAt != nil
}

// SetModifiedAt gets a reference to the given time.Time and assigns it to the ModifiedAt field.
func (o *OrganizationAttributes) SetModifiedAt(v time.Time) {
	o.ModifiedAt = &v
}

// GetName returns the Name field value if set, zero value otherwise.
func (o *OrganizationAttributes) GetName() string {
	if o == nil || o.Name == nil {
		var ret string
		return ret
	}
	return *o.Name
}

// GetNameOk returns a tuple with the Name field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *OrganizationAttributes) GetNameOk() (*string, bool) {
	if o == nil || o.Name == nil {
		return nil, false
	}
	return o.Name, true
}

// HasName returns a boolean if a field has been set.
func (o *OrganizationAttributes) HasName() bool {
	return o != nil && o.Name != nil
}

// SetName gets a reference to the given string and assigns it to the Name field.
func (o *OrganizationAttributes) SetName(v string) {
	o.Name = &v
}

// GetPublicId returns the PublicId field value if set, zero value otherwise.
func (o *OrganizationAttributes) GetPublicId() string {
	if o == nil || o.PublicId == nil {
		var ret string
		return ret
	}
	return *o.PublicId
}

// GetPublicIdOk returns a tuple with the PublicId field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *OrganizationAttributes) GetPublicIdOk() (*string, bool) {
	if o == nil || o.PublicId == nil {
		return nil, false
	}
	return o.PublicId, true
}

// HasPublicId returns a boolean if a field has been set.
func (o *OrganizationAttributes) HasPublicId() bool {
	return o != nil && o.PublicId != nil
}

// SetPublicId gets a reference to the given string and assigns it to the PublicId field.
func (o *OrganizationAttributes) SetPublicId(v string) {
	o.PublicId = &v
}

// GetSharing returns the Sharing field value if set, zero value otherwise.
func (o *OrganizationAttributes) GetSharing() string {
	if o == nil || o.Sharing == nil {
		var ret string
		return ret
	}
	return *o.Sharing
}

// GetSharingOk returns a tuple with the Sharing field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *OrganizationAttributes) GetSharingOk() (*string, bool) {
	if o == nil || o.Sharing == nil {
		return nil, false
	}
	return o.Sharing, true
}

// HasSharing returns a boolean if a field has been set.
func (o *OrganizationAttributes) HasSharing() bool {
	return o != nil && o.Sharing != nil
}

// SetSharing gets a reference to the given string and assigns it to the Sharing field.
func (o *OrganizationAttributes) SetSharing(v string) {
	o.Sharing = &v
}

// GetUrl returns the Url field value if set, zero value otherwise.
func (o *OrganizationAttributes) GetUrl() string {
	if o == nil || o.Url == nil {
		var ret string
		return ret
	}
	return *o.Url
}

// GetUrlOk returns a tuple with the Url field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *OrganizationAttributes) GetUrlOk() (*string, bool) {
	if o == nil || o.Url == nil {
		return nil, false
	}
	return o.Url, true
}

// HasUrl returns a boolean if a field has been set.
func (o *OrganizationAttributes) HasUrl() bool {
	return o != nil && o.Url != nil
}

// SetUrl gets a reference to the given string and assigns it to the Url field.
func (o *OrganizationAttributes) SetUrl(v string) {
	o.Url = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o OrganizationAttributes) MarshalJSON() ([]byte, error) {
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
	if o.Description != nil {
		toSerialize["description"] = o.Description
	}
	if o.Disabled != nil {
		toSerialize["disabled"] = o.Disabled
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
	if o.PublicId != nil {
		toSerialize["public_id"] = o.PublicId
	}
	if o.Sharing != nil {
		toSerialize["sharing"] = o.Sharing
	}
	if o.Url != nil {
		toSerialize["url"] = o.Url
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *OrganizationAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		CreatedAt   *time.Time `json:"created_at,omitempty"`
		Description *string    `json:"description,omitempty"`
		Disabled    *bool      `json:"disabled,omitempty"`
		ModifiedAt  *time.Time `json:"modified_at,omitempty"`
		Name        *string    `json:"name,omitempty"`
		PublicId    *string    `json:"public_id,omitempty"`
		Sharing     *string    `json:"sharing,omitempty"`
		Url         *string    `json:"url,omitempty"`
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
	o.Description = all.Description
	o.Disabled = all.Disabled
	o.ModifiedAt = all.ModifiedAt
	o.Name = all.Name
	o.PublicId = all.PublicId
	o.Sharing = all.Sharing
	o.Url = all.Url
	return nil
}
