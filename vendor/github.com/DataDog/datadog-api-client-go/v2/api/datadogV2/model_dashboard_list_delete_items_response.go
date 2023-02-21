// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// DashboardListDeleteItemsResponse Response containing a list of deleted dashboards.
type DashboardListDeleteItemsResponse struct {
	// List of dashboards deleted from the dashboard list.
	DeletedDashboardsFromList []DashboardListItemResponse `json:"deleted_dashboards_from_list,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewDashboardListDeleteItemsResponse instantiates a new DashboardListDeleteItemsResponse object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewDashboardListDeleteItemsResponse() *DashboardListDeleteItemsResponse {
	this := DashboardListDeleteItemsResponse{}
	return &this
}

// NewDashboardListDeleteItemsResponseWithDefaults instantiates a new DashboardListDeleteItemsResponse object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewDashboardListDeleteItemsResponseWithDefaults() *DashboardListDeleteItemsResponse {
	this := DashboardListDeleteItemsResponse{}
	return &this
}

// GetDeletedDashboardsFromList returns the DeletedDashboardsFromList field value if set, zero value otherwise.
func (o *DashboardListDeleteItemsResponse) GetDeletedDashboardsFromList() []DashboardListItemResponse {
	if o == nil || o.DeletedDashboardsFromList == nil {
		var ret []DashboardListItemResponse
		return ret
	}
	return o.DeletedDashboardsFromList
}

// GetDeletedDashboardsFromListOk returns a tuple with the DeletedDashboardsFromList field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *DashboardListDeleteItemsResponse) GetDeletedDashboardsFromListOk() (*[]DashboardListItemResponse, bool) {
	if o == nil || o.DeletedDashboardsFromList == nil {
		return nil, false
	}
	return &o.DeletedDashboardsFromList, true
}

// HasDeletedDashboardsFromList returns a boolean if a field has been set.
func (o *DashboardListDeleteItemsResponse) HasDeletedDashboardsFromList() bool {
	return o != nil && o.DeletedDashboardsFromList != nil
}

// SetDeletedDashboardsFromList gets a reference to the given []DashboardListItemResponse and assigns it to the DeletedDashboardsFromList field.
func (o *DashboardListDeleteItemsResponse) SetDeletedDashboardsFromList(v []DashboardListItemResponse) {
	o.DeletedDashboardsFromList = v
}

// MarshalJSON serializes the struct using spec logic.
func (o DashboardListDeleteItemsResponse) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.DeletedDashboardsFromList != nil {
		toSerialize["deleted_dashboards_from_list"] = o.DeletedDashboardsFromList
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *DashboardListDeleteItemsResponse) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		DeletedDashboardsFromList []DashboardListItemResponse `json:"deleted_dashboards_from_list,omitempty"`
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
	o.DeletedDashboardsFromList = all.DeletedDashboardsFromList
	return nil
}
