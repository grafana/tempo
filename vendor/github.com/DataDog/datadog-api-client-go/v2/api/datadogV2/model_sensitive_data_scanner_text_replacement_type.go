// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// SensitiveDataScannerTextReplacementType Type of the replacement text. None means no replacement.
// hash means the data will be stubbed. replacement_string means that
// one can chose a text to replace the data. partial_replacement_from_beginning
// allows a user to partially replace the data from the beginning, and
// partial_replacement_from_end on the other hand, allows to replace data from
// the end.
type SensitiveDataScannerTextReplacementType string

// List of SensitiveDataScannerTextReplacementType.
const (
	SENSITIVEDATASCANNERTEXTREPLACEMENTTYPE_NONE                               SensitiveDataScannerTextReplacementType = "none"
	SENSITIVEDATASCANNERTEXTREPLACEMENTTYPE_HASH                               SensitiveDataScannerTextReplacementType = "hash"
	SENSITIVEDATASCANNERTEXTREPLACEMENTTYPE_REPLACEMENT_STRING                 SensitiveDataScannerTextReplacementType = "replacement_string"
	SENSITIVEDATASCANNERTEXTREPLACEMENTTYPE_PARTIAL_REPLACEMENT_FROM_BEGINNING SensitiveDataScannerTextReplacementType = "partial_replacement_from_beginning"
	SENSITIVEDATASCANNERTEXTREPLACEMENTTYPE_PARTIAL_REPLACEMENT_FROM_END       SensitiveDataScannerTextReplacementType = "partial_replacement_from_end"
)

var allowedSensitiveDataScannerTextReplacementTypeEnumValues = []SensitiveDataScannerTextReplacementType{
	SENSITIVEDATASCANNERTEXTREPLACEMENTTYPE_NONE,
	SENSITIVEDATASCANNERTEXTREPLACEMENTTYPE_HASH,
	SENSITIVEDATASCANNERTEXTREPLACEMENTTYPE_REPLACEMENT_STRING,
	SENSITIVEDATASCANNERTEXTREPLACEMENTTYPE_PARTIAL_REPLACEMENT_FROM_BEGINNING,
	SENSITIVEDATASCANNERTEXTREPLACEMENTTYPE_PARTIAL_REPLACEMENT_FROM_END,
}

// GetAllowedValues reeturns the list of possible values.
func (v *SensitiveDataScannerTextReplacementType) GetAllowedValues() []SensitiveDataScannerTextReplacementType {
	return allowedSensitiveDataScannerTextReplacementTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *SensitiveDataScannerTextReplacementType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = SensitiveDataScannerTextReplacementType(value)
	return nil
}

// NewSensitiveDataScannerTextReplacementTypeFromValue returns a pointer to a valid SensitiveDataScannerTextReplacementType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewSensitiveDataScannerTextReplacementTypeFromValue(v string) (*SensitiveDataScannerTextReplacementType, error) {
	ev := SensitiveDataScannerTextReplacementType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for SensitiveDataScannerTextReplacementType: valid values are %v", v, allowedSensitiveDataScannerTextReplacementTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v SensitiveDataScannerTextReplacementType) IsValid() bool {
	for _, existing := range allowedSensitiveDataScannerTextReplacementTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to SensitiveDataScannerTextReplacementType value.
func (v SensitiveDataScannerTextReplacementType) Ptr() *SensitiveDataScannerTextReplacementType {
	return &v
}

// NullableSensitiveDataScannerTextReplacementType handles when a null is used for SensitiveDataScannerTextReplacementType.
type NullableSensitiveDataScannerTextReplacementType struct {
	value *SensitiveDataScannerTextReplacementType
	isSet bool
}

// Get returns the associated value.
func (v NullableSensitiveDataScannerTextReplacementType) Get() *SensitiveDataScannerTextReplacementType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableSensitiveDataScannerTextReplacementType) Set(val *SensitiveDataScannerTextReplacementType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableSensitiveDataScannerTextReplacementType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableSensitiveDataScannerTextReplacementType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableSensitiveDataScannerTextReplacementType initializes the struct as if Set has been called.
func NewNullableSensitiveDataScannerTextReplacementType(val *SensitiveDataScannerTextReplacementType) *NullableSensitiveDataScannerTextReplacementType {
	return &NullableSensitiveDataScannerTextReplacementType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableSensitiveDataScannerTextReplacementType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableSensitiveDataScannerTextReplacementType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
