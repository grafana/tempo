// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// IncidentTimelineCellMarkdownContentType Type of the Markdown timeline cell.
type IncidentTimelineCellMarkdownContentType string

// List of IncidentTimelineCellMarkdownContentType.
const (
	INCIDENTTIMELINECELLMARKDOWNCONTENTTYPE_MARKDOWN IncidentTimelineCellMarkdownContentType = "markdown"
)

var allowedIncidentTimelineCellMarkdownContentTypeEnumValues = []IncidentTimelineCellMarkdownContentType{
	INCIDENTTIMELINECELLMARKDOWNCONTENTTYPE_MARKDOWN,
}

// GetAllowedValues reeturns the list of possible values.
func (v *IncidentTimelineCellMarkdownContentType) GetAllowedValues() []IncidentTimelineCellMarkdownContentType {
	return allowedIncidentTimelineCellMarkdownContentTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *IncidentTimelineCellMarkdownContentType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = IncidentTimelineCellMarkdownContentType(value)
	return nil
}

// NewIncidentTimelineCellMarkdownContentTypeFromValue returns a pointer to a valid IncidentTimelineCellMarkdownContentType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewIncidentTimelineCellMarkdownContentTypeFromValue(v string) (*IncidentTimelineCellMarkdownContentType, error) {
	ev := IncidentTimelineCellMarkdownContentType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for IncidentTimelineCellMarkdownContentType: valid values are %v", v, allowedIncidentTimelineCellMarkdownContentTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v IncidentTimelineCellMarkdownContentType) IsValid() bool {
	for _, existing := range allowedIncidentTimelineCellMarkdownContentTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to IncidentTimelineCellMarkdownContentType value.
func (v IncidentTimelineCellMarkdownContentType) Ptr() *IncidentTimelineCellMarkdownContentType {
	return &v
}

// NullableIncidentTimelineCellMarkdownContentType handles when a null is used for IncidentTimelineCellMarkdownContentType.
type NullableIncidentTimelineCellMarkdownContentType struct {
	value *IncidentTimelineCellMarkdownContentType
	isSet bool
}

// Get returns the associated value.
func (v NullableIncidentTimelineCellMarkdownContentType) Get() *IncidentTimelineCellMarkdownContentType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableIncidentTimelineCellMarkdownContentType) Set(val *IncidentTimelineCellMarkdownContentType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableIncidentTimelineCellMarkdownContentType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableIncidentTimelineCellMarkdownContentType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableIncidentTimelineCellMarkdownContentType initializes the struct as if Set has been called.
func NewNullableIncidentTimelineCellMarkdownContentType(val *IncidentTimelineCellMarkdownContentType) *NullableIncidentTimelineCellMarkdownContentType {
	return &NullableIncidentTimelineCellMarkdownContentType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableIncidentTimelineCellMarkdownContentType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableIncidentTimelineCellMarkdownContentType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
