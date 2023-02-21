// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// IncidentSearchResponseUserFacetData Facet data for user attributes of an incident.
type IncidentSearchResponseUserFacetData struct {
	// Count of the facet value appearing in search results.
	Count *int32 `json:"count,omitempty"`
	// Email of the user.
	Email *string `json:"email,omitempty"`
	// Handle of the user.
	Handle *string `json:"handle,omitempty"`
	// Name of the user.
	Name *string `json:"name,omitempty"`
	// ID of the user.
	Uuid *string `json:"uuid,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewIncidentSearchResponseUserFacetData instantiates a new IncidentSearchResponseUserFacetData object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewIncidentSearchResponseUserFacetData() *IncidentSearchResponseUserFacetData {
	this := IncidentSearchResponseUserFacetData{}
	return &this
}

// NewIncidentSearchResponseUserFacetDataWithDefaults instantiates a new IncidentSearchResponseUserFacetData object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewIncidentSearchResponseUserFacetDataWithDefaults() *IncidentSearchResponseUserFacetData {
	this := IncidentSearchResponseUserFacetData{}
	return &this
}

// GetCount returns the Count field value if set, zero value otherwise.
func (o *IncidentSearchResponseUserFacetData) GetCount() int32 {
	if o == nil || o.Count == nil {
		var ret int32
		return ret
	}
	return *o.Count
}

// GetCountOk returns a tuple with the Count field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IncidentSearchResponseUserFacetData) GetCountOk() (*int32, bool) {
	if o == nil || o.Count == nil {
		return nil, false
	}
	return o.Count, true
}

// HasCount returns a boolean if a field has been set.
func (o *IncidentSearchResponseUserFacetData) HasCount() bool {
	return o != nil && o.Count != nil
}

// SetCount gets a reference to the given int32 and assigns it to the Count field.
func (o *IncidentSearchResponseUserFacetData) SetCount(v int32) {
	o.Count = &v
}

// GetEmail returns the Email field value if set, zero value otherwise.
func (o *IncidentSearchResponseUserFacetData) GetEmail() string {
	if o == nil || o.Email == nil {
		var ret string
		return ret
	}
	return *o.Email
}

// GetEmailOk returns a tuple with the Email field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IncidentSearchResponseUserFacetData) GetEmailOk() (*string, bool) {
	if o == nil || o.Email == nil {
		return nil, false
	}
	return o.Email, true
}

// HasEmail returns a boolean if a field has been set.
func (o *IncidentSearchResponseUserFacetData) HasEmail() bool {
	return o != nil && o.Email != nil
}

// SetEmail gets a reference to the given string and assigns it to the Email field.
func (o *IncidentSearchResponseUserFacetData) SetEmail(v string) {
	o.Email = &v
}

// GetHandle returns the Handle field value if set, zero value otherwise.
func (o *IncidentSearchResponseUserFacetData) GetHandle() string {
	if o == nil || o.Handle == nil {
		var ret string
		return ret
	}
	return *o.Handle
}

// GetHandleOk returns a tuple with the Handle field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IncidentSearchResponseUserFacetData) GetHandleOk() (*string, bool) {
	if o == nil || o.Handle == nil {
		return nil, false
	}
	return o.Handle, true
}

// HasHandle returns a boolean if a field has been set.
func (o *IncidentSearchResponseUserFacetData) HasHandle() bool {
	return o != nil && o.Handle != nil
}

// SetHandle gets a reference to the given string and assigns it to the Handle field.
func (o *IncidentSearchResponseUserFacetData) SetHandle(v string) {
	o.Handle = &v
}

// GetName returns the Name field value if set, zero value otherwise.
func (o *IncidentSearchResponseUserFacetData) GetName() string {
	if o == nil || o.Name == nil {
		var ret string
		return ret
	}
	return *o.Name
}

// GetNameOk returns a tuple with the Name field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IncidentSearchResponseUserFacetData) GetNameOk() (*string, bool) {
	if o == nil || o.Name == nil {
		return nil, false
	}
	return o.Name, true
}

// HasName returns a boolean if a field has been set.
func (o *IncidentSearchResponseUserFacetData) HasName() bool {
	return o != nil && o.Name != nil
}

// SetName gets a reference to the given string and assigns it to the Name field.
func (o *IncidentSearchResponseUserFacetData) SetName(v string) {
	o.Name = &v
}

// GetUuid returns the Uuid field value if set, zero value otherwise.
func (o *IncidentSearchResponseUserFacetData) GetUuid() string {
	if o == nil || o.Uuid == nil {
		var ret string
		return ret
	}
	return *o.Uuid
}

// GetUuidOk returns a tuple with the Uuid field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IncidentSearchResponseUserFacetData) GetUuidOk() (*string, bool) {
	if o == nil || o.Uuid == nil {
		return nil, false
	}
	return o.Uuid, true
}

// HasUuid returns a boolean if a field has been set.
func (o *IncidentSearchResponseUserFacetData) HasUuid() bool {
	return o != nil && o.Uuid != nil
}

// SetUuid gets a reference to the given string and assigns it to the Uuid field.
func (o *IncidentSearchResponseUserFacetData) SetUuid(v string) {
	o.Uuid = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o IncidentSearchResponseUserFacetData) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Count != nil {
		toSerialize["count"] = o.Count
	}
	if o.Email != nil {
		toSerialize["email"] = o.Email
	}
	if o.Handle != nil {
		toSerialize["handle"] = o.Handle
	}
	if o.Name != nil {
		toSerialize["name"] = o.Name
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
func (o *IncidentSearchResponseUserFacetData) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Count  *int32  `json:"count,omitempty"`
		Email  *string `json:"email,omitempty"`
		Handle *string `json:"handle,omitempty"`
		Name   *string `json:"name,omitempty"`
		Uuid   *string `json:"uuid,omitempty"`
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
	o.Count = all.Count
	o.Email = all.Email
	o.Handle = all.Handle
	o.Name = all.Name
	o.Uuid = all.Uuid
	return nil
}
