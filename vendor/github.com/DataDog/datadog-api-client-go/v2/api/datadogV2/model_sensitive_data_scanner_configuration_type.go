// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// SensitiveDataScannerConfigurationType Sensitive Data Scanner configuration type.
type SensitiveDataScannerConfigurationType string

// List of SensitiveDataScannerConfigurationType.
const (
	SENSITIVEDATASCANNERCONFIGURATIONTYPE_SENSITIVE_DATA_SCANNER_CONFIGURATIONS SensitiveDataScannerConfigurationType = "sensitive_data_scanner_configuration"
)

var allowedSensitiveDataScannerConfigurationTypeEnumValues = []SensitiveDataScannerConfigurationType{
	SENSITIVEDATASCANNERCONFIGURATIONTYPE_SENSITIVE_DATA_SCANNER_CONFIGURATIONS,
}

// GetAllowedValues reeturns the list of possible values.
func (v *SensitiveDataScannerConfigurationType) GetAllowedValues() []SensitiveDataScannerConfigurationType {
	return allowedSensitiveDataScannerConfigurationTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *SensitiveDataScannerConfigurationType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = SensitiveDataScannerConfigurationType(value)
	return nil
}

// NewSensitiveDataScannerConfigurationTypeFromValue returns a pointer to a valid SensitiveDataScannerConfigurationType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewSensitiveDataScannerConfigurationTypeFromValue(v string) (*SensitiveDataScannerConfigurationType, error) {
	ev := SensitiveDataScannerConfigurationType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for SensitiveDataScannerConfigurationType: valid values are %v", v, allowedSensitiveDataScannerConfigurationTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v SensitiveDataScannerConfigurationType) IsValid() bool {
	for _, existing := range allowedSensitiveDataScannerConfigurationTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to SensitiveDataScannerConfigurationType value.
func (v SensitiveDataScannerConfigurationType) Ptr() *SensitiveDataScannerConfigurationType {
	return &v
}

// NullableSensitiveDataScannerConfigurationType handles when a null is used for SensitiveDataScannerConfigurationType.
type NullableSensitiveDataScannerConfigurationType struct {
	value *SensitiveDataScannerConfigurationType
	isSet bool
}

// Get returns the associated value.
func (v NullableSensitiveDataScannerConfigurationType) Get() *SensitiveDataScannerConfigurationType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableSensitiveDataScannerConfigurationType) Set(val *SensitiveDataScannerConfigurationType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableSensitiveDataScannerConfigurationType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableSensitiveDataScannerConfigurationType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableSensitiveDataScannerConfigurationType initializes the struct as if Set has been called.
func NewNullableSensitiveDataScannerConfigurationType(val *SensitiveDataScannerConfigurationType) *NullableSensitiveDataScannerConfigurationType {
	return &NullableSensitiveDataScannerConfigurationType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableSensitiveDataScannerConfigurationType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableSensitiveDataScannerConfigurationType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
