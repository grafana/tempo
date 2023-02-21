// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// CIAppPipelineEventsRequest The request for a pipelines search.
type CIAppPipelineEventsRequest struct {
	// The search and filter query settings.
	Filter *CIAppPipelinesQueryFilter `json:"filter,omitempty"`
	// Global query options that are used during the query.
	// Only supply timezone or time offset, not both. Otherwise, the query fails.
	Options *CIAppQueryOptions `json:"options,omitempty"`
	// Paging attributes for listing events.
	Page *CIAppQueryPageOptions `json:"page,omitempty"`
	// Sort parameters when querying events.
	Sort *CIAppSort `json:"sort,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewCIAppPipelineEventsRequest instantiates a new CIAppPipelineEventsRequest object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewCIAppPipelineEventsRequest() *CIAppPipelineEventsRequest {
	this := CIAppPipelineEventsRequest{}
	return &this
}

// NewCIAppPipelineEventsRequestWithDefaults instantiates a new CIAppPipelineEventsRequest object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewCIAppPipelineEventsRequestWithDefaults() *CIAppPipelineEventsRequest {
	this := CIAppPipelineEventsRequest{}
	return &this
}

// GetFilter returns the Filter field value if set, zero value otherwise.
func (o *CIAppPipelineEventsRequest) GetFilter() CIAppPipelinesQueryFilter {
	if o == nil || o.Filter == nil {
		var ret CIAppPipelinesQueryFilter
		return ret
	}
	return *o.Filter
}

// GetFilterOk returns a tuple with the Filter field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *CIAppPipelineEventsRequest) GetFilterOk() (*CIAppPipelinesQueryFilter, bool) {
	if o == nil || o.Filter == nil {
		return nil, false
	}
	return o.Filter, true
}

// HasFilter returns a boolean if a field has been set.
func (o *CIAppPipelineEventsRequest) HasFilter() bool {
	return o != nil && o.Filter != nil
}

// SetFilter gets a reference to the given CIAppPipelinesQueryFilter and assigns it to the Filter field.
func (o *CIAppPipelineEventsRequest) SetFilter(v CIAppPipelinesQueryFilter) {
	o.Filter = &v
}

// GetOptions returns the Options field value if set, zero value otherwise.
func (o *CIAppPipelineEventsRequest) GetOptions() CIAppQueryOptions {
	if o == nil || o.Options == nil {
		var ret CIAppQueryOptions
		return ret
	}
	return *o.Options
}

// GetOptionsOk returns a tuple with the Options field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *CIAppPipelineEventsRequest) GetOptionsOk() (*CIAppQueryOptions, bool) {
	if o == nil || o.Options == nil {
		return nil, false
	}
	return o.Options, true
}

// HasOptions returns a boolean if a field has been set.
func (o *CIAppPipelineEventsRequest) HasOptions() bool {
	return o != nil && o.Options != nil
}

// SetOptions gets a reference to the given CIAppQueryOptions and assigns it to the Options field.
func (o *CIAppPipelineEventsRequest) SetOptions(v CIAppQueryOptions) {
	o.Options = &v
}

// GetPage returns the Page field value if set, zero value otherwise.
func (o *CIAppPipelineEventsRequest) GetPage() CIAppQueryPageOptions {
	if o == nil || o.Page == nil {
		var ret CIAppQueryPageOptions
		return ret
	}
	return *o.Page
}

// GetPageOk returns a tuple with the Page field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *CIAppPipelineEventsRequest) GetPageOk() (*CIAppQueryPageOptions, bool) {
	if o == nil || o.Page == nil {
		return nil, false
	}
	return o.Page, true
}

// HasPage returns a boolean if a field has been set.
func (o *CIAppPipelineEventsRequest) HasPage() bool {
	return o != nil && o.Page != nil
}

// SetPage gets a reference to the given CIAppQueryPageOptions and assigns it to the Page field.
func (o *CIAppPipelineEventsRequest) SetPage(v CIAppQueryPageOptions) {
	o.Page = &v
}

// GetSort returns the Sort field value if set, zero value otherwise.
func (o *CIAppPipelineEventsRequest) GetSort() CIAppSort {
	if o == nil || o.Sort == nil {
		var ret CIAppSort
		return ret
	}
	return *o.Sort
}

// GetSortOk returns a tuple with the Sort field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *CIAppPipelineEventsRequest) GetSortOk() (*CIAppSort, bool) {
	if o == nil || o.Sort == nil {
		return nil, false
	}
	return o.Sort, true
}

// HasSort returns a boolean if a field has been set.
func (o *CIAppPipelineEventsRequest) HasSort() bool {
	return o != nil && o.Sort != nil
}

// SetSort gets a reference to the given CIAppSort and assigns it to the Sort field.
func (o *CIAppPipelineEventsRequest) SetSort(v CIAppSort) {
	o.Sort = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o CIAppPipelineEventsRequest) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Filter != nil {
		toSerialize["filter"] = o.Filter
	}
	if o.Options != nil {
		toSerialize["options"] = o.Options
	}
	if o.Page != nil {
		toSerialize["page"] = o.Page
	}
	if o.Sort != nil {
		toSerialize["sort"] = o.Sort
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *CIAppPipelineEventsRequest) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Filter  *CIAppPipelinesQueryFilter `json:"filter,omitempty"`
		Options *CIAppQueryOptions         `json:"options,omitempty"`
		Page    *CIAppQueryPageOptions     `json:"page,omitempty"`
		Sort    *CIAppSort                 `json:"sort,omitempty"`
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
	if v := all.Sort; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	if all.Filter != nil && all.Filter.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Filter = all.Filter
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
	o.Sort = all.Sort
	return nil
}
