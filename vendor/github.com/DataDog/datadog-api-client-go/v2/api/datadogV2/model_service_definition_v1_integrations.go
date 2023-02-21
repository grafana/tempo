// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// ServiceDefinitionV1Integrations Third party integrations that Datadog supports.
type ServiceDefinitionV1Integrations struct {
	// PagerDuty service URL for the service.
	Pagerduty *string `json:"pagerduty,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewServiceDefinitionV1Integrations instantiates a new ServiceDefinitionV1Integrations object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewServiceDefinitionV1Integrations() *ServiceDefinitionV1Integrations {
	this := ServiceDefinitionV1Integrations{}
	return &this
}

// NewServiceDefinitionV1IntegrationsWithDefaults instantiates a new ServiceDefinitionV1Integrations object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewServiceDefinitionV1IntegrationsWithDefaults() *ServiceDefinitionV1Integrations {
	this := ServiceDefinitionV1Integrations{}
	return &this
}

// GetPagerduty returns the Pagerduty field value if set, zero value otherwise.
func (o *ServiceDefinitionV1Integrations) GetPagerduty() string {
	if o == nil || o.Pagerduty == nil {
		var ret string
		return ret
	}
	return *o.Pagerduty
}

// GetPagerdutyOk returns a tuple with the Pagerduty field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ServiceDefinitionV1Integrations) GetPagerdutyOk() (*string, bool) {
	if o == nil || o.Pagerduty == nil {
		return nil, false
	}
	return o.Pagerduty, true
}

// HasPagerduty returns a boolean if a field has been set.
func (o *ServiceDefinitionV1Integrations) HasPagerduty() bool {
	return o != nil && o.Pagerduty != nil
}

// SetPagerduty gets a reference to the given string and assigns it to the Pagerduty field.
func (o *ServiceDefinitionV1Integrations) SetPagerduty(v string) {
	o.Pagerduty = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o ServiceDefinitionV1Integrations) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
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
func (o *ServiceDefinitionV1Integrations) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Pagerduty *string `json:"pagerduty,omitempty"`
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
	o.Pagerduty = all.Pagerduty
	return nil
}
