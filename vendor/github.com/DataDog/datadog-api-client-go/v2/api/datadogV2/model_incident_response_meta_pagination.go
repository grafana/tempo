// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// IncidentResponseMetaPagination Pagination properties.
type IncidentResponseMetaPagination struct {
	// The index of the first element in the next page of results. Equal to page size added to the current offset.
	NextOffset *int64 `json:"next_offset,omitempty"`
	// The index of the first element in the results.
	Offset *int64 `json:"offset,omitempty"`
	// Maximum size of pages to return.
	Size *int64 `json:"size,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewIncidentResponseMetaPagination instantiates a new IncidentResponseMetaPagination object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewIncidentResponseMetaPagination() *IncidentResponseMetaPagination {
	this := IncidentResponseMetaPagination{}
	return &this
}

// NewIncidentResponseMetaPaginationWithDefaults instantiates a new IncidentResponseMetaPagination object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewIncidentResponseMetaPaginationWithDefaults() *IncidentResponseMetaPagination {
	this := IncidentResponseMetaPagination{}
	return &this
}

// GetNextOffset returns the NextOffset field value if set, zero value otherwise.
func (o *IncidentResponseMetaPagination) GetNextOffset() int64 {
	if o == nil || o.NextOffset == nil {
		var ret int64
		return ret
	}
	return *o.NextOffset
}

// GetNextOffsetOk returns a tuple with the NextOffset field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IncidentResponseMetaPagination) GetNextOffsetOk() (*int64, bool) {
	if o == nil || o.NextOffset == nil {
		return nil, false
	}
	return o.NextOffset, true
}

// HasNextOffset returns a boolean if a field has been set.
func (o *IncidentResponseMetaPagination) HasNextOffset() bool {
	return o != nil && o.NextOffset != nil
}

// SetNextOffset gets a reference to the given int64 and assigns it to the NextOffset field.
func (o *IncidentResponseMetaPagination) SetNextOffset(v int64) {
	o.NextOffset = &v
}

// GetOffset returns the Offset field value if set, zero value otherwise.
func (o *IncidentResponseMetaPagination) GetOffset() int64 {
	if o == nil || o.Offset == nil {
		var ret int64
		return ret
	}
	return *o.Offset
}

// GetOffsetOk returns a tuple with the Offset field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IncidentResponseMetaPagination) GetOffsetOk() (*int64, bool) {
	if o == nil || o.Offset == nil {
		return nil, false
	}
	return o.Offset, true
}

// HasOffset returns a boolean if a field has been set.
func (o *IncidentResponseMetaPagination) HasOffset() bool {
	return o != nil && o.Offset != nil
}

// SetOffset gets a reference to the given int64 and assigns it to the Offset field.
func (o *IncidentResponseMetaPagination) SetOffset(v int64) {
	o.Offset = &v
}

// GetSize returns the Size field value if set, zero value otherwise.
func (o *IncidentResponseMetaPagination) GetSize() int64 {
	if o == nil || o.Size == nil {
		var ret int64
		return ret
	}
	return *o.Size
}

// GetSizeOk returns a tuple with the Size field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IncidentResponseMetaPagination) GetSizeOk() (*int64, bool) {
	if o == nil || o.Size == nil {
		return nil, false
	}
	return o.Size, true
}

// HasSize returns a boolean if a field has been set.
func (o *IncidentResponseMetaPagination) HasSize() bool {
	return o != nil && o.Size != nil
}

// SetSize gets a reference to the given int64 and assigns it to the Size field.
func (o *IncidentResponseMetaPagination) SetSize(v int64) {
	o.Size = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o IncidentResponseMetaPagination) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.NextOffset != nil {
		toSerialize["next_offset"] = o.NextOffset
	}
	if o.Offset != nil {
		toSerialize["offset"] = o.Offset
	}
	if o.Size != nil {
		toSerialize["size"] = o.Size
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *IncidentResponseMetaPagination) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		NextOffset *int64 `json:"next_offset,omitempty"`
		Offset     *int64 `json:"offset,omitempty"`
		Size       *int64 `json:"size,omitempty"`
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
	o.NextOffset = all.NextOffset
	o.Offset = all.Offset
	o.Size = all.Size
	return nil
}
