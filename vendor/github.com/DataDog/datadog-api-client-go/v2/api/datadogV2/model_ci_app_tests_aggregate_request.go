// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// CIAppTestsAggregateRequest The object sent with the request to retrieve aggregation buckets of test events from your organization.
type CIAppTestsAggregateRequest struct {
	// The list of metrics or timeseries to compute for the retrieved buckets.
	Compute []CIAppCompute `json:"compute,omitempty"`
	// The search and filter query settings.
	Filter *CIAppTestsQueryFilter `json:"filter,omitempty"`
	// The rules for the group-by.
	GroupBy []CIAppTestsGroupBy `json:"group_by,omitempty"`
	// Global query options that are used during the query.
	// Only supply timezone or time offset, not both. Otherwise, the query fails.
	Options *CIAppQueryOptions `json:"options,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewCIAppTestsAggregateRequest instantiates a new CIAppTestsAggregateRequest object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewCIAppTestsAggregateRequest() *CIAppTestsAggregateRequest {
	this := CIAppTestsAggregateRequest{}
	return &this
}

// NewCIAppTestsAggregateRequestWithDefaults instantiates a new CIAppTestsAggregateRequest object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewCIAppTestsAggregateRequestWithDefaults() *CIAppTestsAggregateRequest {
	this := CIAppTestsAggregateRequest{}
	return &this
}

// GetCompute returns the Compute field value if set, zero value otherwise.
func (o *CIAppTestsAggregateRequest) GetCompute() []CIAppCompute {
	if o == nil || o.Compute == nil {
		var ret []CIAppCompute
		return ret
	}
	return o.Compute
}

// GetComputeOk returns a tuple with the Compute field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *CIAppTestsAggregateRequest) GetComputeOk() (*[]CIAppCompute, bool) {
	if o == nil || o.Compute == nil {
		return nil, false
	}
	return &o.Compute, true
}

// HasCompute returns a boolean if a field has been set.
func (o *CIAppTestsAggregateRequest) HasCompute() bool {
	return o != nil && o.Compute != nil
}

// SetCompute gets a reference to the given []CIAppCompute and assigns it to the Compute field.
func (o *CIAppTestsAggregateRequest) SetCompute(v []CIAppCompute) {
	o.Compute = v
}

// GetFilter returns the Filter field value if set, zero value otherwise.
func (o *CIAppTestsAggregateRequest) GetFilter() CIAppTestsQueryFilter {
	if o == nil || o.Filter == nil {
		var ret CIAppTestsQueryFilter
		return ret
	}
	return *o.Filter
}

// GetFilterOk returns a tuple with the Filter field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *CIAppTestsAggregateRequest) GetFilterOk() (*CIAppTestsQueryFilter, bool) {
	if o == nil || o.Filter == nil {
		return nil, false
	}
	return o.Filter, true
}

// HasFilter returns a boolean if a field has been set.
func (o *CIAppTestsAggregateRequest) HasFilter() bool {
	return o != nil && o.Filter != nil
}

// SetFilter gets a reference to the given CIAppTestsQueryFilter and assigns it to the Filter field.
func (o *CIAppTestsAggregateRequest) SetFilter(v CIAppTestsQueryFilter) {
	o.Filter = &v
}

// GetGroupBy returns the GroupBy field value if set, zero value otherwise.
func (o *CIAppTestsAggregateRequest) GetGroupBy() []CIAppTestsGroupBy {
	if o == nil || o.GroupBy == nil {
		var ret []CIAppTestsGroupBy
		return ret
	}
	return o.GroupBy
}

// GetGroupByOk returns a tuple with the GroupBy field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *CIAppTestsAggregateRequest) GetGroupByOk() (*[]CIAppTestsGroupBy, bool) {
	if o == nil || o.GroupBy == nil {
		return nil, false
	}
	return &o.GroupBy, true
}

// HasGroupBy returns a boolean if a field has been set.
func (o *CIAppTestsAggregateRequest) HasGroupBy() bool {
	return o != nil && o.GroupBy != nil
}

// SetGroupBy gets a reference to the given []CIAppTestsGroupBy and assigns it to the GroupBy field.
func (o *CIAppTestsAggregateRequest) SetGroupBy(v []CIAppTestsGroupBy) {
	o.GroupBy = v
}

// GetOptions returns the Options field value if set, zero value otherwise.
func (o *CIAppTestsAggregateRequest) GetOptions() CIAppQueryOptions {
	if o == nil || o.Options == nil {
		var ret CIAppQueryOptions
		return ret
	}
	return *o.Options
}

// GetOptionsOk returns a tuple with the Options field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *CIAppTestsAggregateRequest) GetOptionsOk() (*CIAppQueryOptions, bool) {
	if o == nil || o.Options == nil {
		return nil, false
	}
	return o.Options, true
}

// HasOptions returns a boolean if a field has been set.
func (o *CIAppTestsAggregateRequest) HasOptions() bool {
	return o != nil && o.Options != nil
}

// SetOptions gets a reference to the given CIAppQueryOptions and assigns it to the Options field.
func (o *CIAppTestsAggregateRequest) SetOptions(v CIAppQueryOptions) {
	o.Options = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o CIAppTestsAggregateRequest) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Compute != nil {
		toSerialize["compute"] = o.Compute
	}
	if o.Filter != nil {
		toSerialize["filter"] = o.Filter
	}
	if o.GroupBy != nil {
		toSerialize["group_by"] = o.GroupBy
	}
	if o.Options != nil {
		toSerialize["options"] = o.Options
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *CIAppTestsAggregateRequest) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Compute []CIAppCompute         `json:"compute,omitempty"`
		Filter  *CIAppTestsQueryFilter `json:"filter,omitempty"`
		GroupBy []CIAppTestsGroupBy    `json:"group_by,omitempty"`
		Options *CIAppQueryOptions     `json:"options,omitempty"`
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
	o.Compute = all.Compute
	if all.Filter != nil && all.Filter.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Filter = all.Filter
	o.GroupBy = all.GroupBy
	if all.Options != nil && all.Options.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Options = all.Options
	return nil
}
