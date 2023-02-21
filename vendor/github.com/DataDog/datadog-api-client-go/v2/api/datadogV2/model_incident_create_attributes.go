// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// IncidentCreateAttributes The incident's attributes for a create request.
type IncidentCreateAttributes struct {
	// A flag indicating whether the incident caused customer impact.
	CustomerImpacted bool `json:"customer_impacted"`
	// A condensed view of the user-defined fields for which to create initial selections.
	Fields map[string]IncidentFieldAttributes `json:"fields,omitempty"`
	// An array of initial timeline cells to be placed at the beginning of the incident timeline.
	InitialCells []IncidentTimelineCellCreateAttributes `json:"initial_cells,omitempty"`
	// Notification handles that will be notified of the incident at creation.
	NotificationHandles []IncidentNotificationHandle `json:"notification_handles,omitempty"`
	// The title of the incident, which summarizes what happened.
	Title string `json:"title"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewIncidentCreateAttributes instantiates a new IncidentCreateAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewIncidentCreateAttributes(customerImpacted bool, title string) *IncidentCreateAttributes {
	this := IncidentCreateAttributes{}
	this.CustomerImpacted = customerImpacted
	this.Title = title
	return &this
}

// NewIncidentCreateAttributesWithDefaults instantiates a new IncidentCreateAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewIncidentCreateAttributesWithDefaults() *IncidentCreateAttributes {
	this := IncidentCreateAttributes{}
	return &this
}

// GetCustomerImpacted returns the CustomerImpacted field value.
func (o *IncidentCreateAttributes) GetCustomerImpacted() bool {
	if o == nil {
		var ret bool
		return ret
	}
	return o.CustomerImpacted
}

// GetCustomerImpactedOk returns a tuple with the CustomerImpacted field value
// and a boolean to check if the value has been set.
func (o *IncidentCreateAttributes) GetCustomerImpactedOk() (*bool, bool) {
	if o == nil {
		return nil, false
	}
	return &o.CustomerImpacted, true
}

// SetCustomerImpacted sets field value.
func (o *IncidentCreateAttributes) SetCustomerImpacted(v bool) {
	o.CustomerImpacted = v
}

// GetFields returns the Fields field value if set, zero value otherwise.
func (o *IncidentCreateAttributes) GetFields() map[string]IncidentFieldAttributes {
	if o == nil || o.Fields == nil {
		var ret map[string]IncidentFieldAttributes
		return ret
	}
	return o.Fields
}

// GetFieldsOk returns a tuple with the Fields field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IncidentCreateAttributes) GetFieldsOk() (*map[string]IncidentFieldAttributes, bool) {
	if o == nil || o.Fields == nil {
		return nil, false
	}
	return &o.Fields, true
}

// HasFields returns a boolean if a field has been set.
func (o *IncidentCreateAttributes) HasFields() bool {
	return o != nil && o.Fields != nil
}

// SetFields gets a reference to the given map[string]IncidentFieldAttributes and assigns it to the Fields field.
func (o *IncidentCreateAttributes) SetFields(v map[string]IncidentFieldAttributes) {
	o.Fields = v
}

// GetInitialCells returns the InitialCells field value if set, zero value otherwise.
func (o *IncidentCreateAttributes) GetInitialCells() []IncidentTimelineCellCreateAttributes {
	if o == nil || o.InitialCells == nil {
		var ret []IncidentTimelineCellCreateAttributes
		return ret
	}
	return o.InitialCells
}

// GetInitialCellsOk returns a tuple with the InitialCells field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IncidentCreateAttributes) GetInitialCellsOk() (*[]IncidentTimelineCellCreateAttributes, bool) {
	if o == nil || o.InitialCells == nil {
		return nil, false
	}
	return &o.InitialCells, true
}

// HasInitialCells returns a boolean if a field has been set.
func (o *IncidentCreateAttributes) HasInitialCells() bool {
	return o != nil && o.InitialCells != nil
}

// SetInitialCells gets a reference to the given []IncidentTimelineCellCreateAttributes and assigns it to the InitialCells field.
func (o *IncidentCreateAttributes) SetInitialCells(v []IncidentTimelineCellCreateAttributes) {
	o.InitialCells = v
}

// GetNotificationHandles returns the NotificationHandles field value if set, zero value otherwise.
func (o *IncidentCreateAttributes) GetNotificationHandles() []IncidentNotificationHandle {
	if o == nil || o.NotificationHandles == nil {
		var ret []IncidentNotificationHandle
		return ret
	}
	return o.NotificationHandles
}

// GetNotificationHandlesOk returns a tuple with the NotificationHandles field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IncidentCreateAttributes) GetNotificationHandlesOk() (*[]IncidentNotificationHandle, bool) {
	if o == nil || o.NotificationHandles == nil {
		return nil, false
	}
	return &o.NotificationHandles, true
}

// HasNotificationHandles returns a boolean if a field has been set.
func (o *IncidentCreateAttributes) HasNotificationHandles() bool {
	return o != nil && o.NotificationHandles != nil
}

// SetNotificationHandles gets a reference to the given []IncidentNotificationHandle and assigns it to the NotificationHandles field.
func (o *IncidentCreateAttributes) SetNotificationHandles(v []IncidentNotificationHandle) {
	o.NotificationHandles = v
}

// GetTitle returns the Title field value.
func (o *IncidentCreateAttributes) GetTitle() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Title
}

// GetTitleOk returns a tuple with the Title field value
// and a boolean to check if the value has been set.
func (o *IncidentCreateAttributes) GetTitleOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Title, true
}

// SetTitle sets field value.
func (o *IncidentCreateAttributes) SetTitle(v string) {
	o.Title = v
}

// MarshalJSON serializes the struct using spec logic.
func (o IncidentCreateAttributes) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["customer_impacted"] = o.CustomerImpacted
	if o.Fields != nil {
		toSerialize["fields"] = o.Fields
	}
	if o.InitialCells != nil {
		toSerialize["initial_cells"] = o.InitialCells
	}
	if o.NotificationHandles != nil {
		toSerialize["notification_handles"] = o.NotificationHandles
	}
	toSerialize["title"] = o.Title

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *IncidentCreateAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		CustomerImpacted *bool   `json:"customer_impacted"`
		Title            *string `json:"title"`
	}{}
	all := struct {
		CustomerImpacted    bool                                   `json:"customer_impacted"`
		Fields              map[string]IncidentFieldAttributes     `json:"fields,omitempty"`
		InitialCells        []IncidentTimelineCellCreateAttributes `json:"initial_cells,omitempty"`
		NotificationHandles []IncidentNotificationHandle           `json:"notification_handles,omitempty"`
		Title               string                                 `json:"title"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.CustomerImpacted == nil {
		return fmt.Errorf("required field customer_impacted missing")
	}
	if required.Title == nil {
		return fmt.Errorf("required field title missing")
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
	o.CustomerImpacted = all.CustomerImpacted
	o.Fields = all.Fields
	o.InitialCells = all.InitialCells
	o.NotificationHandles = all.NotificationHandles
	o.Title = all.Title
	return nil
}
