// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// ServiceDefinitionV1Info Basic information about a service.
type ServiceDefinitionV1Info struct {
	// Unique identifier of the service. Must be unique across all services and is used to match with a service in Datadog.
	DdService string `json:"dd-service"`
	// A short description of the service.
	Description *string `json:"description,omitempty"`
	// A friendly name of the service.
	DisplayName *string `json:"display-name,omitempty"`
	// Service tier.
	ServiceTier *string `json:"service-tier,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewServiceDefinitionV1Info instantiates a new ServiceDefinitionV1Info object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewServiceDefinitionV1Info(ddService string) *ServiceDefinitionV1Info {
	this := ServiceDefinitionV1Info{}
	this.DdService = ddService
	return &this
}

// NewServiceDefinitionV1InfoWithDefaults instantiates a new ServiceDefinitionV1Info object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewServiceDefinitionV1InfoWithDefaults() *ServiceDefinitionV1Info {
	this := ServiceDefinitionV1Info{}
	return &this
}

// GetDdService returns the DdService field value.
func (o *ServiceDefinitionV1Info) GetDdService() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.DdService
}

// GetDdServiceOk returns a tuple with the DdService field value
// and a boolean to check if the value has been set.
func (o *ServiceDefinitionV1Info) GetDdServiceOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.DdService, true
}

// SetDdService sets field value.
func (o *ServiceDefinitionV1Info) SetDdService(v string) {
	o.DdService = v
}

// GetDescription returns the Description field value if set, zero value otherwise.
func (o *ServiceDefinitionV1Info) GetDescription() string {
	if o == nil || o.Description == nil {
		var ret string
		return ret
	}
	return *o.Description
}

// GetDescriptionOk returns a tuple with the Description field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ServiceDefinitionV1Info) GetDescriptionOk() (*string, bool) {
	if o == nil || o.Description == nil {
		return nil, false
	}
	return o.Description, true
}

// HasDescription returns a boolean if a field has been set.
func (o *ServiceDefinitionV1Info) HasDescription() bool {
	return o != nil && o.Description != nil
}

// SetDescription gets a reference to the given string and assigns it to the Description field.
func (o *ServiceDefinitionV1Info) SetDescription(v string) {
	o.Description = &v
}

// GetDisplayName returns the DisplayName field value if set, zero value otherwise.
func (o *ServiceDefinitionV1Info) GetDisplayName() string {
	if o == nil || o.DisplayName == nil {
		var ret string
		return ret
	}
	return *o.DisplayName
}

// GetDisplayNameOk returns a tuple with the DisplayName field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ServiceDefinitionV1Info) GetDisplayNameOk() (*string, bool) {
	if o == nil || o.DisplayName == nil {
		return nil, false
	}
	return o.DisplayName, true
}

// HasDisplayName returns a boolean if a field has been set.
func (o *ServiceDefinitionV1Info) HasDisplayName() bool {
	return o != nil && o.DisplayName != nil
}

// SetDisplayName gets a reference to the given string and assigns it to the DisplayName field.
func (o *ServiceDefinitionV1Info) SetDisplayName(v string) {
	o.DisplayName = &v
}

// GetServiceTier returns the ServiceTier field value if set, zero value otherwise.
func (o *ServiceDefinitionV1Info) GetServiceTier() string {
	if o == nil || o.ServiceTier == nil {
		var ret string
		return ret
	}
	return *o.ServiceTier
}

// GetServiceTierOk returns a tuple with the ServiceTier field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ServiceDefinitionV1Info) GetServiceTierOk() (*string, bool) {
	if o == nil || o.ServiceTier == nil {
		return nil, false
	}
	return o.ServiceTier, true
}

// HasServiceTier returns a boolean if a field has been set.
func (o *ServiceDefinitionV1Info) HasServiceTier() bool {
	return o != nil && o.ServiceTier != nil
}

// SetServiceTier gets a reference to the given string and assigns it to the ServiceTier field.
func (o *ServiceDefinitionV1Info) SetServiceTier(v string) {
	o.ServiceTier = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o ServiceDefinitionV1Info) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["dd-service"] = o.DdService
	if o.Description != nil {
		toSerialize["description"] = o.Description
	}
	if o.DisplayName != nil {
		toSerialize["display-name"] = o.DisplayName
	}
	if o.ServiceTier != nil {
		toSerialize["service-tier"] = o.ServiceTier
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *ServiceDefinitionV1Info) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		DdService *string `json:"dd-service"`
	}{}
	all := struct {
		DdService   string  `json:"dd-service"`
		Description *string `json:"description,omitempty"`
		DisplayName *string `json:"display-name,omitempty"`
		ServiceTier *string `json:"service-tier,omitempty"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.DdService == nil {
		return fmt.Errorf("required field dd-service missing")
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
	o.DdService = all.DdService
	o.Description = all.Description
	o.DisplayName = all.DisplayName
	o.ServiceTier = all.ServiceTier
	return nil
}
