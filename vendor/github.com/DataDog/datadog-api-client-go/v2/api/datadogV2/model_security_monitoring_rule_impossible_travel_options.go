// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// SecurityMonitoringRuleImpossibleTravelOptions Options on impossible travel rules.
type SecurityMonitoringRuleImpossibleTravelOptions struct {
	// If true, signals are suppressed for the first 24 hours. In that time, Datadog learns the user's regular
	// access locations. This can be helpful to reduce noise and infer VPN usage or credentialed API access.
	BaselineUserLocations *bool `json:"baselineUserLocations,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewSecurityMonitoringRuleImpossibleTravelOptions instantiates a new SecurityMonitoringRuleImpossibleTravelOptions object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSecurityMonitoringRuleImpossibleTravelOptions() *SecurityMonitoringRuleImpossibleTravelOptions {
	this := SecurityMonitoringRuleImpossibleTravelOptions{}
	return &this
}

// NewSecurityMonitoringRuleImpossibleTravelOptionsWithDefaults instantiates a new SecurityMonitoringRuleImpossibleTravelOptions object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSecurityMonitoringRuleImpossibleTravelOptionsWithDefaults() *SecurityMonitoringRuleImpossibleTravelOptions {
	this := SecurityMonitoringRuleImpossibleTravelOptions{}
	return &this
}

// GetBaselineUserLocations returns the BaselineUserLocations field value if set, zero value otherwise.
func (o *SecurityMonitoringRuleImpossibleTravelOptions) GetBaselineUserLocations() bool {
	if o == nil || o.BaselineUserLocations == nil {
		var ret bool
		return ret
	}
	return *o.BaselineUserLocations
}

// GetBaselineUserLocationsOk returns a tuple with the BaselineUserLocations field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringRuleImpossibleTravelOptions) GetBaselineUserLocationsOk() (*bool, bool) {
	if o == nil || o.BaselineUserLocations == nil {
		return nil, false
	}
	return o.BaselineUserLocations, true
}

// HasBaselineUserLocations returns a boolean if a field has been set.
func (o *SecurityMonitoringRuleImpossibleTravelOptions) HasBaselineUserLocations() bool {
	return o != nil && o.BaselineUserLocations != nil
}

// SetBaselineUserLocations gets a reference to the given bool and assigns it to the BaselineUserLocations field.
func (o *SecurityMonitoringRuleImpossibleTravelOptions) SetBaselineUserLocations(v bool) {
	o.BaselineUserLocations = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SecurityMonitoringRuleImpossibleTravelOptions) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.BaselineUserLocations != nil {
		toSerialize["baselineUserLocations"] = o.BaselineUserLocations
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SecurityMonitoringRuleImpossibleTravelOptions) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		BaselineUserLocations *bool `json:"baselineUserLocations,omitempty"`
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
	o.BaselineUserLocations = all.BaselineUserLocations
	return nil
}
