// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// SensitiveDataScannerRuleType Sensitive Data Scanner rule type.
type SensitiveDataScannerRuleType string

// List of SensitiveDataScannerRuleType.
const (
	SENSITIVEDATASCANNERRULETYPE_SENSITIVE_DATA_SCANNER_RULE SensitiveDataScannerRuleType = "sensitive_data_scanner_rule"
)

var allowedSensitiveDataScannerRuleTypeEnumValues = []SensitiveDataScannerRuleType{
	SENSITIVEDATASCANNERRULETYPE_SENSITIVE_DATA_SCANNER_RULE,
}

// GetAllowedValues reeturns the list of possible values.
func (v *SensitiveDataScannerRuleType) GetAllowedValues() []SensitiveDataScannerRuleType {
	return allowedSensitiveDataScannerRuleTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *SensitiveDataScannerRuleType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = SensitiveDataScannerRuleType(value)
	return nil
}

// NewSensitiveDataScannerRuleTypeFromValue returns a pointer to a valid SensitiveDataScannerRuleType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewSensitiveDataScannerRuleTypeFromValue(v string) (*SensitiveDataScannerRuleType, error) {
	ev := SensitiveDataScannerRuleType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for SensitiveDataScannerRuleType: valid values are %v", v, allowedSensitiveDataScannerRuleTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v SensitiveDataScannerRuleType) IsValid() bool {
	for _, existing := range allowedSensitiveDataScannerRuleTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to SensitiveDataScannerRuleType value.
func (v SensitiveDataScannerRuleType) Ptr() *SensitiveDataScannerRuleType {
	return &v
}

// NullableSensitiveDataScannerRuleType handles when a null is used for SensitiveDataScannerRuleType.
type NullableSensitiveDataScannerRuleType struct {
	value *SensitiveDataScannerRuleType
	isSet bool
}

// Get returns the associated value.
func (v NullableSensitiveDataScannerRuleType) Get() *SensitiveDataScannerRuleType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableSensitiveDataScannerRuleType) Set(val *SensitiveDataScannerRuleType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableSensitiveDataScannerRuleType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableSensitiveDataScannerRuleType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableSensitiveDataScannerRuleType initializes the struct as if Set has been called.
func NewNullableSensitiveDataScannerRuleType(val *SensitiveDataScannerRuleType) *NullableSensitiveDataScannerRuleType {
	return &NullableSensitiveDataScannerRuleType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableSensitiveDataScannerRuleType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableSensitiveDataScannerRuleType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
