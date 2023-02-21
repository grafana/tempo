// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// DashboardListItems Dashboards within a list.
type DashboardListItems struct {
	// List of dashboards in the dashboard list.
	Dashboards []DashboardListItem `json:"dashboards"`
	// Number of dashboards in the dashboard list.
	Total *int64 `json:"total,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewDashboardListItems instantiates a new DashboardListItems object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewDashboardListItems(dashboards []DashboardListItem) *DashboardListItems {
	this := DashboardListItems{}
	this.Dashboards = dashboards
	return &this
}

// NewDashboardListItemsWithDefaults instantiates a new DashboardListItems object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewDashboardListItemsWithDefaults() *DashboardListItems {
	this := DashboardListItems{}
	return &this
}

// GetDashboards returns the Dashboards field value.
func (o *DashboardListItems) GetDashboards() []DashboardListItem {
	if o == nil {
		var ret []DashboardListItem
		return ret
	}
	return o.Dashboards
}

// GetDashboardsOk returns a tuple with the Dashboards field value
// and a boolean to check if the value has been set.
func (o *DashboardListItems) GetDashboardsOk() (*[]DashboardListItem, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Dashboards, true
}

// SetDashboards sets field value.
func (o *DashboardListItems) SetDashboards(v []DashboardListItem) {
	o.Dashboards = v
}

// GetTotal returns the Total field value if set, zero value otherwise.
func (o *DashboardListItems) GetTotal() int64 {
	if o == nil || o.Total == nil {
		var ret int64
		return ret
	}
	return *o.Total
}

// GetTotalOk returns a tuple with the Total field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *DashboardListItems) GetTotalOk() (*int64, bool) {
	if o == nil || o.Total == nil {
		return nil, false
	}
	return o.Total, true
}

// HasTotal returns a boolean if a field has been set.
func (o *DashboardListItems) HasTotal() bool {
	return o != nil && o.Total != nil
}

// SetTotal gets a reference to the given int64 and assigns it to the Total field.
func (o *DashboardListItems) SetTotal(v int64) {
	o.Total = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o DashboardListItems) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["dashboards"] = o.Dashboards
	if o.Total != nil {
		toSerialize["total"] = o.Total
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *DashboardListItems) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Dashboards *[]DashboardListItem `json:"dashboards"`
	}{}
	all := struct {
		Dashboards []DashboardListItem `json:"dashboards"`
		Total      *int64              `json:"total,omitempty"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Dashboards == nil {
		return fmt.Errorf("required field dashboards missing")
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
	o.Dashboards = all.Dashboards
	o.Total = all.Total
	return nil
}
