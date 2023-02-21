// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// SecurityMonitoringRuleCase Case when signal is generated.
type SecurityMonitoringRuleCase struct {
	// A rule case contains logical operations (`>`,`>=`, `&&`, `||`) to determine if a signal should be generated
	// based on the event counts in the previously defined queries.
	Condition *string `json:"condition,omitempty"`
	// Name of the case.
	Name *string `json:"name,omitempty"`
	// Notification targets for each rule case.
	Notifications []string `json:"notifications,omitempty"`
	// Severity of the Security Signal.
	Status *SecurityMonitoringRuleSeverity `json:"status,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewSecurityMonitoringRuleCase instantiates a new SecurityMonitoringRuleCase object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSecurityMonitoringRuleCase() *SecurityMonitoringRuleCase {
	this := SecurityMonitoringRuleCase{}
	return &this
}

// NewSecurityMonitoringRuleCaseWithDefaults instantiates a new SecurityMonitoringRuleCase object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSecurityMonitoringRuleCaseWithDefaults() *SecurityMonitoringRuleCase {
	this := SecurityMonitoringRuleCase{}
	return &this
}

// GetCondition returns the Condition field value if set, zero value otherwise.
func (o *SecurityMonitoringRuleCase) GetCondition() string {
	if o == nil || o.Condition == nil {
		var ret string
		return ret
	}
	return *o.Condition
}

// GetConditionOk returns a tuple with the Condition field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringRuleCase) GetConditionOk() (*string, bool) {
	if o == nil || o.Condition == nil {
		return nil, false
	}
	return o.Condition, true
}

// HasCondition returns a boolean if a field has been set.
func (o *SecurityMonitoringRuleCase) HasCondition() bool {
	return o != nil && o.Condition != nil
}

// SetCondition gets a reference to the given string and assigns it to the Condition field.
func (o *SecurityMonitoringRuleCase) SetCondition(v string) {
	o.Condition = &v
}

// GetName returns the Name field value if set, zero value otherwise.
func (o *SecurityMonitoringRuleCase) GetName() string {
	if o == nil || o.Name == nil {
		var ret string
		return ret
	}
	return *o.Name
}

// GetNameOk returns a tuple with the Name field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringRuleCase) GetNameOk() (*string, bool) {
	if o == nil || o.Name == nil {
		return nil, false
	}
	return o.Name, true
}

// HasName returns a boolean if a field has been set.
func (o *SecurityMonitoringRuleCase) HasName() bool {
	return o != nil && o.Name != nil
}

// SetName gets a reference to the given string and assigns it to the Name field.
func (o *SecurityMonitoringRuleCase) SetName(v string) {
	o.Name = &v
}

// GetNotifications returns the Notifications field value if set, zero value otherwise.
func (o *SecurityMonitoringRuleCase) GetNotifications() []string {
	if o == nil || o.Notifications == nil {
		var ret []string
		return ret
	}
	return o.Notifications
}

// GetNotificationsOk returns a tuple with the Notifications field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringRuleCase) GetNotificationsOk() (*[]string, bool) {
	if o == nil || o.Notifications == nil {
		return nil, false
	}
	return &o.Notifications, true
}

// HasNotifications returns a boolean if a field has been set.
func (o *SecurityMonitoringRuleCase) HasNotifications() bool {
	return o != nil && o.Notifications != nil
}

// SetNotifications gets a reference to the given []string and assigns it to the Notifications field.
func (o *SecurityMonitoringRuleCase) SetNotifications(v []string) {
	o.Notifications = v
}

// GetStatus returns the Status field value if set, zero value otherwise.
func (o *SecurityMonitoringRuleCase) GetStatus() SecurityMonitoringRuleSeverity {
	if o == nil || o.Status == nil {
		var ret SecurityMonitoringRuleSeverity
		return ret
	}
	return *o.Status
}

// GetStatusOk returns a tuple with the Status field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringRuleCase) GetStatusOk() (*SecurityMonitoringRuleSeverity, bool) {
	if o == nil || o.Status == nil {
		return nil, false
	}
	return o.Status, true
}

// HasStatus returns a boolean if a field has been set.
func (o *SecurityMonitoringRuleCase) HasStatus() bool {
	return o != nil && o.Status != nil
}

// SetStatus gets a reference to the given SecurityMonitoringRuleSeverity and assigns it to the Status field.
func (o *SecurityMonitoringRuleCase) SetStatus(v SecurityMonitoringRuleSeverity) {
	o.Status = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SecurityMonitoringRuleCase) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Condition != nil {
		toSerialize["condition"] = o.Condition
	}
	if o.Name != nil {
		toSerialize["name"] = o.Name
	}
	if o.Notifications != nil {
		toSerialize["notifications"] = o.Notifications
	}
	if o.Status != nil {
		toSerialize["status"] = o.Status
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SecurityMonitoringRuleCase) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Condition     *string                         `json:"condition,omitempty"`
		Name          *string                         `json:"name,omitempty"`
		Notifications []string                        `json:"notifications,omitempty"`
		Status        *SecurityMonitoringRuleSeverity `json:"status,omitempty"`
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
	if v := all.Status; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.Condition = all.Condition
	o.Name = all.Name
	o.Notifications = all.Notifications
	o.Status = all.Status
	return nil
}
