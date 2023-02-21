// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV1

import (
	"encoding/json"
)

// SearchSLOResponseDataAttributes Attributes
type SearchSLOResponseDataAttributes struct {
	// Facets
	Facets *SearchSLOResponseDataAttributesFacets `json:"facets,omitempty"`
	// SLOs
	Slos []SearchServiceLevelObjective `json:"slos,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewSearchSLOResponseDataAttributes instantiates a new SearchSLOResponseDataAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSearchSLOResponseDataAttributes() *SearchSLOResponseDataAttributes {
	this := SearchSLOResponseDataAttributes{}
	return &this
}

// NewSearchSLOResponseDataAttributesWithDefaults instantiates a new SearchSLOResponseDataAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSearchSLOResponseDataAttributesWithDefaults() *SearchSLOResponseDataAttributes {
	this := SearchSLOResponseDataAttributes{}
	return &this
}

// GetFacets returns the Facets field value if set, zero value otherwise.
func (o *SearchSLOResponseDataAttributes) GetFacets() SearchSLOResponseDataAttributesFacets {
	if o == nil || o.Facets == nil {
		var ret SearchSLOResponseDataAttributesFacets
		return ret
	}
	return *o.Facets
}

// GetFacetsOk returns a tuple with the Facets field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SearchSLOResponseDataAttributes) GetFacetsOk() (*SearchSLOResponseDataAttributesFacets, bool) {
	if o == nil || o.Facets == nil {
		return nil, false
	}
	return o.Facets, true
}

// HasFacets returns a boolean if a field has been set.
func (o *SearchSLOResponseDataAttributes) HasFacets() bool {
	return o != nil && o.Facets != nil
}

// SetFacets gets a reference to the given SearchSLOResponseDataAttributesFacets and assigns it to the Facets field.
func (o *SearchSLOResponseDataAttributes) SetFacets(v SearchSLOResponseDataAttributesFacets) {
	o.Facets = &v
}

// GetSlos returns the Slos field value if set, zero value otherwise.
func (o *SearchSLOResponseDataAttributes) GetSlos() []SearchServiceLevelObjective {
	if o == nil || o.Slos == nil {
		var ret []SearchServiceLevelObjective
		return ret
	}
	return o.Slos
}

// GetSlosOk returns a tuple with the Slos field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SearchSLOResponseDataAttributes) GetSlosOk() (*[]SearchServiceLevelObjective, bool) {
	if o == nil || o.Slos == nil {
		return nil, false
	}
	return &o.Slos, true
}

// HasSlos returns a boolean if a field has been set.
func (o *SearchSLOResponseDataAttributes) HasSlos() bool {
	return o != nil && o.Slos != nil
}

// SetSlos gets a reference to the given []SearchServiceLevelObjective and assigns it to the Slos field.
func (o *SearchSLOResponseDataAttributes) SetSlos(v []SearchServiceLevelObjective) {
	o.Slos = v
}

// MarshalJSON serializes the struct using spec logic.
func (o SearchSLOResponseDataAttributes) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Facets != nil {
		toSerialize["facets"] = o.Facets
	}
	if o.Slos != nil {
		toSerialize["slos"] = o.Slos
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SearchSLOResponseDataAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Facets *SearchSLOResponseDataAttributesFacets `json:"facets,omitempty"`
		Slos   []SearchServiceLevelObjective          `json:"slos,omitempty"`
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
	if all.Facets != nil && all.Facets.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Facets = all.Facets
	o.Slos = all.Slos
	return nil
}
