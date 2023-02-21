// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// HourlyUsageMetadata The object containing document metadata.
type HourlyUsageMetadata struct {
	// The metadata for the current pagination.
	Pagination *HourlyUsagePagination `json:"pagination,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewHourlyUsageMetadata instantiates a new HourlyUsageMetadata object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewHourlyUsageMetadata() *HourlyUsageMetadata {
	this := HourlyUsageMetadata{}
	return &this
}

// NewHourlyUsageMetadataWithDefaults instantiates a new HourlyUsageMetadata object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewHourlyUsageMetadataWithDefaults() *HourlyUsageMetadata {
	this := HourlyUsageMetadata{}
	return &this
}

// GetPagination returns the Pagination field value if set, zero value otherwise.
func (o *HourlyUsageMetadata) GetPagination() HourlyUsagePagination {
	if o == nil || o.Pagination == nil {
		var ret HourlyUsagePagination
		return ret
	}
	return *o.Pagination
}

// GetPaginationOk returns a tuple with the Pagination field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *HourlyUsageMetadata) GetPaginationOk() (*HourlyUsagePagination, bool) {
	if o == nil || o.Pagination == nil {
		return nil, false
	}
	return o.Pagination, true
}

// HasPagination returns a boolean if a field has been set.
func (o *HourlyUsageMetadata) HasPagination() bool {
	return o != nil && o.Pagination != nil
}

// SetPagination gets a reference to the given HourlyUsagePagination and assigns it to the Pagination field.
func (o *HourlyUsageMetadata) SetPagination(v HourlyUsagePagination) {
	o.Pagination = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o HourlyUsageMetadata) MarshalJSON() ([]byte, error) {
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
func (o *HourlyUsageMetadata) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Pagination *HourlyUsagePagination `json:"pagination,omitempty"`
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
