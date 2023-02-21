// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// SecurityFilterExclusionFilter Exclusion filter for the security filter.
type SecurityFilterExclusionFilter struct {
	// Exclusion filter name.
	Name string `json:"name"`
	// Exclusion filter query. Logs that match this query are excluded from the security filter.
	Query string `json:"query"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewSecurityFilterExclusionFilter instantiates a new SecurityFilterExclusionFilter object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSecurityFilterExclusionFilter(name string, query string) *SecurityFilterExclusionFilter {
	this := SecurityFilterExclusionFilter{}
	this.Name = name
	this.Query = query
	return &this
}

// NewSecurityFilterExclusionFilterWithDefaults instantiates a new SecurityFilterExclusionFilter object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSecurityFilterExclusionFilterWithDefaults() *SecurityFilterExclusionFilter {
	this := SecurityFilterExclusionFilter{}
	return &this
}

// GetName returns the Name field value.
func (o *SecurityFilterExclusionFilter) GetName() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Name
}

// GetNameOk returns a tuple with the Name field value
// and a boolean to check if the value has been set.
func (o *SecurityFilterExclusionFilter) GetNameOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Name, true
}

// SetName sets field value.
func (o *SecurityFilterExclusionFilter) SetName(v string) {
	o.Name = v
}

// GetQuery returns the Query field value.
func (o *SecurityFilterExclusionFilter) GetQuery() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Query
}

// GetQueryOk returns a tuple with the Query field value
// and a boolean to check if the value has been set.
func (o *SecurityFilterExclusionFilter) GetQueryOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Query, true
}

// SetQuery sets field value.
func (o *SecurityFilterExclusionFilter) SetQuery(v string) {
	o.Query = v
}

// MarshalJSON serializes the struct using spec logic.
func (o SecurityFilterExclusionFilter) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["name"] = o.Name
	toSerialize["query"] = o.Query

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SecurityFilterExclusionFilter) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Name  *string `json:"name"`
		Query *string `json:"query"`
	}{}
	all := struct {
		Name  string `json:"name"`
		Query string `json:"query"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Name == nil {
		return fmt.Errorf("required field name missing")
	}
	if required.Query == nil {
		return fmt.Errorf("required field query missing")
	}
	err = json.Unmarshal(bytes, &all)
	if err != nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.Name = all.Name
	o.Query = all.Query
	return nil
}
