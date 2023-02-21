// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// SensitiveDataScannerGroupType Sensitive Data Scanner group type.
type SensitiveDataScannerGroupType string

// List of SensitiveDataScannerGroupType.
const (
	SENSITIVEDATASCANNERGROUPTYPE_SENSITIVE_DATA_SCANNER_GROUP SensitiveDataScannerGroupType = "sensitive_data_scanner_group"
)

var allowedSensitiveDataScannerGroupTypeEnumValues = []SensitiveDataScannerGroupType{
	SENSITIVEDATASCANNERGROUPTYPE_SENSITIVE_DATA_SCANNER_GROUP,
}

// GetAllowedValues reeturns the list of possible values.
func (v *SensitiveDataScannerGroupType) GetAllowedValues() []SensitiveDataScannerGroupType {
	return allowedSensitiveDataScannerGroupTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *SensitiveDataScannerGroupType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = SensitiveDataScannerGroupType(value)
	return nil
}

// NewSensitiveDataScannerGroupTypeFromValue returns a pointer to a valid SensitiveDataScannerGroupType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewSensitiveDataScannerGroupTypeFromValue(v string) (*SensitiveDataScannerGroupType, error) {
	ev := SensitiveDataScannerGroupType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for SensitiveDataScannerGroupType: valid values are %v", v, allowedSensitiveDataScannerGroupTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v SensitiveDataScannerGroupType) IsValid() bool {
	for _, existing := range allowedSensitiveDataScannerGroupTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to SensitiveDataScannerGroupType value.
func (v SensitiveDataScannerGroupType) Ptr() *SensitiveDataScannerGroupType {
	return &v
}

// NullableSensitiveDataScannerGroupType handles when a null is used for SensitiveDataScannerGroupType.
type NullableSensitiveDataScannerGroupType struct {
	value *SensitiveDataScannerGroupType
	isSet bool
}

// Get returns the associated value.
func (v NullableSensitiveDataScannerGroupType) Get() *SensitiveDataScannerGroupType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableSensitiveDataScannerGroupType) Set(val *SensitiveDataScannerGroupType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableSensitiveDataScannerGroupType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableSensitiveDataScannerGroupType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableSensitiveDataScannerGroupType initializes the struct as if Set has been called.
func NewNullableSensitiveDataScannerGroupType(val *SensitiveDataScannerGroupType) *NullableSensitiveDataScannerGroupType {
	return &NullableSensitiveDataScannerGroupType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableSensitiveDataScannerGroupType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableSensitiveDataScannerGroupType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
