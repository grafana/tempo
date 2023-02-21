// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// ServiceDefinitionDataAttributes Service definition attributes.
type ServiceDefinitionDataAttributes struct {
	// Metadata about a service definition.
	Meta *ServiceDefinitionMeta `json:"meta,omitempty"`
	// Service definition schema.
	Schema *ServiceDefinitionSchema `json:"schema,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewServiceDefinitionDataAttributes instantiates a new ServiceDefinitionDataAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewServiceDefinitionDataAttributes() *ServiceDefinitionDataAttributes {
	this := ServiceDefinitionDataAttributes{}
	return &this
}

// NewServiceDefinitionDataAttributesWithDefaults instantiates a new ServiceDefinitionDataAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewServiceDefinitionDataAttributesWithDefaults() *ServiceDefinitionDataAttributes {
	this := ServiceDefinitionDataAttributes{}
	return &this
}

// GetMeta returns the Meta field value if set, zero value otherwise.
func (o *ServiceDefinitionDataAttributes) GetMeta() ServiceDefinitionMeta {
	if o == nil || o.Meta == nil {
		var ret ServiceDefinitionMeta
		return ret
	}
	return *o.Meta
}

// GetMetaOk returns a tuple with the Meta field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ServiceDefinitionDataAttributes) GetMetaOk() (*ServiceDefinitionMeta, bool) {
	if o == nil || o.Meta == nil {
		return nil, false
	}
	return o.Meta, true
}

// HasMeta returns a boolean if a field has been set.
func (o *ServiceDefinitionDataAttributes) HasMeta() bool {
	return o != nil && o.Meta != nil
}

// SetMeta gets a reference to the given ServiceDefinitionMeta and assigns it to the Meta field.
func (o *ServiceDefinitionDataAttributes) SetMeta(v ServiceDefinitionMeta) {
	o.Meta = &v
}

// GetSchema returns the Schema field value if set, zero value otherwise.
func (o *ServiceDefinitionDataAttributes) GetSchema() ServiceDefinitionSchema {
	if o == nil || o.Schema == nil {
		var ret ServiceDefinitionSchema
		return ret
	}
	return *o.Schema
}

// GetSchemaOk returns a tuple with the Schema field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ServiceDefinitionDataAttributes) GetSchemaOk() (*ServiceDefinitionSchema, bool) {
	if o == nil || o.Schema == nil {
		return nil, false
	}
	return o.Schema, true
}

// HasSchema returns a boolean if a field has been set.
func (o *ServiceDefinitionDataAttributes) HasSchema() bool {
	return o != nil && o.Schema != nil
}

// SetSchema gets a reference to the given ServiceDefinitionSchema and assigns it to the Schema field.
func (o *ServiceDefinitionDataAttributes) SetSchema(v ServiceDefinitionSchema) {
	o.Schema = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o ServiceDefinitionDataAttributes) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Meta != nil {
		toSerialize["meta"] = o.Meta
	}
	if o.Schema != nil {
		toSerialize["schema"] = o.Schema
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *ServiceDefinitionDataAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Meta   *ServiceDefinitionMeta   `json:"meta,omitempty"`
		Schema *ServiceDefinitionSchema `json:"schema,omitempty"`
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
	if all.Meta != nil && all.Meta.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Meta = all.Meta
	o.Schema = all.Schema
	return nil
}
