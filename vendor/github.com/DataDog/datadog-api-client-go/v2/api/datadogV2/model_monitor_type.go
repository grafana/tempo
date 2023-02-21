// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// MonitorType Attributes from the monitor that triggered the event.
type MonitorType struct {
	// The POSIX timestamp of the monitor's creation in nanoseconds.
	CreatedAt *int64 `json:"created_at,omitempty"`
	// Monitor group status used when there is no `result_groups`.
	GroupStatus *int32 `json:"group_status,omitempty"`
	// Groups to which the monitor belongs.
	Groups []string `json:"groups,omitempty"`
	// The monitor ID.
	Id *int64 `json:"id,omitempty"`
	// The monitor message.
	Message *string `json:"message,omitempty"`
	// The monitor's last-modified timestamp.
	Modified *int64 `json:"modified,omitempty"`
	// The monitor name.
	Name *string `json:"name,omitempty"`
	// The query that triggers the alert.
	Query *string `json:"query,omitempty"`
	// A list of tags attached to the monitor.
	Tags []string `json:"tags,omitempty"`
	// The templated name of the monitor before resolving any template variables.
	TemplatedName *string `json:"templated_name,omitempty"`
	// The monitor type.
	Type *string `json:"type,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewMonitorType instantiates a new MonitorType object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewMonitorType() *MonitorType {
	this := MonitorType{}
	return &this
}

// NewMonitorTypeWithDefaults instantiates a new MonitorType object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewMonitorTypeWithDefaults() *MonitorType {
	this := MonitorType{}
	return &this
}

// GetCreatedAt returns the CreatedAt field value if set, zero value otherwise.
func (o *MonitorType) GetCreatedAt() int64 {
	if o == nil || o.CreatedAt == nil {
		var ret int64
		return ret
	}
	return *o.CreatedAt
}

// GetCreatedAtOk returns a tuple with the CreatedAt field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorType) GetCreatedAtOk() (*int64, bool) {
	if o == nil || o.CreatedAt == nil {
		return nil, false
	}
	return o.CreatedAt, true
}

// HasCreatedAt returns a boolean if a field has been set.
func (o *MonitorType) HasCreatedAt() bool {
	return o != nil && o.CreatedAt != nil
}

// SetCreatedAt gets a reference to the given int64 and assigns it to the CreatedAt field.
func (o *MonitorType) SetCreatedAt(v int64) {
	o.CreatedAt = &v
}

// GetGroupStatus returns the GroupStatus field value if set, zero value otherwise.
func (o *MonitorType) GetGroupStatus() int32 {
	if o == nil || o.GroupStatus == nil {
		var ret int32
		return ret
	}
	return *o.GroupStatus
}

// GetGroupStatusOk returns a tuple with the GroupStatus field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorType) GetGroupStatusOk() (*int32, bool) {
	if o == nil || o.GroupStatus == nil {
		return nil, false
	}
	return o.GroupStatus, true
}

// HasGroupStatus returns a boolean if a field has been set.
func (o *MonitorType) HasGroupStatus() bool {
	return o != nil && o.GroupStatus != nil
}

// SetGroupStatus gets a reference to the given int32 and assigns it to the GroupStatus field.
func (o *MonitorType) SetGroupStatus(v int32) {
	o.GroupStatus = &v
}

// GetGroups returns the Groups field value if set, zero value otherwise.
func (o *MonitorType) GetGroups() []string {
	if o == nil || o.Groups == nil {
		var ret []string
		return ret
	}
	return o.Groups
}

// GetGroupsOk returns a tuple with the Groups field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorType) GetGroupsOk() (*[]string, bool) {
	if o == nil || o.Groups == nil {
		return nil, false
	}
	return &o.Groups, true
}

// HasGroups returns a boolean if a field has been set.
func (o *MonitorType) HasGroups() bool {
	return o != nil && o.Groups != nil
}

// SetGroups gets a reference to the given []string and assigns it to the Groups field.
func (o *MonitorType) SetGroups(v []string) {
	o.Groups = v
}

// GetId returns the Id field value if set, zero value otherwise.
func (o *MonitorType) GetId() int64 {
	if o == nil || o.Id == nil {
		var ret int64
		return ret
	}
	return *o.Id
}

// GetIdOk returns a tuple with the Id field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorType) GetIdOk() (*int64, bool) {
	if o == nil || o.Id == nil {
		return nil, false
	}
	return o.Id, true
}

// HasId returns a boolean if a field has been set.
func (o *MonitorType) HasId() bool {
	return o != nil && o.Id != nil
}

// SetId gets a reference to the given int64 and assigns it to the Id field.
func (o *MonitorType) SetId(v int64) {
	o.Id = &v
}

// GetMessage returns the Message field value if set, zero value otherwise.
func (o *MonitorType) GetMessage() string {
	if o == nil || o.Message == nil {
		var ret string
		return ret
	}
	return *o.Message
}

// GetMessageOk returns a tuple with the Message field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorType) GetMessageOk() (*string, bool) {
	if o == nil || o.Message == nil {
		return nil, false
	}
	return o.Message, true
}

// HasMessage returns a boolean if a field has been set.
func (o *MonitorType) HasMessage() bool {
	return o != nil && o.Message != nil
}

// SetMessage gets a reference to the given string and assigns it to the Message field.
func (o *MonitorType) SetMessage(v string) {
	o.Message = &v
}

// GetModified returns the Modified field value if set, zero value otherwise.
func (o *MonitorType) GetModified() int64 {
	if o == nil || o.Modified == nil {
		var ret int64
		return ret
	}
	return *o.Modified
}

// GetModifiedOk returns a tuple with the Modified field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorType) GetModifiedOk() (*int64, bool) {
	if o == nil || o.Modified == nil {
		return nil, false
	}
	return o.Modified, true
}

// HasModified returns a boolean if a field has been set.
func (o *MonitorType) HasModified() bool {
	return o != nil && o.Modified != nil
}

// SetModified gets a reference to the given int64 and assigns it to the Modified field.
func (o *MonitorType) SetModified(v int64) {
	o.Modified = &v
}

// GetName returns the Name field value if set, zero value otherwise.
func (o *MonitorType) GetName() string {
	if o == nil || o.Name == nil {
		var ret string
		return ret
	}
	return *o.Name
}

// GetNameOk returns a tuple with the Name field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorType) GetNameOk() (*string, bool) {
	if o == nil || o.Name == nil {
		return nil, false
	}
	return o.Name, true
}

// HasName returns a boolean if a field has been set.
func (o *MonitorType) HasName() bool {
	return o != nil && o.Name != nil
}

// SetName gets a reference to the given string and assigns it to the Name field.
func (o *MonitorType) SetName(v string) {
	o.Name = &v
}

// GetQuery returns the Query field value if set, zero value otherwise.
func (o *MonitorType) GetQuery() string {
	if o == nil || o.Query == nil {
		var ret string
		return ret
	}
	return *o.Query
}

// GetQueryOk returns a tuple with the Query field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorType) GetQueryOk() (*string, bool) {
	if o == nil || o.Query == nil {
		return nil, false
	}
	return o.Query, true
}

// HasQuery returns a boolean if a field has been set.
func (o *MonitorType) HasQuery() bool {
	return o != nil && o.Query != nil
}

// SetQuery gets a reference to the given string and assigns it to the Query field.
func (o *MonitorType) SetQuery(v string) {
	o.Query = &v
}

// GetTags returns the Tags field value if set, zero value otherwise.
func (o *MonitorType) GetTags() []string {
	if o == nil || o.Tags == nil {
		var ret []string
		return ret
	}
	return o.Tags
}

// GetTagsOk returns a tuple with the Tags field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorType) GetTagsOk() (*[]string, bool) {
	if o == nil || o.Tags == nil {
		return nil, false
	}
	return &o.Tags, true
}

// HasTags returns a boolean if a field has been set.
func (o *MonitorType) HasTags() bool {
	return o != nil && o.Tags != nil
}

// SetTags gets a reference to the given []string and assigns it to the Tags field.
func (o *MonitorType) SetTags(v []string) {
	o.Tags = v
}

// GetTemplatedName returns the TemplatedName field value if set, zero value otherwise.
func (o *MonitorType) GetTemplatedName() string {
	if o == nil || o.TemplatedName == nil {
		var ret string
		return ret
	}
	return *o.TemplatedName
}

// GetTemplatedNameOk returns a tuple with the TemplatedName field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorType) GetTemplatedNameOk() (*string, bool) {
	if o == nil || o.TemplatedName == nil {
		return nil, false
	}
	return o.TemplatedName, true
}

// HasTemplatedName returns a boolean if a field has been set.
func (o *MonitorType) HasTemplatedName() bool {
	return o != nil && o.TemplatedName != nil
}

// SetTemplatedName gets a reference to the given string and assigns it to the TemplatedName field.
func (o *MonitorType) SetTemplatedName(v string) {
	o.TemplatedName = &v
}

// GetType returns the Type field value if set, zero value otherwise.
func (o *MonitorType) GetType() string {
	if o == nil || o.Type == nil {
		var ret string
		return ret
	}
	return *o.Type
}

// GetTypeOk returns a tuple with the Type field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorType) GetTypeOk() (*string, bool) {
	if o == nil || o.Type == nil {
		return nil, false
	}
	return o.Type, true
}

// HasType returns a boolean if a field has been set.
func (o *MonitorType) HasType() bool {
	return o != nil && o.Type != nil
}

// SetType gets a reference to the given string and assigns it to the Type field.
func (o *MonitorType) SetType(v string) {
	o.Type = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o MonitorType) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.CreatedAt != nil {
		toSerialize["created_at"] = o.CreatedAt
	}
	if o.GroupStatus != nil {
		toSerialize["group_status"] = o.GroupStatus
	}
	if o.Groups != nil {
		toSerialize["groups"] = o.Groups
	}
	if o.Id != nil {
		toSerialize["id"] = o.Id
	}
	if o.Message != nil {
		toSerialize["message"] = o.Message
	}
	if o.Modified != nil {
		toSerialize["modified"] = o.Modified
	}
	if o.Name != nil {
		toSerialize["name"] = o.Name
	}
	if o.Query != nil {
		toSerialize["query"] = o.Query
	}
	if o.Tags != nil {
		toSerialize["tags"] = o.Tags
	}
	if o.TemplatedName != nil {
		toSerialize["templated_name"] = o.TemplatedName
	}
	if o.Type != nil {
		toSerialize["type"] = o.Type
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *MonitorType) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		CreatedAt     *int64   `json:"created_at,omitempty"`
		GroupStatus   *int32   `json:"group_status,omitempty"`
		Groups        []string `json:"groups,omitempty"`
		Id            *int64   `json:"id,omitempty"`
		Message       *string  `json:"message,omitempty"`
		Modified      *int64   `json:"modified,omitempty"`
		Name          *string  `json:"name,omitempty"`
		Query         *string  `json:"query,omitempty"`
		Tags          []string `json:"tags,omitempty"`
		TemplatedName *string  `json:"templated_name,omitempty"`
		Type          *string  `json:"type,omitempty"`
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
	o.GroupStatus = all.GroupStatus
	o.Groups = all.Groups
	o.Id = all.Id
	o.Message = all.Message
	o.Modified = all.Modified
	o.Name = all.Name
	o.Query = all.Query
	o.Tags = all.Tags
	o.TemplatedName = all.TemplatedName
	o.Type = all.Type
	return nil
}

// NullableMonitorType handles when a null is used for MonitorType.
type NullableMonitorType struct {
	value *MonitorType
	isSet bool
}

// Get returns the associated value.
func (v NullableMonitorType) Get() *MonitorType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableMonitorType) Set(val *MonitorType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableMonitorType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag/
func (v *NullableMonitorType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableMonitorType initializes the struct as if Set has been called.
func NewNullableMonitorType(val *MonitorType) *NullableMonitorType {
	return &NullableMonitorType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableMonitorType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableMonitorType) UnmarshalJSON(src []byte) error {
	v.isSet = true

	// this object is nullable so check if the payload is null or empty string
	if string(src) == "" || string(src) == "{}" {
		return nil
	}

	return json.Unmarshal(src, &v.value)
}
