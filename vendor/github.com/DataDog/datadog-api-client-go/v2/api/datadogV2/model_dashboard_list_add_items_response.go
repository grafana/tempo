// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// DashboardListAddItemsResponse Response containing a list of added dashboards.
type DashboardListAddItemsResponse struct {
	// List of dashboards added to the dashboard list.
	AddedDashboardsToList []DashboardListItemResponse `json:"added_dashboards_to_list,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewDashboardListAddItemsResponse instantiates a new DashboardListAddItemsResponse object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewDashboardListAddItemsResponse() *DashboardListAddItemsResponse {
	this := DashboardListAddItemsResponse{}
	return &this
}

// NewDashboardListAddItemsResponseWithDefaults instantiates a new DashboardListAddItemsResponse object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewDashboardListAddItemsResponseWithDefaults() *DashboardListAddItemsResponse {
	this := DashboardListAddItemsResponse{}
	return &this
}

// GetAddedDashboardsToList returns the AddedDashboardsToList field value if set, zero value otherwise.
func (o *DashboardListAddItemsResponse) GetAddedDashboardsToList() []DashboardListItemResponse {
	if o == nil || o.AddedDashboardsToList == nil {
		var ret []DashboardListItemResponse
		return ret
	}
	return o.AddedDashboardsToList
}

// GetAddedDashboardsToListOk returns a tuple with the AddedDashboardsToList field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *DashboardListAddItemsResponse) GetAddedDashboardsToListOk() (*[]DashboardListItemResponse, bool) {
	if o == nil || o.AddedDashboardsToList == nil {
		return nil, false
	}
	return &o.AddedDashboardsToList, true
}

// HasAddedDashboardsToList returns a boolean if a field has been set.
func (o *DashboardListAddItemsResponse) HasAddedDashboardsToList() bool {
	return o != nil && o.AddedDashboardsToList != nil
}

// SetAddedDashboardsToList gets a reference to the given []DashboardListItemResponse and assigns it to the AddedDashboardsToList field.
func (o *DashboardListAddItemsResponse) SetAddedDashboardsToList(v []DashboardListItemResponse) {
	o.AddedDashboardsToList = v
}

// MarshalJSON serializes the struct using spec logic.
func (o DashboardListAddItemsResponse) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.AddedDashboardsToList != nil {
		toSerialize["added_dashboards_to_list"] = o.AddedDashboardsToList
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *DashboardListAddItemsResponse) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		AddedDashboardsToList []DashboardListItemResponse `json:"added_dashboards_to_list,omitempty"`
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
	o.AddedDashboardsToList = all.AddedDashboardsToList
	return nil
}
