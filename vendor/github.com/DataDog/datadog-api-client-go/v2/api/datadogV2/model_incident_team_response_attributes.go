// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"time"
)

// IncidentTeamResponseAttributes The incident team's attributes from a response.
type IncidentTeamResponseAttributes struct {
	// Timestamp of when the incident team was created.
	Created *time.Time `json:"created,omitempty"`
	// Timestamp of when the incident team was modified.
	Modified *time.Time `json:"modified,omitempty"`
	// Name of the incident team.
	Name *string `json:"name,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewIncidentTeamResponseAttributes instantiates a new IncidentTeamResponseAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewIncidentTeamResponseAttributes() *IncidentTeamResponseAttributes {
	this := IncidentTeamResponseAttributes{}
	return &this
}

// NewIncidentTeamResponseAttributesWithDefaults instantiates a new IncidentTeamResponseAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewIncidentTeamResponseAttributesWithDefaults() *IncidentTeamResponseAttributes {
	this := IncidentTeamResponseAttributes{}
	return &this
}

// GetCreated returns the Created field value if set, zero value otherwise.
func (o *IncidentTeamResponseAttributes) GetCreated() time.Time {
	if o == nil || o.Created == nil {
		var ret time.Time
		return ret
	}
	return *o.Created
}

// GetCreatedOk returns a tuple with the Created field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IncidentTeamResponseAttributes) GetCreatedOk() (*time.Time, bool) {
	if o == nil || o.Created == nil {
		return nil, false
	}
	return o.Created, true
}

// HasCreated returns a boolean if a field has been set.
func (o *IncidentTeamResponseAttributes) HasCreated() bool {
	return o != nil && o.Created != nil
}

// SetCreated gets a reference to the given time.Time and assigns it to the Created field.
func (o *IncidentTeamResponseAttributes) SetCreated(v time.Time) {
	o.Created = &v
}

// GetModified returns the Modified field value if set, zero value otherwise.
func (o *IncidentTeamResponseAttributes) GetModified() time.Time {
	if o == nil || o.Modified == nil {
		var ret time.Time
		return ret
	}
	return *o.Modified
}

// GetModifiedOk returns a tuple with the Modified field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IncidentTeamResponseAttributes) GetModifiedOk() (*time.Time, bool) {
	if o == nil || o.Modified == nil {
		return nil, false
	}
	return o.Modified, true
}

// HasModified returns a boolean if a field has been set.
func (o *IncidentTeamResponseAttributes) HasModified() bool {
	return o != nil && o.Modified != nil
}

// SetModified gets a reference to the given time.Time and assigns it to the Modified field.
func (o *IncidentTeamResponseAttributes) SetModified(v time.Time) {
	o.Modified = &v
}

// GetName returns the Name field value if set, zero value otherwise.
func (o *IncidentTeamResponseAttributes) GetName() string {
	if o == nil || o.Name == nil {
		var ret string
		return ret
	}
	return *o.Name
}

// GetNameOk returns a tuple with the Name field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IncidentTeamResponseAttributes) GetNameOk() (*string, bool) {
	if o == nil || o.Name == nil {
		return nil, false
	}
	return o.Name, true
}

// HasName returns a boolean if a field has been set.
func (o *IncidentTeamResponseAttributes) HasName() bool {
	return o != nil && o.Name != nil
}

// SetName gets a reference to the given string and assigns it to the Name field.
func (o *IncidentTeamResponseAttributes) SetName(v string) {
	o.Name = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o IncidentTeamResponseAttributes) MarshalJSON() ([]byte, error) {
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
	if o.Modified != nil {
		if o.Modified.Nanosecond() == 0 {
			toSerialize["modified"] = o.Modified.Format("2006-01-02T15:04:05Z07:00")
		} else {
			toSerialize["modified"] = o.Modified.Format("2006-01-02T15:04:05.000Z07:00")
		}
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
func (o *IncidentTeamResponseAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Created  *time.Time `json:"created,omitempty"`
		Modified *time.Time `json:"modified,omitempty"`
		Name     *string    `json:"name,omitempty"`
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
	o.Modified = all.Modified
	o.Name = all.Name
	return nil
}
