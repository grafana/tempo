// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// RUMAggregateRequest The object sent with the request to retrieve aggregation buckets of RUM events from your organization.
type RUMAggregateRequest struct {
	// The list of metrics or timeseries to compute for the retrieved buckets.
	Compute []RUMCompute `json:"compute,omitempty"`
	// The search and filter query settings.
	Filter *RUMQueryFilter `json:"filter,omitempty"`
	// The rules for the group by.
	GroupBy []RUMGroupBy `json:"group_by,omitempty"`
	// Global query options that are used during the query.
	// Note: Only supply timezone or time offset, not both. Otherwise, the query fails.
	Options *RUMQueryOptions `json:"options,omitempty"`
	// Paging attributes for listing events.
	Page *RUMQueryPageOptions `json:"page,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewRUMAggregateRequest instantiates a new RUMAggregateRequest object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewRUMAggregateRequest() *RUMAggregateRequest {
	this := RUMAggregateRequest{}
	return &this
}

// NewRUMAggregateRequestWithDefaults instantiates a new RUMAggregateRequest object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewRUMAggregateRequestWithDefaults() *RUMAggregateRequest {
	this := RUMAggregateRequest{}
	return &this
}

// GetCompute returns the Compute field value if set, zero value otherwise.
func (o *RUMAggregateRequest) GetCompute() []RUMCompute {
	if o == nil || o.Compute == nil {
		var ret []RUMCompute
		return ret
	}
	return o.Compute
}

// GetComputeOk returns a tuple with the Compute field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *RUMAggregateRequest) GetComputeOk() (*[]RUMCompute, bool) {
	if o == nil || o.Compute == nil {
		return nil, false
	}
	return &o.Compute, true
}

// HasCompute returns a boolean if a field has been set.
func (o *RUMAggregateRequest) HasCompute() bool {
	return o != nil && o.Compute != nil
}

// SetCompute gets a reference to the given []RUMCompute and assigns it to the Compute field.
func (o *RUMAggregateRequest) SetCompute(v []RUMCompute) {
	o.Compute = v
}

// GetFilter returns the Filter field value if set, zero value otherwise.
func (o *RUMAggregateRequest) GetFilter() RUMQueryFilter {
	if o == nil || o.Filter == nil {
		var ret RUMQueryFilter
		return ret
	}
	return *o.Filter
}

// GetFilterOk returns a tuple with the Filter field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *RUMAggregateRequest) GetFilterOk() (*RUMQueryFilter, bool) {
	if o == nil || o.Filter == nil {
		return nil, false
	}
	return o.Filter, true
}

// HasFilter returns a boolean if a field has been set.
func (o *RUMAggregateRequest) HasFilter() bool {
	return o != nil && o.Filter != nil
}

// SetFilter gets a reference to the given RUMQueryFilter and assigns it to the Filter field.
func (o *RUMAggregateRequest) SetFilter(v RUMQueryFilter) {
	o.Filter = &v
}

// GetGroupBy returns the GroupBy field value if set, zero value otherwise.
func (o *RUMAggregateRequest) GetGroupBy() []RUMGroupBy {
	if o == nil || o.GroupBy == nil {
		var ret []RUMGroupBy
		return ret
	}
	return o.GroupBy
}

// GetGroupByOk returns a tuple with the GroupBy field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *RUMAggregateRequest) GetGroupByOk() (*[]RUMGroupBy, bool) {
	if o == nil || o.GroupBy == nil {
		return nil, false
	}
	return &o.GroupBy, true
}

// HasGroupBy returns a boolean if a field has been set.
func (o *RUMAggregateRequest) HasGroupBy() bool {
	return o != nil && o.GroupBy != nil
}

// SetGroupBy gets a reference to the given []RUMGroupBy and assigns it to the GroupBy field.
func (o *RUMAggregateRequest) SetGroupBy(v []RUMGroupBy) {
	o.GroupBy = v
}

// GetOptions returns the Options field value if set, zero value otherwise.
func (o *RUMAggregateRequest) GetOptions() RUMQueryOptions {
	if o == nil || o.Options == nil {
		var ret RUMQueryOptions
		return ret
	}
	return *o.Options
}

// GetOptionsOk returns a tuple with the Options field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *RUMAggregateRequest) GetOptionsOk() (*RUMQueryOptions, bool) {
	if o == nil || o.Options == nil {
		return nil, false
	}
	return o.Options, true
}

// HasOptions returns a boolean if a field has been set.
func (o *RUMAggregateRequest) HasOptions() bool {
	return o != nil && o.Options != nil
}

// SetOptions gets a reference to the given RUMQueryOptions and assigns it to the Options field.
func (o *RUMAggregateRequest) SetOptions(v RUMQueryOptions) {
	o.Options = &v
}

// GetPage returns the Page field value if set, zero value otherwise.
func (o *RUMAggregateRequest) GetPage() RUMQueryPageOptions {
	if o == nil || o.Page == nil {
		var ret RUMQueryPageOptions
		return ret
	}
	return *o.Page
}

// GetPageOk returns a tuple with the Page field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *RUMAggregateRequest) GetPageOk() (*RUMQueryPageOptions, bool) {
	if o == nil || o.Page == nil {
		return nil, false
	}
	return o.Page, true
}

// HasPage returns a boolean if a field has been set.
func (o *RUMAggregateRequest) HasPage() bool {
	return o != nil && o.Page != nil
}

// SetPage gets a reference to the given RUMQueryPageOptions and assigns it to the Page field.
func (o *RUMAggregateRequest) SetPage(v RUMQueryPageOptions) {
	o.Page = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o RUMAggregateRequest) MarshalJSON() ([]byte, error) {
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
	if o.Page != nil {
		toSerialize["page"] = o.Page
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *RUMAggregateRequest) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Compute []RUMCompute         `json:"compute,omitempty"`
		Filter  *RUMQueryFilter      `json:"filter,omitempty"`
		GroupBy []RUMGroupBy         `json:"group_by,omitempty"`
		Options *RUMQueryOptions     `json:"options,omitempty"`
		Page    *RUMQueryPageOptions `json:"page,omitempty"`
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
	if all.Page != nil && all.Page.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Page = all.Page
	return nil
}
