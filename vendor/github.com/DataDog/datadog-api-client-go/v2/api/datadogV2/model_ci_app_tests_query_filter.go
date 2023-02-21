// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// CIAppTestsQueryFilter The search and filter query settings.
type CIAppTestsQueryFilter struct {
	// The minimum time for the requested events; supports date, math, and regular timestamps (in milliseconds).
	From *string `json:"from,omitempty"`
	// The search query following the Log search syntax.
	Query *string `json:"query,omitempty"`
	// The maximum time for the requested events, supports date, math, and regular timestamps (in milliseconds).
	To *string `json:"to,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewCIAppTestsQueryFilter instantiates a new CIAppTestsQueryFilter object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewCIAppTestsQueryFilter() *CIAppTestsQueryFilter {
	this := CIAppTestsQueryFilter{}
	var from string = "now-15m"
	this.From = &from
	var query string = "*"
	this.Query = &query
	var to string = "now"
	this.To = &to
	return &this
}

// NewCIAppTestsQueryFilterWithDefaults instantiates a new CIAppTestsQueryFilter object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewCIAppTestsQueryFilterWithDefaults() *CIAppTestsQueryFilter {
	this := CIAppTestsQueryFilter{}
	var from string = "now-15m"
	this.From = &from
	var query string = "*"
	this.Query = &query
	var to string = "now"
	this.To = &to
	return &this
}

// GetFrom returns the From field value if set, zero value otherwise.
func (o *CIAppTestsQueryFilter) GetFrom() string {
	if o == nil || o.From == nil {
		var ret string
		return ret
	}
	return *o.From
}

// GetFromOk returns a tuple with the From field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *CIAppTestsQueryFilter) GetFromOk() (*string, bool) {
	if o == nil || o.From == nil {
		return nil, false
	}
	return o.From, true
}

// HasFrom returns a boolean if a field has been set.
func (o *CIAppTestsQueryFilter) HasFrom() bool {
	return o != nil && o.From != nil
}

// SetFrom gets a reference to the given string and assigns it to the From field.
func (o *CIAppTestsQueryFilter) SetFrom(v string) {
	o.From = &v
}

// GetQuery returns the Query field value if set, zero value otherwise.
func (o *CIAppTestsQueryFilter) GetQuery() string {
	if o == nil || o.Query == nil {
		var ret string
		return ret
	}
	return *o.Query
}

// GetQueryOk returns a tuple with the Query field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *CIAppTestsQueryFilter) GetQueryOk() (*string, bool) {
	if o == nil || o.Query == nil {
		return nil, false
	}
	return o.Query, true
}

// HasQuery returns a boolean if a field has been set.
func (o *CIAppTestsQueryFilter) HasQuery() bool {
	return o != nil && o.Query != nil
}

// SetQuery gets a reference to the given string and assigns it to the Query field.
func (o *CIAppTestsQueryFilter) SetQuery(v string) {
	o.Query = &v
}

// GetTo returns the To field value if set, zero value otherwise.
func (o *CIAppTestsQueryFilter) GetTo() string {
	if o == nil || o.To == nil {
		var ret string
		return ret
	}
	return *o.To
}

// GetToOk returns a tuple with the To field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *CIAppTestsQueryFilter) GetToOk() (*string, bool) {
	if o == nil || o.To == nil {
		return nil, false
	}
	return o.To, true
}

// HasTo returns a boolean if a field has been set.
func (o *CIAppTestsQueryFilter) HasTo() bool {
	return o != nil && o.To != nil
}

// SetTo gets a reference to the given string and assigns it to the To field.
func (o *CIAppTestsQueryFilter) SetTo(v string) {
	o.To = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o CIAppTestsQueryFilter) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.From != nil {
		toSerialize["from"] = o.From
	}
	if o.Query != nil {
		toSerialize["query"] = o.Query
	}
	if o.To != nil {
		toSerialize["to"] = o.To
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *CIAppTestsQueryFilter) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		From  *string `json:"from,omitempty"`
		Query *string `json:"query,omitempty"`
		To    *string `json:"to,omitempty"`
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
	o.From = all.From
	o.Query = all.Query
	o.To = all.To
	return nil
}
