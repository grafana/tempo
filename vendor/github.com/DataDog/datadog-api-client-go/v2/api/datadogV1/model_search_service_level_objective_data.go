// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV1

import (
	"encoding/json"
)

// SearchServiceLevelObjectiveData A service level objective ID and attributes.
type SearchServiceLevelObjectiveData struct {
	// A service level objective object includes a service level indicator, thresholds
	// for one or more timeframes, and metadata (`name`, `description`, and `tags`).
	Attributes *SearchServiceLevelObjectiveAttributes `json:"attributes,omitempty"`
	// A unique identifier for the service level objective object.
	//
	// Always included in service level objective responses.
	Id *string `json:"id,omitempty"`
	// The type of the object, must be `slo`.
	Type *string `json:"type,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewSearchServiceLevelObjectiveData instantiates a new SearchServiceLevelObjectiveData object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSearchServiceLevelObjectiveData() *SearchServiceLevelObjectiveData {
	this := SearchServiceLevelObjectiveData{}
	return &this
}

// NewSearchServiceLevelObjectiveDataWithDefaults instantiates a new SearchServiceLevelObjectiveData object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSearchServiceLevelObjectiveDataWithDefaults() *SearchServiceLevelObjectiveData {
	this := SearchServiceLevelObjectiveData{}
	return &this
}

// GetAttributes returns the Attributes field value if set, zero value otherwise.
func (o *SearchServiceLevelObjectiveData) GetAttributes() SearchServiceLevelObjectiveAttributes {
	if o == nil || o.Attributes == nil {
		var ret SearchServiceLevelObjectiveAttributes
		return ret
	}
	return *o.Attributes
}

// GetAttributesOk returns a tuple with the Attributes field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SearchServiceLevelObjectiveData) GetAttributesOk() (*SearchServiceLevelObjectiveAttributes, bool) {
	if o == nil || o.Attributes == nil {
		return nil, false
	}
	return o.Attributes, true
}

// HasAttributes returns a boolean if a field has been set.
func (o *SearchServiceLevelObjectiveData) HasAttributes() bool {
	return o != nil && o.Attributes != nil
}

// SetAttributes gets a reference to the given SearchServiceLevelObjectiveAttributes and assigns it to the Attributes field.
func (o *SearchServiceLevelObjectiveData) SetAttributes(v SearchServiceLevelObjectiveAttributes) {
	o.Attributes = &v
}

// GetId returns the Id field value if set, zero value otherwise.
func (o *SearchServiceLevelObjectiveData) GetId() string {
	if o == nil || o.Id == nil {
		var ret string
		return ret
	}
	return *o.Id
}

// GetIdOk returns a tuple with the Id field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SearchServiceLevelObjectiveData) GetIdOk() (*string, bool) {
	if o == nil || o.Id == nil {
		return nil, false
	}
	return o.Id, true
}

// HasId returns a boolean if a field has been set.
func (o *SearchServiceLevelObjectiveData) HasId() bool {
	return o != nil && o.Id != nil
}

// SetId gets a reference to the given string and assigns it to the Id field.
func (o *SearchServiceLevelObjectiveData) SetId(v string) {
	o.Id = &v
}

// GetType returns the Type field value if set, zero value otherwise.
func (o *SearchServiceLevelObjectiveData) GetType() string {
	if o == nil || o.Type == nil {
		var ret string
		return ret
	}
	return *o.Type
}

// GetTypeOk returns a tuple with the Type field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SearchServiceLevelObjectiveData) GetTypeOk() (*string, bool) {
	if o == nil || o.Type == nil {
		return nil, false
	}
	return o.Type, true
}

// HasType returns a boolean if a field has been set.
func (o *SearchServiceLevelObjectiveData) HasType() bool {
	return o != nil && o.Type != nil
}

// SetType gets a reference to the given string and assigns it to the Type field.
func (o *SearchServiceLevelObjectiveData) SetType(v string) {
	o.Type = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SearchServiceLevelObjectiveData) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Attributes != nil {
		toSerialize["attributes"] = o.Attributes
	}
	if o.Id != nil {
		toSerialize["id"] = o.Id
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
func (o *SearchServiceLevelObjectiveData) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Attributes *SearchServiceLevelObjectiveAttributes `json:"attributes,omitempty"`
		Id         *string                                `json:"id,omitempty"`
		Type       *string                                `json:"type,omitempty"`
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
	if all.Attributes != nil && all.Attributes.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Attributes = all.Attributes
	o.Id = all.Id
	o.Type = all.Type
	return nil
}
