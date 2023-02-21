// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// EventsListRequest The object sent with the request to retrieve a list of events from your organization.
type EventsListRequest struct {
	// The search and filter query settings.
	Filter *EventsQueryFilter `json:"filter,omitempty"`
	// The global query options that are used. Either provide a timezone or a time offset but not both,
	// otherwise the query fails.
	Options *EventsQueryOptions `json:"options,omitempty"`
	// Pagination settings.
	Page *EventsRequestPage `json:"page,omitempty"`
	// The sort parameters when querying events.
	Sort *EventsSort `json:"sort,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewEventsListRequest instantiates a new EventsListRequest object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewEventsListRequest() *EventsListRequest {
	this := EventsListRequest{}
	return &this
}

// NewEventsListRequestWithDefaults instantiates a new EventsListRequest object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewEventsListRequestWithDefaults() *EventsListRequest {
	this := EventsListRequest{}
	return &this
}

// GetFilter returns the Filter field value if set, zero value otherwise.
func (o *EventsListRequest) GetFilter() EventsQueryFilter {
	if o == nil || o.Filter == nil {
		var ret EventsQueryFilter
		return ret
	}
	return *o.Filter
}

// GetFilterOk returns a tuple with the Filter field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *EventsListRequest) GetFilterOk() (*EventsQueryFilter, bool) {
	if o == nil || o.Filter == nil {
		return nil, false
	}
	return o.Filter, true
}

// HasFilter returns a boolean if a field has been set.
func (o *EventsListRequest) HasFilter() bool {
	return o != nil && o.Filter != nil
}

// SetFilter gets a reference to the given EventsQueryFilter and assigns it to the Filter field.
func (o *EventsListRequest) SetFilter(v EventsQueryFilter) {
	o.Filter = &v
}

// GetOptions returns the Options field value if set, zero value otherwise.
func (o *EventsListRequest) GetOptions() EventsQueryOptions {
	if o == nil || o.Options == nil {
		var ret EventsQueryOptions
		return ret
	}
	return *o.Options
}

// GetOptionsOk returns a tuple with the Options field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *EventsListRequest) GetOptionsOk() (*EventsQueryOptions, bool) {
	if o == nil || o.Options == nil {
		return nil, false
	}
	return o.Options, true
}

// HasOptions returns a boolean if a field has been set.
func (o *EventsListRequest) HasOptions() bool {
	return o != nil && o.Options != nil
}

// SetOptions gets a reference to the given EventsQueryOptions and assigns it to the Options field.
func (o *EventsListRequest) SetOptions(v EventsQueryOptions) {
	o.Options = &v
}

// GetPage returns the Page field value if set, zero value otherwise.
func (o *EventsListRequest) GetPage() EventsRequestPage {
	if o == nil || o.Page == nil {
		var ret EventsRequestPage
		return ret
	}
	return *o.Page
}

// GetPageOk returns a tuple with the Page field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *EventsListRequest) GetPageOk() (*EventsRequestPage, bool) {
	if o == nil || o.Page == nil {
		return nil, false
	}
	return o.Page, true
}

// HasPage returns a boolean if a field has been set.
func (o *EventsListRequest) HasPage() bool {
	return o != nil && o.Page != nil
}

// SetPage gets a reference to the given EventsRequestPage and assigns it to the Page field.
func (o *EventsListRequest) SetPage(v EventsRequestPage) {
	o.Page = &v
}

// GetSort returns the Sort field value if set, zero value otherwise.
func (o *EventsListRequest) GetSort() EventsSort {
	if o == nil || o.Sort == nil {
		var ret EventsSort
		return ret
	}
	return *o.Sort
}

// GetSortOk returns a tuple with the Sort field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *EventsListRequest) GetSortOk() (*EventsSort, bool) {
	if o == nil || o.Sort == nil {
		return nil, false
	}
	return o.Sort, true
}

// HasSort returns a boolean if a field has been set.
func (o *EventsListRequest) HasSort() bool {
	return o != nil && o.Sort != nil
}

// SetSort gets a reference to the given EventsSort and assigns it to the Sort field.
func (o *EventsListRequest) SetSort(v EventsSort) {
	o.Sort = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o EventsListRequest) MarshalJSON() ([]byte, error) {
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
func (o *EventsListRequest) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Filter  *EventsQueryFilter  `json:"filter,omitempty"`
		Options *EventsQueryOptions `json:"options,omitempty"`
		Page    *EventsRequestPage  `json:"page,omitempty"`
		Sort    *EventsSort         `json:"sort,omitempty"`
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
