// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// ServiceDefinitionV2Integrations Third party integrations that Datadog supports.
type ServiceDefinitionV2Integrations struct {
	// Opsgenie integration for the service.
	Opsgenie *ServiceDefinitionV2Opsgenie `json:"opsgenie,omitempty"`
	// PagerDuty service URL for the service.
	Pagerduty *string `json:"pagerduty,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewServiceDefinitionV2Integrations instantiates a new ServiceDefinitionV2Integrations object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewServiceDefinitionV2Integrations() *ServiceDefinitionV2Integrations {
	this := ServiceDefinitionV2Integrations{}
	return &this
}

// NewServiceDefinitionV2IntegrationsWithDefaults instantiates a new ServiceDefinitionV2Integrations object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewServiceDefinitionV2IntegrationsWithDefaults() *ServiceDefinitionV2Integrations {
	this := ServiceDefinitionV2Integrations{}
	return &this
}

// GetOpsgenie returns the Opsgenie field value if set, zero value otherwise.
func (o *ServiceDefinitionV2Integrations) GetOpsgenie() ServiceDefinitionV2Opsgenie {
	if o == nil || o.Opsgenie == nil {
		var ret ServiceDefinitionV2Opsgenie
		return ret
	}
	return *o.Opsgenie
}

// GetOpsgenieOk returns a tuple with the Opsgenie field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ServiceDefinitionV2Integrations) GetOpsgenieOk() (*ServiceDefinitionV2Opsgenie, bool) {
	if o == nil || o.Opsgenie == nil {
		return nil, false
	}
	return o.Opsgenie, true
}

// HasOpsgenie returns a boolean if a field has been set.
func (o *ServiceDefinitionV2Integrations) HasOpsgenie() bool {
	return o != nil && o.Opsgenie != nil
}

// SetOpsgenie gets a reference to the given ServiceDefinitionV2Opsgenie and assigns it to the Opsgenie field.
func (o *ServiceDefinitionV2Integrations) SetOpsgenie(v ServiceDefinitionV2Opsgenie) {
	o.Opsgenie = &v
}

// GetPagerduty returns the Pagerduty field value if set, zero value otherwise.
func (o *ServiceDefinitionV2Integrations) GetPagerduty() string {
	if o == nil || o.Pagerduty == nil {
		var ret string
		return ret
	}
	return *o.Pagerduty
}

// GetPagerdutyOk returns a tuple with the Pagerduty field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ServiceDefinitionV2Integrations) GetPagerdutyOk() (*string, bool) {
	if o == nil || o.Pagerduty == nil {
		return nil, false
	}
	return o.Pagerduty, true
}

// HasPagerduty returns a boolean if a field has been set.
func (o *ServiceDefinitionV2Integrations) HasPagerduty() bool {
	return o != nil && o.Pagerduty != nil
}

// SetPagerduty gets a reference to the given string and assigns it to the Pagerduty field.
func (o *ServiceDefinitionV2Integrations) SetPagerduty(v string) {
	o.Pagerduty = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o ServiceDefinitionV2Integrations) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Opsgenie != nil {
		toSerialize["opsgenie"] = o.Opsgenie
	}
	if o.Pagerduty != nil {
		toSerialize["pagerduty"] = o.Pagerduty
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *ServiceDefinitionV2Integrations) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Opsgenie  *ServiceDefinitionV2Opsgenie `json:"opsgenie,omitempty"`
		Pagerduty *string                      `json:"pagerduty,omitempty"`
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
	if all.Opsgenie != nil && all.Opsgenie.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Opsgenie = all.Opsgenie
	o.Pagerduty = all.Pagerduty
	return nil
}
