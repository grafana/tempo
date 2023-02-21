// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// IncidentResponseMeta The metadata object containing pagination metadata.
type IncidentResponseMeta struct {
	// Pagination properties.
	Pagination *IncidentResponseMetaPagination `json:"pagination,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewIncidentResponseMeta instantiates a new IncidentResponseMeta object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewIncidentResponseMeta() *IncidentResponseMeta {
	this := IncidentResponseMeta{}
	return &this
}

// NewIncidentResponseMetaWithDefaults instantiates a new IncidentResponseMeta object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewIncidentResponseMetaWithDefaults() *IncidentResponseMeta {
	this := IncidentResponseMeta{}
	return &this
}

// GetPagination returns the Pagination field value if set, zero value otherwise.
func (o *IncidentResponseMeta) GetPagination() IncidentResponseMetaPagination {
	if o == nil || o.Pagination == nil {
		var ret IncidentResponseMetaPagination
		return ret
	}
	return *o.Pagination
}

// GetPaginationOk returns a tuple with the Pagination field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IncidentResponseMeta) GetPaginationOk() (*IncidentResponseMetaPagination, bool) {
	if o == nil || o.Pagination == nil {
		return nil, false
	}
	return o.Pagination, true
}

// HasPagination returns a boolean if a field has been set.
func (o *IncidentResponseMeta) HasPagination() bool {
	return o != nil && o.Pagination != nil
}

// SetPagination gets a reference to the given IncidentResponseMetaPagination and assigns it to the Pagination field.
func (o *IncidentResponseMeta) SetPagination(v IncidentResponseMetaPagination) {
	o.Pagination = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o IncidentResponseMeta) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Pagination != nil {
		toSerialize["pagination"] = o.Pagination
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *IncidentResponseMeta) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Pagination *IncidentResponseMetaPagination `json:"pagination,omitempty"`
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
	if all.Pagination != nil && all.Pagination.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Pagination = all.Pagination
	return nil
}
