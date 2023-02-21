// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// SensitiveDataScannerStandardPatternType Sensitive Data Scanner standard pattern type.
type SensitiveDataScannerStandardPatternType string

// List of SensitiveDataScannerStandardPatternType.
const (
	SENSITIVEDATASCANNERSTANDARDPATTERNTYPE_SENSITIVE_DATA_SCANNER_STANDARD_PATTERN SensitiveDataScannerStandardPatternType = "sensitive_data_scanner_standard_pattern"
)

var allowedSensitiveDataScannerStandardPatternTypeEnumValues = []SensitiveDataScannerStandardPatternType{
	SENSITIVEDATASCANNERSTANDARDPATTERNTYPE_SENSITIVE_DATA_SCANNER_STANDARD_PATTERN,
}

// GetAllowedValues reeturns the list of possible values.
func (v *SensitiveDataScannerStandardPatternType) GetAllowedValues() []SensitiveDataScannerStandardPatternType {
	return allowedSensitiveDataScannerStandardPatternTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *SensitiveDataScannerStandardPatternType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = SensitiveDataScannerStandardPatternType(value)
	return nil
}

// NewSensitiveDataScannerStandardPatternTypeFromValue returns a pointer to a valid SensitiveDataScannerStandardPatternType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewSensitiveDataScannerStandardPatternTypeFromValue(v string) (*SensitiveDataScannerStandardPatternType, error) {
	ev := SensitiveDataScannerStandardPatternType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for SensitiveDataScannerStandardPatternType: valid values are %v", v, allowedSensitiveDataScannerStandardPatternTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v SensitiveDataScannerStandardPatternType) IsValid() bool {
	for _, existing := range allowedSensitiveDataScannerStandardPatternTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to SensitiveDataScannerStandardPatternType value.
func (v SensitiveDataScannerStandardPatternType) Ptr() *SensitiveDataScannerStandardPatternType {
	return &v
}

// NullableSensitiveDataScannerStandardPatternType handles when a null is used for SensitiveDataScannerStandardPatternType.
type NullableSensitiveDataScannerStandardPatternType struct {
	value *SensitiveDataScannerStandardPatternType
	isSet bool
}

// Get returns the associated value.
func (v NullableSensitiveDataScannerStandardPatternType) Get() *SensitiveDataScannerStandardPatternType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableSensitiveDataScannerStandardPatternType) Set(val *SensitiveDataScannerStandardPatternType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableSensitiveDataScannerStandardPatternType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableSensitiveDataScannerStandardPatternType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableSensitiveDataScannerStandardPatternType initializes the struct as if Set has been called.
func NewNullableSensitiveDataScannerStandardPatternType(val *SensitiveDataScannerStandardPatternType) *NullableSensitiveDataScannerStandardPatternType {
	return &NullableSensitiveDataScannerStandardPatternType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableSensitiveDataScannerStandardPatternType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableSensitiveDataScannerStandardPatternType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
