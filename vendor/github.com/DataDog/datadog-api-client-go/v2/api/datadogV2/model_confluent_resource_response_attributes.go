// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// ConfluentResourceResponseAttributes Model representation of a Confluent Cloud resource.
type ConfluentResourceResponseAttributes struct {
	// The resource type of the Resource. Can be `kafka`, `connector`, `ksql`, or `schema_registry`.
	ResourceType string `json:"resource_type"`
	// A list of strings representing tags. Can be a single key, or key-value pairs separated by a colon.
	Tags []string `json:"tags,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewConfluentResourceResponseAttributes instantiates a new ConfluentResourceResponseAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewConfluentResourceResponseAttributes(resourceType string) *ConfluentResourceResponseAttributes {
	this := ConfluentResourceResponseAttributes{}
	this.ResourceType = resourceType
	return &this
}

// NewConfluentResourceResponseAttributesWithDefaults instantiates a new ConfluentResourceResponseAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewConfluentResourceResponseAttributesWithDefaults() *ConfluentResourceResponseAttributes {
	this := ConfluentResourceResponseAttributes{}
	return &this
}

// GetResourceType returns the ResourceType field value.
func (o *ConfluentResourceResponseAttributes) GetResourceType() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.ResourceType
}

// GetResourceTypeOk returns a tuple with the ResourceType field value
// and a boolean to check if the value has been set.
func (o *ConfluentResourceResponseAttributes) GetResourceTypeOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.ResourceType, true
}

// SetResourceType sets field value.
func (o *ConfluentResourceResponseAttributes) SetResourceType(v string) {
	o.ResourceType = v
}

// GetTags returns the Tags field value if set, zero value otherwise.
func (o *ConfluentResourceResponseAttributes) GetTags() []string {
	if o == nil || o.Tags == nil {
		var ret []string
		return ret
	}
	return o.Tags
}

// GetTagsOk returns a tuple with the Tags field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ConfluentResourceResponseAttributes) GetTagsOk() (*[]string, bool) {
	if o == nil || o.Tags == nil {
		return nil, false
	}
	return &o.Tags, true
}

// HasTags returns a boolean if a field has been set.
func (o *ConfluentResourceResponseAttributes) HasTags() bool {
	return o != nil && o.Tags != nil
}

// SetTags gets a reference to the given []string and assigns it to the Tags field.
func (o *ConfluentResourceResponseAttributes) SetTags(v []string) {
	o.Tags = v
}

// MarshalJSON serializes the struct using spec logic.
func (o ConfluentResourceResponseAttributes) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["resource_type"] = o.ResourceType
	if o.Tags != nil {
		toSerialize["tags"] = o.Tags
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *ConfluentResourceResponseAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		ResourceType *string `json:"resource_type"`
	}{}
	all := struct {
		ResourceType string   `json:"resource_type"`
		Tags         []string `json:"tags,omitempty"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.ResourceType == nil {
		return fmt.Errorf("required field resource_type missing")
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
	o.ResourceType = all.ResourceType
	o.Tags = all.Tags
	return nil
}
