// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// SensitiveDataScannerStandardPatternAttributes Attributes of the Sensitive Data Scanner standard pattern.
type SensitiveDataScannerStandardPatternAttributes struct {
	// Name of the standard pattern.
	Name *string `json:"name,omitempty"`
	// Regex to match.
	Pattern *string `json:"pattern,omitempty"`
	// List of tags.
	Tags []string `json:"tags,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewSensitiveDataScannerStandardPatternAttributes instantiates a new SensitiveDataScannerStandardPatternAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSensitiveDataScannerStandardPatternAttributes() *SensitiveDataScannerStandardPatternAttributes {
	this := SensitiveDataScannerStandardPatternAttributes{}
	return &this
}

// NewSensitiveDataScannerStandardPatternAttributesWithDefaults instantiates a new SensitiveDataScannerStandardPatternAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSensitiveDataScannerStandardPatternAttributesWithDefaults() *SensitiveDataScannerStandardPatternAttributes {
	this := SensitiveDataScannerStandardPatternAttributes{}
	return &this
}

// GetName returns the Name field value if set, zero value otherwise.
func (o *SensitiveDataScannerStandardPatternAttributes) GetName() string {
	if o == nil || o.Name == nil {
		var ret string
		return ret
	}
	return *o.Name
}

// GetNameOk returns a tuple with the Name field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SensitiveDataScannerStandardPatternAttributes) GetNameOk() (*string, bool) {
	if o == nil || o.Name == nil {
		return nil, false
	}
	return o.Name, true
}

// HasName returns a boolean if a field has been set.
func (o *SensitiveDataScannerStandardPatternAttributes) HasName() bool {
	return o != nil && o.Name != nil
}

// SetName gets a reference to the given string and assigns it to the Name field.
func (o *SensitiveDataScannerStandardPatternAttributes) SetName(v string) {
	o.Name = &v
}

// GetPattern returns the Pattern field value if set, zero value otherwise.
func (o *SensitiveDataScannerStandardPatternAttributes) GetPattern() string {
	if o == nil || o.Pattern == nil {
		var ret string
		return ret
	}
	return *o.Pattern
}

// GetPatternOk returns a tuple with the Pattern field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SensitiveDataScannerStandardPatternAttributes) GetPatternOk() (*string, bool) {
	if o == nil || o.Pattern == nil {
		return nil, false
	}
	return o.Pattern, true
}

// HasPattern returns a boolean if a field has been set.
func (o *SensitiveDataScannerStandardPatternAttributes) HasPattern() bool {
	return o != nil && o.Pattern != nil
}

// SetPattern gets a reference to the given string and assigns it to the Pattern field.
func (o *SensitiveDataScannerStandardPatternAttributes) SetPattern(v string) {
	o.Pattern = &v
}

// GetTags returns the Tags field value if set, zero value otherwise.
func (o *SensitiveDataScannerStandardPatternAttributes) GetTags() []string {
	if o == nil || o.Tags == nil {
		var ret []string
		return ret
	}
	return o.Tags
}

// GetTagsOk returns a tuple with the Tags field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SensitiveDataScannerStandardPatternAttributes) GetTagsOk() (*[]string, bool) {
	if o == nil || o.Tags == nil {
		return nil, false
	}
	return &o.Tags, true
}

// HasTags returns a boolean if a field has been set.
func (o *SensitiveDataScannerStandardPatternAttributes) HasTags() bool {
	return o != nil && o.Tags != nil
}

// SetTags gets a reference to the given []string and assigns it to the Tags field.
func (o *SensitiveDataScannerStandardPatternAttributes) SetTags(v []string) {
	o.Tags = v
}

// MarshalJSON serializes the struct using spec logic.
func (o SensitiveDataScannerStandardPatternAttributes) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Name != nil {
		toSerialize["name"] = o.Name
	}
	if o.Pattern != nil {
		toSerialize["pattern"] = o.Pattern
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
func (o *SensitiveDataScannerStandardPatternAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Name    *string  `json:"name,omitempty"`
		Pattern *string  `json:"pattern,omitempty"`
		Tags    []string `json:"tags,omitempty"`
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
	o.Name = all.Name
	o.Pattern = all.Pattern
	o.Tags = all.Tags
	return nil
}
