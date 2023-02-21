// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// FastlyAccounResponseAttributes Attributes object of a Fastly account.
type FastlyAccounResponseAttributes struct {
	// The name of the Fastly account.
	Name string `json:"name"`
	// A list of services belonging to the parent account.
	Services []FastlyService `json:"services,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewFastlyAccounResponseAttributes instantiates a new FastlyAccounResponseAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewFastlyAccounResponseAttributes(name string) *FastlyAccounResponseAttributes {
	this := FastlyAccounResponseAttributes{}
	this.Name = name
	return &this
}

// NewFastlyAccounResponseAttributesWithDefaults instantiates a new FastlyAccounResponseAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewFastlyAccounResponseAttributesWithDefaults() *FastlyAccounResponseAttributes {
	this := FastlyAccounResponseAttributes{}
	return &this
}

// GetName returns the Name field value.
func (o *FastlyAccounResponseAttributes) GetName() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Name
}

// GetNameOk returns a tuple with the Name field value
// and a boolean to check if the value has been set.
func (o *FastlyAccounResponseAttributes) GetNameOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Name, true
}

// SetName sets field value.
func (o *FastlyAccounResponseAttributes) SetName(v string) {
	o.Name = v
}

// GetServices returns the Services field value if set, zero value otherwise.
func (o *FastlyAccounResponseAttributes) GetServices() []FastlyService {
	if o == nil || o.Services == nil {
		var ret []FastlyService
		return ret
	}
	return o.Services
}

// GetServicesOk returns a tuple with the Services field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *FastlyAccounResponseAttributes) GetServicesOk() (*[]FastlyService, bool) {
	if o == nil || o.Services == nil {
		return nil, false
	}
	return &o.Services, true
}

// HasServices returns a boolean if a field has been set.
func (o *FastlyAccounResponseAttributes) HasServices() bool {
	return o != nil && o.Services != nil
}

// SetServices gets a reference to the given []FastlyService and assigns it to the Services field.
func (o *FastlyAccounResponseAttributes) SetServices(v []FastlyService) {
	o.Services = v
}

// MarshalJSON serializes the struct using spec logic.
func (o FastlyAccounResponseAttributes) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["name"] = o.Name
	if o.Services != nil {
		toSerialize["services"] = o.Services
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *FastlyAccounResponseAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Name *string `json:"name"`
	}{}
	all := struct {
		Name     string          `json:"name"`
		Services []FastlyService `json:"services,omitempty"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Name == nil {
		return fmt.Errorf("required field name missing")
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
	o.Name = all.Name
	o.Services = all.Services
	return nil
}
