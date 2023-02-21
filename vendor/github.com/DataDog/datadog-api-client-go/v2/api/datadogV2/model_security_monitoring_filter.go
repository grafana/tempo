// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// SecurityMonitoringFilter The rule's suppression filter.
type SecurityMonitoringFilter struct {
	// The type of filtering action.
	Action *SecurityMonitoringFilterAction `json:"action,omitempty"`
	// Query for selecting logs to apply the filtering action.
	Query *string `json:"query,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewSecurityMonitoringFilter instantiates a new SecurityMonitoringFilter object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSecurityMonitoringFilter() *SecurityMonitoringFilter {
	this := SecurityMonitoringFilter{}
	return &this
}

// NewSecurityMonitoringFilterWithDefaults instantiates a new SecurityMonitoringFilter object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSecurityMonitoringFilterWithDefaults() *SecurityMonitoringFilter {
	this := SecurityMonitoringFilter{}
	return &this
}

// GetAction returns the Action field value if set, zero value otherwise.
func (o *SecurityMonitoringFilter) GetAction() SecurityMonitoringFilterAction {
	if o == nil || o.Action == nil {
		var ret SecurityMonitoringFilterAction
		return ret
	}
	return *o.Action
}

// GetActionOk returns a tuple with the Action field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringFilter) GetActionOk() (*SecurityMonitoringFilterAction, bool) {
	if o == nil || o.Action == nil {
		return nil, false
	}
	return o.Action, true
}

// HasAction returns a boolean if a field has been set.
func (o *SecurityMonitoringFilter) HasAction() bool {
	return o != nil && o.Action != nil
}

// SetAction gets a reference to the given SecurityMonitoringFilterAction and assigns it to the Action field.
func (o *SecurityMonitoringFilter) SetAction(v SecurityMonitoringFilterAction) {
	o.Action = &v
}

// GetQuery returns the Query field value if set, zero value otherwise.
func (o *SecurityMonitoringFilter) GetQuery() string {
	if o == nil || o.Query == nil {
		var ret string
		return ret
	}
	return *o.Query
}

// GetQueryOk returns a tuple with the Query field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringFilter) GetQueryOk() (*string, bool) {
	if o == nil || o.Query == nil {
		return nil, false
	}
	return o.Query, true
}

// HasQuery returns a boolean if a field has been set.
func (o *SecurityMonitoringFilter) HasQuery() bool {
	return o != nil && o.Query != nil
}

// SetQuery gets a reference to the given string and assigns it to the Query field.
func (o *SecurityMonitoringFilter) SetQuery(v string) {
	o.Query = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SecurityMonitoringFilter) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Action != nil {
		toSerialize["action"] = o.Action
	}
	if o.Query != nil {
		toSerialize["query"] = o.Query
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SecurityMonitoringFilter) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Action *SecurityMonitoringFilterAction `json:"action,omitempty"`
		Query  *string                         `json:"query,omitempty"`
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
	if v := all.Action; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.Action = all.Action
	o.Query = all.Query
	return nil
}
