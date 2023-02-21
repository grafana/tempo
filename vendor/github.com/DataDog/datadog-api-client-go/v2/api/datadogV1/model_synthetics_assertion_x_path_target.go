// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV1

import (
	"encoding/json"
	"fmt"
)

// SyntheticsAssertionXPathTarget An assertion for the `validatesXPath` operator.
type SyntheticsAssertionXPathTarget struct {
	// Assertion operator to apply.
	Operator SyntheticsAssertionXPathOperator `json:"operator"`
	// The associated assertion property.
	Property *string `json:"property,omitempty"`
	// Composed target for `validatesXPath` operator.
	Target *SyntheticsAssertionXPathTargetTarget `json:"target,omitempty"`
	// Type of the assertion.
	Type SyntheticsAssertionType `json:"type"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewSyntheticsAssertionXPathTarget instantiates a new SyntheticsAssertionXPathTarget object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSyntheticsAssertionXPathTarget(operator SyntheticsAssertionXPathOperator, typeVar SyntheticsAssertionType) *SyntheticsAssertionXPathTarget {
	this := SyntheticsAssertionXPathTarget{}
	this.Operator = operator
	this.Type = typeVar
	return &this
}

// NewSyntheticsAssertionXPathTargetWithDefaults instantiates a new SyntheticsAssertionXPathTarget object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSyntheticsAssertionXPathTargetWithDefaults() *SyntheticsAssertionXPathTarget {
	this := SyntheticsAssertionXPathTarget{}
	return &this
}

// GetOperator returns the Operator field value.
func (o *SyntheticsAssertionXPathTarget) GetOperator() SyntheticsAssertionXPathOperator {
	if o == nil {
		var ret SyntheticsAssertionXPathOperator
		return ret
	}
	return o.Operator
}

// GetOperatorOk returns a tuple with the Operator field value
// and a boolean to check if the value has been set.
func (o *SyntheticsAssertionXPathTarget) GetOperatorOk() (*SyntheticsAssertionXPathOperator, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Operator, true
}

// SetOperator sets field value.
func (o *SyntheticsAssertionXPathTarget) SetOperator(v SyntheticsAssertionXPathOperator) {
	o.Operator = v
}

// GetProperty returns the Property field value if set, zero value otherwise.
func (o *SyntheticsAssertionXPathTarget) GetProperty() string {
	if o == nil || o.Property == nil {
		var ret string
		return ret
	}
	return *o.Property
}

// GetPropertyOk returns a tuple with the Property field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsAssertionXPathTarget) GetPropertyOk() (*string, bool) {
	if o == nil || o.Property == nil {
		return nil, false
	}
	return o.Property, true
}

// HasProperty returns a boolean if a field has been set.
func (o *SyntheticsAssertionXPathTarget) HasProperty() bool {
	return o != nil && o.Property != nil
}

// SetProperty gets a reference to the given string and assigns it to the Property field.
func (o *SyntheticsAssertionXPathTarget) SetProperty(v string) {
	o.Property = &v
}

// GetTarget returns the Target field value if set, zero value otherwise.
func (o *SyntheticsAssertionXPathTarget) GetTarget() SyntheticsAssertionXPathTargetTarget {
	if o == nil || o.Target == nil {
		var ret SyntheticsAssertionXPathTargetTarget
		return ret
	}
	return *o.Target
}

// GetTargetOk returns a tuple with the Target field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsAssertionXPathTarget) GetTargetOk() (*SyntheticsAssertionXPathTargetTarget, bool) {
	if o == nil || o.Target == nil {
		return nil, false
	}
	return o.Target, true
}

// HasTarget returns a boolean if a field has been set.
func (o *SyntheticsAssertionXPathTarget) HasTarget() bool {
	return o != nil && o.Target != nil
}

// SetTarget gets a reference to the given SyntheticsAssertionXPathTargetTarget and assigns it to the Target field.
func (o *SyntheticsAssertionXPathTarget) SetTarget(v SyntheticsAssertionXPathTargetTarget) {
	o.Target = &v
}

// GetType returns the Type field value.
func (o *SyntheticsAssertionXPathTarget) GetType() SyntheticsAssertionType {
	if o == nil {
		var ret SyntheticsAssertionType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *SyntheticsAssertionXPathTarget) GetTypeOk() (*SyntheticsAssertionType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *SyntheticsAssertionXPathTarget) SetType(v SyntheticsAssertionType) {
	o.Type = v
}

// MarshalJSON serializes the struct using spec logic.
func (o SyntheticsAssertionXPathTarget) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["operator"] = o.Operator
	if o.Property != nil {
		toSerialize["property"] = o.Property
	}
	if o.Target != nil {
		toSerialize["target"] = o.Target
	}
	toSerialize["type"] = o.Type

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SyntheticsAssertionXPathTarget) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Operator *SyntheticsAssertionXPathOperator `json:"operator"`
		Type     *SyntheticsAssertionType          `json:"type"`
	}{}
	all := struct {
		Operator SyntheticsAssertionXPathOperator      `json:"operator"`
		Property *string                               `json:"property,omitempty"`
		Target   *SyntheticsAssertionXPathTargetTarget `json:"target,omitempty"`
		Type     SyntheticsAssertionType               `json:"type"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Operator == nil {
		return fmt.Errorf("required field operator missing")
	}
	if required.Type == nil {
		return fmt.Errorf("required field type missing")
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
	if v := all.Operator; !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	if v := all.Type; !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.Operator = all.Operator
	o.Property = all.Property
	if all.Target != nil && all.Target.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Target = all.Target
	o.Type = all.Type
	return nil
}
