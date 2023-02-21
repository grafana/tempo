// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// LogsAggregateRequest The object sent with the request to retrieve a list of logs from your organization.
type LogsAggregateRequest struct {
	// The list of metrics or timeseries to compute for the retrieved buckets.
	Compute []LogsCompute `json:"compute,omitempty"`
	// The search and filter query settings
	Filter *LogsQueryFilter `json:"filter,omitempty"`
	// The rules for the group by
	GroupBy []LogsGroupBy `json:"group_by,omitempty"`
	// Global query options that are used during the query.
	// Note: You should only supply timezone or time offset but not both otherwise the query will fail.
	Options *LogsQueryOptions `json:"options,omitempty"`
	// Paging settings
	Page *LogsAggregateRequestPage `json:"page,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewLogsAggregateRequest instantiates a new LogsAggregateRequest object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewLogsAggregateRequest() *LogsAggregateRequest {
	this := LogsAggregateRequest{}
	return &this
}

// NewLogsAggregateRequestWithDefaults instantiates a new LogsAggregateRequest object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewLogsAggregateRequestWithDefaults() *LogsAggregateRequest {
	this := LogsAggregateRequest{}
	return &this
}

// GetCompute returns the Compute field value if set, zero value otherwise.
func (o *LogsAggregateRequest) GetCompute() []LogsCompute {
	if o == nil || o.Compute == nil {
		var ret []LogsCompute
		return ret
	}
	return o.Compute
}

// GetComputeOk returns a tuple with the Compute field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsAggregateRequest) GetComputeOk() (*[]LogsCompute, bool) {
	if o == nil || o.Compute == nil {
		return nil, false
	}
	return &o.Compute, true
}

// HasCompute returns a boolean if a field has been set.
func (o *LogsAggregateRequest) HasCompute() bool {
	return o != nil && o.Compute != nil
}

// SetCompute gets a reference to the given []LogsCompute and assigns it to the Compute field.
func (o *LogsAggregateRequest) SetCompute(v []LogsCompute) {
	o.Compute = v
}

// GetFilter returns the Filter field value if set, zero value otherwise.
func (o *LogsAggregateRequest) GetFilter() LogsQueryFilter {
	if o == nil || o.Filter == nil {
		var ret LogsQueryFilter
		return ret
	}
	return *o.Filter
}

// GetFilterOk returns a tuple with the Filter field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsAggregateRequest) GetFilterOk() (*LogsQueryFilter, bool) {
	if o == nil || o.Filter == nil {
		return nil, false
	}
	return o.Filter, true
}

// HasFilter returns a boolean if a field has been set.
func (o *LogsAggregateRequest) HasFilter() bool {
	return o != nil && o.Filter != nil
}

// SetFilter gets a reference to the given LogsQueryFilter and assigns it to the Filter field.
func (o *LogsAggregateRequest) SetFilter(v LogsQueryFilter) {
	o.Filter = &v
}

// GetGroupBy returns the GroupBy field value if set, zero value otherwise.
func (o *LogsAggregateRequest) GetGroupBy() []LogsGroupBy {
	if o == nil || o.GroupBy == nil {
		var ret []LogsGroupBy
		return ret
	}
	return o.GroupBy
}

// GetGroupByOk returns a tuple with the GroupBy field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsAggregateRequest) GetGroupByOk() (*[]LogsGroupBy, bool) {
	if o == nil || o.GroupBy == nil {
		return nil, false
	}
	return &o.GroupBy, true
}

// HasGroupBy returns a boolean if a field has been set.
func (o *LogsAggregateRequest) HasGroupBy() bool {
	return o != nil && o.GroupBy != nil
}

// SetGroupBy gets a reference to the given []LogsGroupBy and assigns it to the GroupBy field.
func (o *LogsAggregateRequest) SetGroupBy(v []LogsGroupBy) {
	o.GroupBy = v
}

// GetOptions returns the Options field value if set, zero value otherwise.
func (o *LogsAggregateRequest) GetOptions() LogsQueryOptions {
	if o == nil || o.Options == nil {
		var ret LogsQueryOptions
		return ret
	}
	return *o.Options
}

// GetOptionsOk returns a tuple with the Options field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsAggregateRequest) GetOptionsOk() (*LogsQueryOptions, bool) {
	if o == nil || o.Options == nil {
		return nil, false
	}
	return o.Options, true
}

// HasOptions returns a boolean if a field has been set.
func (o *LogsAggregateRequest) HasOptions() bool {
	return o != nil && o.Options != nil
}

// SetOptions gets a reference to the given LogsQueryOptions and assigns it to the Options field.
func (o *LogsAggregateRequest) SetOptions(v LogsQueryOptions) {
	o.Options = &v
}

// GetPage returns the Page field value if set, zero value otherwise.
func (o *LogsAggregateRequest) GetPage() LogsAggregateRequestPage {
	if o == nil || o.Page == nil {
		var ret LogsAggregateRequestPage
		return ret
	}
	return *o.Page
}

// GetPageOk returns a tuple with the Page field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsAggregateRequest) GetPageOk() (*LogsAggregateRequestPage, bool) {
	if o == nil || o.Page == nil {
		return nil, false
	}
	return o.Page, true
}

// HasPage returns a boolean if a field has been set.
func (o *LogsAggregateRequest) HasPage() bool {
	return o != nil && o.Page != nil
}

// SetPage gets a reference to the given LogsAggregateRequestPage and assigns it to the Page field.
func (o *LogsAggregateRequest) SetPage(v LogsAggregateRequestPage) {
	o.Page = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o LogsAggregateRequest) MarshalJSON() ([]byte, error) {
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
func (o *LogsAggregateRequest) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Compute []LogsCompute             `json:"compute,omitempty"`
		Filter  *LogsQueryFilter          `json:"filter,omitempty"`
		GroupBy []LogsGroupBy             `json:"group_by,omitempty"`
		Options *LogsQueryOptions         `json:"options,omitempty"`
		Page    *LogsAggregateRequestPage `json:"page,omitempty"`
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
