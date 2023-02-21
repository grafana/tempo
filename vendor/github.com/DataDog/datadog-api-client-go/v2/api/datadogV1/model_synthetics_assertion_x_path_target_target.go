// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV1

import (
	"encoding/json"
)

// SyntheticsAssertionXPathTargetTarget Composed target for `validatesXPath` operator.
type SyntheticsAssertionXPathTargetTarget struct {
	// The specific operator to use on the path.
	Operator *string `json:"operator,omitempty"`
	// The path target value to compare to.
	TargetValue interface{} `json:"targetValue,omitempty"`
	// The X path to assert.
	XPath *string `json:"xPath,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewSyntheticsAssertionXPathTargetTarget instantiates a new SyntheticsAssertionXPathTargetTarget object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSyntheticsAssertionXPathTargetTarget() *SyntheticsAssertionXPathTargetTarget {
	this := SyntheticsAssertionXPathTargetTarget{}
	return &this
}

// NewSyntheticsAssertionXPathTargetTargetWithDefaults instantiates a new SyntheticsAssertionXPathTargetTarget object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSyntheticsAssertionXPathTargetTargetWithDefaults() *SyntheticsAssertionXPathTargetTarget {
	this := SyntheticsAssertionXPathTargetTarget{}
	return &this
}

// GetOperator returns the Operator field value if set, zero value otherwise.
func (o *SyntheticsAssertionXPathTargetTarget) GetOperator() string {
	if o == nil || o.Operator == nil {
		var ret string
		return ret
	}
	return *o.Operator
}

// GetOperatorOk returns a tuple with the Operator field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsAssertionXPathTargetTarget) GetOperatorOk() (*string, bool) {
	if o == nil || o.Operator == nil {
		return nil, false
	}
	return o.Operator, true
}

// HasOperator returns a boolean if a field has been set.
func (o *SyntheticsAssertionXPathTargetTarget) HasOperator() bool {
	return o != nil && o.Operator != nil
}

// SetOperator gets a reference to the given string and assigns it to the Operator field.
func (o *SyntheticsAssertionXPathTargetTarget) SetOperator(v string) {
	o.Operator = &v
}

// GetTargetValue returns the TargetValue field value if set, zero value otherwise.
func (o *SyntheticsAssertionXPathTargetTarget) GetTargetValue() interface{} {
	if o == nil || o.TargetValue == nil {
		var ret interface{}
		return ret
	}
	return o.TargetValue
}

// GetTargetValueOk returns a tuple with the TargetValue field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsAssertionXPathTargetTarget) GetTargetValueOk() (*interface{}, bool) {
	if o == nil || o.TargetValue == nil {
		return nil, false
	}
	return &o.TargetValue, true
}

// HasTargetValue returns a boolean if a field has been set.
func (o *SyntheticsAssertionXPathTargetTarget) HasTargetValue() bool {
	return o != nil && o.TargetValue != nil
}

// SetTargetValue gets a reference to the given interface{} and assigns it to the TargetValue field.
func (o *SyntheticsAssertionXPathTargetTarget) SetTargetValue(v interface{}) {
	o.TargetValue = v
}

// GetXPath returns the XPath field value if set, zero value otherwise.
func (o *SyntheticsAssertionXPathTargetTarget) GetXPath() string {
	if o == nil || o.XPath == nil {
		var ret string
		return ret
	}
	return *o.XPath
}

// GetXPathOk returns a tuple with the XPath field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsAssertionXPathTargetTarget) GetXPathOk() (*string, bool) {
	if o == nil || o.XPath == nil {
		return nil, false
	}
	return o.XPath, true
}

// HasXPath returns a boolean if a field has been set.
func (o *SyntheticsAssertionXPathTargetTarget) HasXPath() bool {
	return o != nil && o.XPath != nil
}

// SetXPath gets a reference to the given string and assigns it to the XPath field.
func (o *SyntheticsAssertionXPathTargetTarget) SetXPath(v string) {
	o.XPath = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SyntheticsAssertionXPathTargetTarget) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Operator != nil {
		toSerialize["operator"] = o.Operator
	}
	if o.TargetValue != nil {
		toSerialize["targetValue"] = o.TargetValue
	}
	if o.XPath != nil {
		toSerialize["xPath"] = o.XPath
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SyntheticsAssertionXPathTargetTarget) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Operator    *string     `json:"operator,omitempty"`
		TargetValue interface{} `json:"targetValue,omitempty"`
		XPath       *string     `json:"xPath,omitempty"`
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
	o.Operator = all.Operator
	o.TargetValue = all.TargetValue
	o.XPath = all.XPath
	return nil
}
