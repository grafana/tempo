// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV1

import (
	"encoding/json"
	"fmt"
)

// RunWorkflowWidgetInput Object to map a dashboard template variable to a workflow input.
type RunWorkflowWidgetInput struct {
	// Name of the workflow input.
	Name string `json:"name"`
	// Dashboard template variable. Can be suffixed with '.value' or '.key'.
	Value string `json:"value"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewRunWorkflowWidgetInput instantiates a new RunWorkflowWidgetInput object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewRunWorkflowWidgetInput(name string, value string) *RunWorkflowWidgetInput {
	this := RunWorkflowWidgetInput{}
	this.Name = name
	this.Value = value
	return &this
}

// NewRunWorkflowWidgetInputWithDefaults instantiates a new RunWorkflowWidgetInput object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewRunWorkflowWidgetInputWithDefaults() *RunWorkflowWidgetInput {
	this := RunWorkflowWidgetInput{}
	return &this
}

// GetName returns the Name field value.
func (o *RunWorkflowWidgetInput) GetName() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Name
}

// GetNameOk returns a tuple with the Name field value
// and a boolean to check if the value has been set.
func (o *RunWorkflowWidgetInput) GetNameOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Name, true
}

// SetName sets field value.
func (o *RunWorkflowWidgetInput) SetName(v string) {
	o.Name = v
}

// GetValue returns the Value field value.
func (o *RunWorkflowWidgetInput) GetValue() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Value
}

// GetValueOk returns a tuple with the Value field value
// and a boolean to check if the value has been set.
func (o *RunWorkflowWidgetInput) GetValueOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Value, true
}

// SetValue sets field value.
func (o *RunWorkflowWidgetInput) SetValue(v string) {
	o.Value = v
}

// MarshalJSON serializes the struct using spec logic.
func (o RunWorkflowWidgetInput) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["name"] = o.Name
	toSerialize["value"] = o.Value

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *RunWorkflowWidgetInput) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Name  *string `json:"name"`
		Value *string `json:"value"`
	}{}
	all := struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Name == nil {
		return fmt.Errorf("required field name missing")
	}
	if required.Value == nil {
		return fmt.Errorf("required field value missing")
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
	o.Value = all.Value
	return nil
}
