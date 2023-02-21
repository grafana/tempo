// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// SecurityFilterExclusionFilterResponse A single exclusion filter.
type SecurityFilterExclusionFilterResponse struct {
	// The exclusion filter name.
	Name *string `json:"name,omitempty"`
	// The exclusion filter query.
	Query *string `json:"query,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewSecurityFilterExclusionFilterResponse instantiates a new SecurityFilterExclusionFilterResponse object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSecurityFilterExclusionFilterResponse() *SecurityFilterExclusionFilterResponse {
	this := SecurityFilterExclusionFilterResponse{}
	return &this
}

// NewSecurityFilterExclusionFilterResponseWithDefaults instantiates a new SecurityFilterExclusionFilterResponse object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSecurityFilterExclusionFilterResponseWithDefaults() *SecurityFilterExclusionFilterResponse {
	this := SecurityFilterExclusionFilterResponse{}
	return &this
}

// GetName returns the Name field value if set, zero value otherwise.
func (o *SecurityFilterExclusionFilterResponse) GetName() string {
	if o == nil || o.Name == nil {
		var ret string
		return ret
	}
	return *o.Name
}

// GetNameOk returns a tuple with the Name field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityFilterExclusionFilterResponse) GetNameOk() (*string, bool) {
	if o == nil || o.Name == nil {
		return nil, false
	}
	return o.Name, true
}

// HasName returns a boolean if a field has been set.
func (o *SecurityFilterExclusionFilterResponse) HasName() bool {
	return o != nil && o.Name != nil
}

// SetName gets a reference to the given string and assigns it to the Name field.
func (o *SecurityFilterExclusionFilterResponse) SetName(v string) {
	o.Name = &v
}

// GetQuery returns the Query field value if set, zero value otherwise.
func (o *SecurityFilterExclusionFilterResponse) GetQuery() string {
	if o == nil || o.Query == nil {
		var ret string
		return ret
	}
	return *o.Query
}

// GetQueryOk returns a tuple with the Query field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityFilterExclusionFilterResponse) GetQueryOk() (*string, bool) {
	if o == nil || o.Query == nil {
		return nil, false
	}
	return o.Query, true
}

// HasQuery returns a boolean if a field has been set.
func (o *SecurityFilterExclusionFilterResponse) HasQuery() bool {
	return o != nil && o.Query != nil
}

// SetQuery gets a reference to the given string and assigns it to the Query field.
func (o *SecurityFilterExclusionFilterResponse) SetQuery(v string) {
	o.Query = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SecurityFilterExclusionFilterResponse) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Name != nil {
		toSerialize["name"] = o.Name
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
func (o *SecurityFilterExclusionFilterResponse) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Name  *string `json:"name,omitempty"`
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
	o.Name = all.Name
	o.Query = all.Query
	return nil
}
