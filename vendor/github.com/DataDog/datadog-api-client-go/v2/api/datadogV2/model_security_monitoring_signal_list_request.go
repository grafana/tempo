// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// SecurityMonitoringSignalListRequest The request for a security signal list.
type SecurityMonitoringSignalListRequest struct {
	// Search filters for listing security signals.
	Filter *SecurityMonitoringSignalListRequestFilter `json:"filter,omitempty"`
	// The paging attributes for listing security signals.
	Page *SecurityMonitoringSignalListRequestPage `json:"page,omitempty"`
	// The sort parameters used for querying security signals.
	Sort *SecurityMonitoringSignalsSort `json:"sort,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewSecurityMonitoringSignalListRequest instantiates a new SecurityMonitoringSignalListRequest object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSecurityMonitoringSignalListRequest() *SecurityMonitoringSignalListRequest {
	this := SecurityMonitoringSignalListRequest{}
	return &this
}

// NewSecurityMonitoringSignalListRequestWithDefaults instantiates a new SecurityMonitoringSignalListRequest object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSecurityMonitoringSignalListRequestWithDefaults() *SecurityMonitoringSignalListRequest {
	this := SecurityMonitoringSignalListRequest{}
	return &this
}

// GetFilter returns the Filter field value if set, zero value otherwise.
func (o *SecurityMonitoringSignalListRequest) GetFilter() SecurityMonitoringSignalListRequestFilter {
	if o == nil || o.Filter == nil {
		var ret SecurityMonitoringSignalListRequestFilter
		return ret
	}
	return *o.Filter
}

// GetFilterOk returns a tuple with the Filter field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringSignalListRequest) GetFilterOk() (*SecurityMonitoringSignalListRequestFilter, bool) {
	if o == nil || o.Filter == nil {
		return nil, false
	}
	return o.Filter, true
}

// HasFilter returns a boolean if a field has been set.
func (o *SecurityMonitoringSignalListRequest) HasFilter() bool {
	return o != nil && o.Filter != nil
}

// SetFilter gets a reference to the given SecurityMonitoringSignalListRequestFilter and assigns it to the Filter field.
func (o *SecurityMonitoringSignalListRequest) SetFilter(v SecurityMonitoringSignalListRequestFilter) {
	o.Filter = &v
}

// GetPage returns the Page field value if set, zero value otherwise.
func (o *SecurityMonitoringSignalListRequest) GetPage() SecurityMonitoringSignalListRequestPage {
	if o == nil || o.Page == nil {
		var ret SecurityMonitoringSignalListRequestPage
		return ret
	}
	return *o.Page
}

// GetPageOk returns a tuple with the Page field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringSignalListRequest) GetPageOk() (*SecurityMonitoringSignalListRequestPage, bool) {
	if o == nil || o.Page == nil {
		return nil, false
	}
	return o.Page, true
}

// HasPage returns a boolean if a field has been set.
func (o *SecurityMonitoringSignalListRequest) HasPage() bool {
	return o != nil && o.Page != nil
}

// SetPage gets a reference to the given SecurityMonitoringSignalListRequestPage and assigns it to the Page field.
func (o *SecurityMonitoringSignalListRequest) SetPage(v SecurityMonitoringSignalListRequestPage) {
	o.Page = &v
}

// GetSort returns the Sort field value if set, zero value otherwise.
func (o *SecurityMonitoringSignalListRequest) GetSort() SecurityMonitoringSignalsSort {
	if o == nil || o.Sort == nil {
		var ret SecurityMonitoringSignalsSort
		return ret
	}
	return *o.Sort
}

// GetSortOk returns a tuple with the Sort field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringSignalListRequest) GetSortOk() (*SecurityMonitoringSignalsSort, bool) {
	if o == nil || o.Sort == nil {
		return nil, false
	}
	return o.Sort, true
}

// HasSort returns a boolean if a field has been set.
func (o *SecurityMonitoringSignalListRequest) HasSort() bool {
	return o != nil && o.Sort != nil
}

// SetSort gets a reference to the given SecurityMonitoringSignalsSort and assigns it to the Sort field.
func (o *SecurityMonitoringSignalListRequest) SetSort(v SecurityMonitoringSignalsSort) {
	o.Sort = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SecurityMonitoringSignalListRequest) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Filter != nil {
		toSerialize["filter"] = o.Filter
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
func (o *SecurityMonitoringSignalListRequest) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Filter *SecurityMonitoringSignalListRequestFilter `json:"filter,omitempty"`
		Page   *SecurityMonitoringSignalListRequestPage   `json:"page,omitempty"`
		Sort   *SecurityMonitoringSignalsSort             `json:"sort,omitempty"`
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
