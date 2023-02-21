// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV1

import (
	"encoding/json"
	"fmt"
)

// SLOListWidgetQuery Updated SLO List widget.
type SLOListWidgetQuery struct {
	// Maximum number of results to display in the table.
	Limit *int64 `json:"limit,omitempty"`
	// Widget query.
	QueryString string `json:"query_string"`
	// Options for sorting results.
	Sort []WidgetFieldSort `json:"sort,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewSLOListWidgetQuery instantiates a new SLOListWidgetQuery object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSLOListWidgetQuery(queryString string) *SLOListWidgetQuery {
	this := SLOListWidgetQuery{}
	var limit int64 = 100
	this.Limit = &limit
	this.QueryString = queryString
	return &this
}

// NewSLOListWidgetQueryWithDefaults instantiates a new SLOListWidgetQuery object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSLOListWidgetQueryWithDefaults() *SLOListWidgetQuery {
	this := SLOListWidgetQuery{}
	var limit int64 = 100
	this.Limit = &limit
	return &this
}

// GetLimit returns the Limit field value if set, zero value otherwise.
func (o *SLOListWidgetQuery) GetLimit() int64 {
	if o == nil || o.Limit == nil {
		var ret int64
		return ret
	}
	return *o.Limit
}

// GetLimitOk returns a tuple with the Limit field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOListWidgetQuery) GetLimitOk() (*int64, bool) {
	if o == nil || o.Limit == nil {
		return nil, false
	}
	return o.Limit, true
}

// HasLimit returns a boolean if a field has been set.
func (o *SLOListWidgetQuery) HasLimit() bool {
	return o != nil && o.Limit != nil
}

// SetLimit gets a reference to the given int64 and assigns it to the Limit field.
func (o *SLOListWidgetQuery) SetLimit(v int64) {
	o.Limit = &v
}

// GetQueryString returns the QueryString field value.
func (o *SLOListWidgetQuery) GetQueryString() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.QueryString
}

// GetQueryStringOk returns a tuple with the QueryString field value
// and a boolean to check if the value has been set.
func (o *SLOListWidgetQuery) GetQueryStringOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.QueryString, true
}

// SetQueryString sets field value.
func (o *SLOListWidgetQuery) SetQueryString(v string) {
	o.QueryString = v
}

// GetSort returns the Sort field value if set, zero value otherwise.
func (o *SLOListWidgetQuery) GetSort() []WidgetFieldSort {
	if o == nil || o.Sort == nil {
		var ret []WidgetFieldSort
		return ret
	}
	return o.Sort
}

// GetSortOk returns a tuple with the Sort field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOListWidgetQuery) GetSortOk() (*[]WidgetFieldSort, bool) {
	if o == nil || o.Sort == nil {
		return nil, false
	}
	return &o.Sort, true
}

// HasSort returns a boolean if a field has been set.
func (o *SLOListWidgetQuery) HasSort() bool {
	return o != nil && o.Sort != nil
}

// SetSort gets a reference to the given []WidgetFieldSort and assigns it to the Sort field.
func (o *SLOListWidgetQuery) SetSort(v []WidgetFieldSort) {
	o.Sort = v
}

// MarshalJSON serializes the struct using spec logic.
func (o SLOListWidgetQuery) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Limit != nil {
		toSerialize["limit"] = o.Limit
	}
	toSerialize["query_string"] = o.QueryString
	if o.Sort != nil {
		toSerialize["sort"] = o.Sort
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SLOListWidgetQuery) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		QueryString *string `json:"query_string"`
	}{}
	all := struct {
		Limit       *int64            `json:"limit,omitempty"`
		QueryString string            `json:"query_string"`
		Sort        []WidgetFieldSort `json:"sort,omitempty"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.QueryString == nil {
		return fmt.Errorf("required field query_string missing")
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
	o.Limit = all.Limit
	o.QueryString = all.QueryString
	o.Sort = all.Sort
	return nil
}
