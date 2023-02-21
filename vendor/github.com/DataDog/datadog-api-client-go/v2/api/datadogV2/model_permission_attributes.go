// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"time"
)

// PermissionAttributes Attributes of a permission.
type PermissionAttributes struct {
	// Creation time of the permission.
	Created *time.Time `json:"created,omitempty"`
	// Description of the permission.
	Description *string `json:"description,omitempty"`
	// Displayed name for the permission.
	DisplayName *string `json:"display_name,omitempty"`
	// Display type.
	DisplayType *string `json:"display_type,omitempty"`
	// Name of the permission group.
	GroupName *string `json:"group_name,omitempty"`
	// Name of the permission.
	Name *string `json:"name,omitempty"`
	// Whether or not the permission is restricted.
	Restricted *bool `json:"restricted,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewPermissionAttributes instantiates a new PermissionAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewPermissionAttributes() *PermissionAttributes {
	this := PermissionAttributes{}
	return &this
}

// NewPermissionAttributesWithDefaults instantiates a new PermissionAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewPermissionAttributesWithDefaults() *PermissionAttributes {
	this := PermissionAttributes{}
	return &this
}

// GetCreated returns the Created field value if set, zero value otherwise.
func (o *PermissionAttributes) GetCreated() time.Time {
	if o == nil || o.Created == nil {
		var ret time.Time
		return ret
	}
	return *o.Created
}

// GetCreatedOk returns a tuple with the Created field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *PermissionAttributes) GetCreatedOk() (*time.Time, bool) {
	if o == nil || o.Created == nil {
		return nil, false
	}
	return o.Created, true
}

// HasCreated returns a boolean if a field has been set.
func (o *PermissionAttributes) HasCreated() bool {
	return o != nil && o.Created != nil
}

// SetCreated gets a reference to the given time.Time and assigns it to the Created field.
func (o *PermissionAttributes) SetCreated(v time.Time) {
	o.Created = &v
}

// GetDescription returns the Description field value if set, zero value otherwise.
func (o *PermissionAttributes) GetDescription() string {
	if o == nil || o.Description == nil {
		var ret string
		return ret
	}
	return *o.Description
}

// GetDescriptionOk returns a tuple with the Description field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *PermissionAttributes) GetDescriptionOk() (*string, bool) {
	if o == nil || o.Description == nil {
		return nil, false
	}
	return o.Description, true
}

// HasDescription returns a boolean if a field has been set.
func (o *PermissionAttributes) HasDescription() bool {
	return o != nil && o.Description != nil
}

// SetDescription gets a reference to the given string and assigns it to the Description field.
func (o *PermissionAttributes) SetDescription(v string) {
	o.Description = &v
}

// GetDisplayName returns the DisplayName field value if set, zero value otherwise.
func (o *PermissionAttributes) GetDisplayName() string {
	if o == nil || o.DisplayName == nil {
		var ret string
		return ret
	}
	return *o.DisplayName
}

// GetDisplayNameOk returns a tuple with the DisplayName field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *PermissionAttributes) GetDisplayNameOk() (*string, bool) {
	if o == nil || o.DisplayName == nil {
		return nil, false
	}
	return o.DisplayName, true
}

// HasDisplayName returns a boolean if a field has been set.
func (o *PermissionAttributes) HasDisplayName() bool {
	return o != nil && o.DisplayName != nil
}

// SetDisplayName gets a reference to the given string and assigns it to the DisplayName field.
func (o *PermissionAttributes) SetDisplayName(v string) {
	o.DisplayName = &v
}

// GetDisplayType returns the DisplayType field value if set, zero value otherwise.
func (o *PermissionAttributes) GetDisplayType() string {
	if o == nil || o.DisplayType == nil {
		var ret string
		return ret
	}
	return *o.DisplayType
}

// GetDisplayTypeOk returns a tuple with the DisplayType field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *PermissionAttributes) GetDisplayTypeOk() (*string, bool) {
	if o == nil || o.DisplayType == nil {
		return nil, false
	}
	return o.DisplayType, true
}

// HasDisplayType returns a boolean if a field has been set.
func (o *PermissionAttributes) HasDisplayType() bool {
	return o != nil && o.DisplayType != nil
}

// SetDisplayType gets a reference to the given string and assigns it to the DisplayType field.
func (o *PermissionAttributes) SetDisplayType(v string) {
	o.DisplayType = &v
}

// GetGroupName returns the GroupName field value if set, zero value otherwise.
func (o *PermissionAttributes) GetGroupName() string {
	if o == nil || o.GroupName == nil {
		var ret string
		return ret
	}
	return *o.GroupName
}

// GetGroupNameOk returns a tuple with the GroupName field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *PermissionAttributes) GetGroupNameOk() (*string, bool) {
	if o == nil || o.GroupName == nil {
		return nil, false
	}
	return o.GroupName, true
}

// HasGroupName returns a boolean if a field has been set.
func (o *PermissionAttributes) HasGroupName() bool {
	return o != nil && o.GroupName != nil
}

// SetGroupName gets a reference to the given string and assigns it to the GroupName field.
func (o *PermissionAttributes) SetGroupName(v string) {
	o.GroupName = &v
}

// GetName returns the Name field value if set, zero value otherwise.
func (o *PermissionAttributes) GetName() string {
	if o == nil || o.Name == nil {
		var ret string
		return ret
	}
	return *o.Name
}

// GetNameOk returns a tuple with the Name field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *PermissionAttributes) GetNameOk() (*string, bool) {
	if o == nil || o.Name == nil {
		return nil, false
	}
	return o.Name, true
}

// HasName returns a boolean if a field has been set.
func (o *PermissionAttributes) HasName() bool {
	return o != nil && o.Name != nil
}

// SetName gets a reference to the given string and assigns it to the Name field.
func (o *PermissionAttributes) SetName(v string) {
	o.Name = &v
}

// GetRestricted returns the Restricted field value if set, zero value otherwise.
func (o *PermissionAttributes) GetRestricted() bool {
	if o == nil || o.Restricted == nil {
		var ret bool
		return ret
	}
	return *o.Restricted
}

// GetRestrictedOk returns a tuple with the Restricted field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *PermissionAttributes) GetRestrictedOk() (*bool, bool) {
	if o == nil || o.Restricted == nil {
		return nil, false
	}
	return o.Restricted, true
}

// HasRestricted returns a boolean if a field has been set.
func (o *PermissionAttributes) HasRestricted() bool {
	return o != nil && o.Restricted != nil
}

// SetRestricted gets a reference to the given bool and assigns it to the Restricted field.
func (o *PermissionAttributes) SetRestricted(v bool) {
	o.Restricted = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o PermissionAttributes) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Created != nil {
		if o.Created.Nanosecond() == 0 {
			toSerialize["created"] = o.Created.Format("2006-01-02T15:04:05Z07:00")
		} else {
			toSerialize["created"] = o.Created.Format("2006-01-02T15:04:05.000Z07:00")
		}
	}
	if o.Description != nil {
		toSerialize["description"] = o.Description
	}
	if o.DisplayName != nil {
		toSerialize["display_name"] = o.DisplayName
	}
	if o.DisplayType != nil {
		toSerialize["display_type"] = o.DisplayType
	}
	if o.GroupName != nil {
		toSerialize["group_name"] = o.GroupName
	}
	if o.Name != nil {
		toSerialize["name"] = o.Name
	}
	if o.Restricted != nil {
		toSerialize["restricted"] = o.Restricted
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *PermissionAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Created     *time.Time `json:"created,omitempty"`
		Description *string    `json:"description,omitempty"`
		DisplayName *string    `json:"display_name,omitempty"`
		DisplayType *string    `json:"display_type,omitempty"`
		GroupName   *string    `json:"group_name,omitempty"`
		Name        *string    `json:"name,omitempty"`
		Restricted  *bool      `json:"restricted,omitempty"`
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
	o.Created = all.Created
	o.Description = all.Description
	o.DisplayName = all.DisplayName
	o.DisplayType = all.DisplayType
	o.GroupName = all.GroupName
	o.Name = all.Name
	o.Restricted = all.Restricted
	return nil
}
