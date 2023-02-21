// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// ConfluentResourceRequestAttributes Attributes object for updating a Confluent resource.
type ConfluentResourceRequestAttributes struct {
	// The resource type of the Resource. Can be `kafka`, `connector`, `ksql`, or `schema_registry`.
	ResourceType *string `json:"resource_type,omitempty"`
	// A list of strings representing tags. Can be a single key, or key-value pairs separated by a colon.
	Tags []string `json:"tags,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewConfluentResourceRequestAttributes instantiates a new ConfluentResourceRequestAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewConfluentResourceRequestAttributes() *ConfluentResourceRequestAttributes {
	this := ConfluentResourceRequestAttributes{}
	return &this
}

// NewConfluentResourceRequestAttributesWithDefaults instantiates a new ConfluentResourceRequestAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewConfluentResourceRequestAttributesWithDefaults() *ConfluentResourceRequestAttributes {
	this := ConfluentResourceRequestAttributes{}
	return &this
}

// GetResourceType returns the ResourceType field value if set, zero value otherwise.
func (o *ConfluentResourceRequestAttributes) GetResourceType() string {
	if o == nil || o.ResourceType == nil {
		var ret string
		return ret
	}
	return *o.ResourceType
}

// GetResourceTypeOk returns a tuple with the ResourceType field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ConfluentResourceRequestAttributes) GetResourceTypeOk() (*string, bool) {
	if o == nil || o.ResourceType == nil {
		return nil, false
	}
	return o.ResourceType, true
}

// HasResourceType returns a boolean if a field has been set.
func (o *ConfluentResourceRequestAttributes) HasResourceType() bool {
	return o != nil && o.ResourceType != nil
}

// SetResourceType gets a reference to the given string and assigns it to the ResourceType field.
func (o *ConfluentResourceRequestAttributes) SetResourceType(v string) {
	o.ResourceType = &v
}

// GetTags returns the Tags field value if set, zero value otherwise.
func (o *ConfluentResourceRequestAttributes) GetTags() []string {
	if o == nil || o.Tags == nil {
		var ret []string
		return ret
	}
	return o.Tags
}

// GetTagsOk returns a tuple with the Tags field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ConfluentResourceRequestAttributes) GetTagsOk() (*[]string, bool) {
	if o == nil || o.Tags == nil {
		return nil, false
	}
	return &o.Tags, true
}

// HasTags returns a boolean if a field has been set.
func (o *ConfluentResourceRequestAttributes) HasTags() bool {
	return o != nil && o.Tags != nil
}

// SetTags gets a reference to the given []string and assigns it to the Tags field.
func (o *ConfluentResourceRequestAttributes) SetTags(v []string) {
	o.Tags = v
}

// MarshalJSON serializes the struct using spec logic.
func (o ConfluentResourceRequestAttributes) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.ResourceType != nil {
		toSerialize["resource_type"] = o.ResourceType
	}
	if o.Tags != nil {
		toSerialize["tags"] = o.Tags
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *ConfluentResourceRequestAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		ResourceType *string  `json:"resource_type,omitempty"`
		Tags         []string `json:"tags,omitempty"`
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
	o.ResourceType = all.ResourceType
	o.Tags = all.Tags
	return nil
}
