// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// CloudConfigurationRuleCaseCreate Description of signals.
type CloudConfigurationRuleCaseCreate struct {
	// Notification targets for each rule case.
	Notifications []string `json:"notifications,omitempty"`
	// Severity of the Security Signal.
	Status SecurityMonitoringRuleSeverity `json:"status"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewCloudConfigurationRuleCaseCreate instantiates a new CloudConfigurationRuleCaseCreate object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewCloudConfigurationRuleCaseCreate(status SecurityMonitoringRuleSeverity) *CloudConfigurationRuleCaseCreate {
	this := CloudConfigurationRuleCaseCreate{}
	this.Status = status
	return &this
}

// NewCloudConfigurationRuleCaseCreateWithDefaults instantiates a new CloudConfigurationRuleCaseCreate object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewCloudConfigurationRuleCaseCreateWithDefaults() *CloudConfigurationRuleCaseCreate {
	this := CloudConfigurationRuleCaseCreate{}
	return &this
}

// GetNotifications returns the Notifications field value if set, zero value otherwise.
func (o *CloudConfigurationRuleCaseCreate) GetNotifications() []string {
	if o == nil || o.Notifications == nil {
		var ret []string
		return ret
	}
	return o.Notifications
}

// GetNotificationsOk returns a tuple with the Notifications field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *CloudConfigurationRuleCaseCreate) GetNotificationsOk() (*[]string, bool) {
	if o == nil || o.Notifications == nil {
		return nil, false
	}
	return &o.Notifications, true
}

// HasNotifications returns a boolean if a field has been set.
func (o *CloudConfigurationRuleCaseCreate) HasNotifications() bool {
	return o != nil && o.Notifications != nil
}

// SetNotifications gets a reference to the given []string and assigns it to the Notifications field.
func (o *CloudConfigurationRuleCaseCreate) SetNotifications(v []string) {
	o.Notifications = v
}

// GetStatus returns the Status field value.
func (o *CloudConfigurationRuleCaseCreate) GetStatus() SecurityMonitoringRuleSeverity {
	if o == nil {
		var ret SecurityMonitoringRuleSeverity
		return ret
	}
	return o.Status
}

// GetStatusOk returns a tuple with the Status field value
// and a boolean to check if the value has been set.
func (o *CloudConfigurationRuleCaseCreate) GetStatusOk() (*SecurityMonitoringRuleSeverity, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Status, true
}

// SetStatus sets field value.
func (o *CloudConfigurationRuleCaseCreate) SetStatus(v SecurityMonitoringRuleSeverity) {
	o.Status = v
}

// MarshalJSON serializes the struct using spec logic.
func (o CloudConfigurationRuleCaseCreate) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Notifications != nil {
		toSerialize["notifications"] = o.Notifications
	}
	toSerialize["status"] = o.Status

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *CloudConfigurationRuleCaseCreate) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Status *SecurityMonitoringRuleSeverity `json:"status"`
	}{}
	all := struct {
		Notifications []string                       `json:"notifications,omitempty"`
		Status        SecurityMonitoringRuleSeverity `json:"status"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Status == nil {
		return fmt.Errorf("required field status missing")
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
	if v := all.Status; !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.Notifications = all.Notifications
	o.Status = all.Status
	return nil
}
