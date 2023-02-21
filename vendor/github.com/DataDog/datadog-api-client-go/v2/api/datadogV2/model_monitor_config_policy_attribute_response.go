// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// MonitorConfigPolicyAttributeResponse Policy and policy type for a monitor configuration policy.
type MonitorConfigPolicyAttributeResponse struct {
	// Configuration for the policy.
	Policy *MonitorConfigPolicyPolicy `json:"policy,omitempty"`
	// The monitor configuration policy type.
	PolicyType *MonitorConfigPolicyType `json:"policy_type,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewMonitorConfigPolicyAttributeResponse instantiates a new MonitorConfigPolicyAttributeResponse object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewMonitorConfigPolicyAttributeResponse() *MonitorConfigPolicyAttributeResponse {
	this := MonitorConfigPolicyAttributeResponse{}
	var policyType MonitorConfigPolicyType = MONITORCONFIGPOLICYTYPE_TAG
	this.PolicyType = &policyType
	return &this
}

// NewMonitorConfigPolicyAttributeResponseWithDefaults instantiates a new MonitorConfigPolicyAttributeResponse object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewMonitorConfigPolicyAttributeResponseWithDefaults() *MonitorConfigPolicyAttributeResponse {
	this := MonitorConfigPolicyAttributeResponse{}
	var policyType MonitorConfigPolicyType = MONITORCONFIGPOLICYTYPE_TAG
	this.PolicyType = &policyType
	return &this
}

// GetPolicy returns the Policy field value if set, zero value otherwise.
func (o *MonitorConfigPolicyAttributeResponse) GetPolicy() MonitorConfigPolicyPolicy {
	if o == nil || o.Policy == nil {
		var ret MonitorConfigPolicyPolicy
		return ret
	}
	return *o.Policy
}

// GetPolicyOk returns a tuple with the Policy field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorConfigPolicyAttributeResponse) GetPolicyOk() (*MonitorConfigPolicyPolicy, bool) {
	if o == nil || o.Policy == nil {
		return nil, false
	}
	return o.Policy, true
}

// HasPolicy returns a boolean if a field has been set.
func (o *MonitorConfigPolicyAttributeResponse) HasPolicy() bool {
	return o != nil && o.Policy != nil
}

// SetPolicy gets a reference to the given MonitorConfigPolicyPolicy and assigns it to the Policy field.
func (o *MonitorConfigPolicyAttributeResponse) SetPolicy(v MonitorConfigPolicyPolicy) {
	o.Policy = &v
}

// GetPolicyType returns the PolicyType field value if set, zero value otherwise.
func (o *MonitorConfigPolicyAttributeResponse) GetPolicyType() MonitorConfigPolicyType {
	if o == nil || o.PolicyType == nil {
		var ret MonitorConfigPolicyType
		return ret
	}
	return *o.PolicyType
}

// GetPolicyTypeOk returns a tuple with the PolicyType field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorConfigPolicyAttributeResponse) GetPolicyTypeOk() (*MonitorConfigPolicyType, bool) {
	if o == nil || o.PolicyType == nil {
		return nil, false
	}
	return o.PolicyType, true
}

// HasPolicyType returns a boolean if a field has been set.
func (o *MonitorConfigPolicyAttributeResponse) HasPolicyType() bool {
	return o != nil && o.PolicyType != nil
}

// SetPolicyType gets a reference to the given MonitorConfigPolicyType and assigns it to the PolicyType field.
func (o *MonitorConfigPolicyAttributeResponse) SetPolicyType(v MonitorConfigPolicyType) {
	o.PolicyType = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o MonitorConfigPolicyAttributeResponse) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Policy != nil {
		toSerialize["policy"] = o.Policy
	}
	if o.PolicyType != nil {
		toSerialize["policy_type"] = o.PolicyType
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *MonitorConfigPolicyAttributeResponse) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Policy     *MonitorConfigPolicyPolicy `json:"policy,omitempty"`
		PolicyType *MonitorConfigPolicyType   `json:"policy_type,omitempty"`
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
	if v := all.PolicyType; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.Policy = all.Policy
	o.PolicyType = all.PolicyType
	return nil
}
