// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// SensitiveDataScannerProduct Datadog product onto which Sensitive Data Scanner can be activated.
type SensitiveDataScannerProduct string

// List of SensitiveDataScannerProduct.
const (
	SENSITIVEDATASCANNERPRODUCT_LOGS   SensitiveDataScannerProduct = "logs"
	SENSITIVEDATASCANNERPRODUCT_RUM    SensitiveDataScannerProduct = "rum"
	SENSITIVEDATASCANNERPRODUCT_EVENTS SensitiveDataScannerProduct = "events"
	SENSITIVEDATASCANNERPRODUCT_APM    SensitiveDataScannerProduct = "apm"
)

var allowedSensitiveDataScannerProductEnumValues = []SensitiveDataScannerProduct{
	SENSITIVEDATASCANNERPRODUCT_LOGS,
	SENSITIVEDATASCANNERPRODUCT_RUM,
	SENSITIVEDATASCANNERPRODUCT_EVENTS,
	SENSITIVEDATASCANNERPRODUCT_APM,
}

// GetAllowedValues reeturns the list of possible values.
func (v *SensitiveDataScannerProduct) GetAllowedValues() []SensitiveDataScannerProduct {
	return allowedSensitiveDataScannerProductEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *SensitiveDataScannerProduct) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = SensitiveDataScannerProduct(value)
	return nil
}

// NewSensitiveDataScannerProductFromValue returns a pointer to a valid SensitiveDataScannerProduct
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewSensitiveDataScannerProductFromValue(v string) (*SensitiveDataScannerProduct, error) {
	ev := SensitiveDataScannerProduct(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for SensitiveDataScannerProduct: valid values are %v", v, allowedSensitiveDataScannerProductEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v SensitiveDataScannerProduct) IsValid() bool {
	for _, existing := range allowedSensitiveDataScannerProductEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to SensitiveDataScannerProduct value.
func (v SensitiveDataScannerProduct) Ptr() *SensitiveDataScannerProduct {
	return &v
}

// NullableSensitiveDataScannerProduct handles when a null is used for SensitiveDataScannerProduct.
type NullableSensitiveDataScannerProduct struct {
	value *SensitiveDataScannerProduct
	isSet bool
}

// Get returns the associated value.
func (v NullableSensitiveDataScannerProduct) Get() *SensitiveDataScannerProduct {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableSensitiveDataScannerProduct) Set(val *SensitiveDataScannerProduct) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableSensitiveDataScannerProduct) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableSensitiveDataScannerProduct) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableSensitiveDataScannerProduct initializes the struct as if Set has been called.
func NewNullableSensitiveDataScannerProduct(val *SensitiveDataScannerProduct) *NullableSensitiveDataScannerProduct {
	return &NullableSensitiveDataScannerProduct{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableSensitiveDataScannerProduct) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableSensitiveDataScannerProduct) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
