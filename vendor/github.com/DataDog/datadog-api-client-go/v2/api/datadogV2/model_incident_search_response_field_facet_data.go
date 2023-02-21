// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// IncidentSearchResponseFieldFacetData Facet value and number of occurrences for a property field of an incident.
type IncidentSearchResponseFieldFacetData struct {
	// Count of the facet value appearing in search results.
	Count *int32 `json:"count,omitempty"`
	// The facet value appearing in search results.
	Name *string `json:"name,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewIncidentSearchResponseFieldFacetData instantiates a new IncidentSearchResponseFieldFacetData object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewIncidentSearchResponseFieldFacetData() *IncidentSearchResponseFieldFacetData {
	this := IncidentSearchResponseFieldFacetData{}
	return &this
}

// NewIncidentSearchResponseFieldFacetDataWithDefaults instantiates a new IncidentSearchResponseFieldFacetData object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewIncidentSearchResponseFieldFacetDataWithDefaults() *IncidentSearchResponseFieldFacetData {
	this := IncidentSearchResponseFieldFacetData{}
	return &this
}

// GetCount returns the Count field value if set, zero value otherwise.
func (o *IncidentSearchResponseFieldFacetData) GetCount() int32 {
	if o == nil || o.Count == nil {
		var ret int32
		return ret
	}
	return *o.Count
}

// GetCountOk returns a tuple with the Count field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IncidentSearchResponseFieldFacetData) GetCountOk() (*int32, bool) {
	if o == nil || o.Count == nil {
		return nil, false
	}
	return o.Count, true
}

// HasCount returns a boolean if a field has been set.
func (o *IncidentSearchResponseFieldFacetData) HasCount() bool {
	return o != nil && o.Count != nil
}

// SetCount gets a reference to the given int32 and assigns it to the Count field.
func (o *IncidentSearchResponseFieldFacetData) SetCount(v int32) {
	o.Count = &v
}

// GetName returns the Name field value if set, zero value otherwise.
func (o *IncidentSearchResponseFieldFacetData) GetName() string {
	if o == nil || o.Name == nil {
		var ret string
		return ret
	}
	return *o.Name
}

// GetNameOk returns a tuple with the Name field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IncidentSearchResponseFieldFacetData) GetNameOk() (*string, bool) {
	if o == nil || o.Name == nil {
		return nil, false
	}
	return o.Name, true
}

// HasName returns a boolean if a field has been set.
func (o *IncidentSearchResponseFieldFacetData) HasName() bool {
	return o != nil && o.Name != nil
}

// SetName gets a reference to the given string and assigns it to the Name field.
func (o *IncidentSearchResponseFieldFacetData) SetName(v string) {
	o.Name = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o IncidentSearchResponseFieldFacetData) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Count != nil {
		toSerialize["count"] = o.Count
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
func (o *IncidentSearchResponseFieldFacetData) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Count *int32  `json:"count,omitempty"`
		Name  *string `json:"name,omitempty"`
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
	o.Name = all.Name
	return nil
}
