// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV1

import (
	"encoding/json"
	"fmt"
)

// RunWorkflowWidgetDefinitionType Type of the run workflow widget.
type RunWorkflowWidgetDefinitionType string

// List of RunWorkflowWidgetDefinitionType.
const (
	RUNWORKFLOWWIDGETDEFINITIONTYPE_RUN_WORKFLOW RunWorkflowWidgetDefinitionType = "run_workflow"
)

var allowedRunWorkflowWidgetDefinitionTypeEnumValues = []RunWorkflowWidgetDefinitionType{
	RUNWORKFLOWWIDGETDEFINITIONTYPE_RUN_WORKFLOW,
}

// GetAllowedValues reeturns the list of possible values.
func (v *RunWorkflowWidgetDefinitionType) GetAllowedValues() []RunWorkflowWidgetDefinitionType {
	return allowedRunWorkflowWidgetDefinitionTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *RunWorkflowWidgetDefinitionType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = RunWorkflowWidgetDefinitionType(value)
	return nil
}

// NewRunWorkflowWidgetDefinitionTypeFromValue returns a pointer to a valid RunWorkflowWidgetDefinitionType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewRunWorkflowWidgetDefinitionTypeFromValue(v string) (*RunWorkflowWidgetDefinitionType, error) {
	ev := RunWorkflowWidgetDefinitionType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for RunWorkflowWidgetDefinitionType: valid values are %v", v, allowedRunWorkflowWidgetDefinitionTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v RunWorkflowWidgetDefinitionType) IsValid() bool {
	for _, existing := range allowedRunWorkflowWidgetDefinitionTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to RunWorkflowWidgetDefinitionType value.
func (v RunWorkflowWidgetDefinitionType) Ptr() *RunWorkflowWidgetDefinitionType {
	return &v
}

// NullableRunWorkflowWidgetDefinitionType handles when a null is used for RunWorkflowWidgetDefinitionType.
type NullableRunWorkflowWidgetDefinitionType struct {
	value *RunWorkflowWidgetDefinitionType
	isSet bool
}

// Get returns the associated value.
func (v NullableRunWorkflowWidgetDefinitionType) Get() *RunWorkflowWidgetDefinitionType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableRunWorkflowWidgetDefinitionType) Set(val *RunWorkflowWidgetDefinitionType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableRunWorkflowWidgetDefinitionType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableRunWorkflowWidgetDefinitionType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableRunWorkflowWidgetDefinitionType initializes the struct as if Set has been called.
func NewNullableRunWorkflowWidgetDefinitionType(val *RunWorkflowWidgetDefinitionType) *NullableRunWorkflowWidgetDefinitionType {
	return &NullableRunWorkflowWidgetDefinitionType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableRunWorkflowWidgetDefinitionType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableRunWorkflowWidgetDefinitionType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
