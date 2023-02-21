// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// SensitiveDataScannerFilter Filter for the Scanning Group.
type SensitiveDataScannerFilter struct {
	// Query to filter the events.
	Query *string `json:"query,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewSensitiveDataScannerFilter instantiates a new SensitiveDataScannerFilter object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSensitiveDataScannerFilter() *SensitiveDataScannerFilter {
	this := SensitiveDataScannerFilter{}
	return &this
}

// NewSensitiveDataScannerFilterWithDefaults instantiates a new SensitiveDataScannerFilter object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSensitiveDataScannerFilterWithDefaults() *SensitiveDataScannerFilter {
	this := SensitiveDataScannerFilter{}
	return &this
}

// GetQuery returns the Query field value if set, zero value otherwise.
func (o *SensitiveDataScannerFilter) GetQuery() string {
	if o == nil || o.Query == nil {
		var ret string
		return ret
	}
	return *o.Query
}

// GetQueryOk returns a tuple with the Query field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SensitiveDataScannerFilter) GetQueryOk() (*string, bool) {
	if o == nil || o.Query == nil {
		return nil, false
	}
	return o.Query, true
}

// HasQuery returns a boolean if a field has been set.
func (o *SensitiveDataScannerFilter) HasQuery() bool {
	return o != nil && o.Query != nil
}

// SetQuery gets a reference to the given string and assigns it to the Query field.
func (o *SensitiveDataScannerFilter) SetQuery(v string) {
	o.Query = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SensitiveDataScannerFilter) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Query != nil {
		toSerialize["query"] = o.Query
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SensitiveDataScannerFilter) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Query *string `json:"query,omitempty"`
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
	o.Query = all.Query
	return nil
}
