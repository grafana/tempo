// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV1

import (
	"encoding/json"
	"fmt"
)

// SyntheticsTestOptionsHTTPVersion HTTP version to use for a Synthetic test.
type SyntheticsTestOptionsHTTPVersion string

// List of SyntheticsTestOptionsHTTPVersion.
const (
	SYNTHETICSTESTOPTIONSHTTPVERSION_HTTP1 SyntheticsTestOptionsHTTPVersion = "http1"
	SYNTHETICSTESTOPTIONSHTTPVERSION_HTTP2 SyntheticsTestOptionsHTTPVersion = "http2"
	SYNTHETICSTESTOPTIONSHTTPVERSION_ANY   SyntheticsTestOptionsHTTPVersion = "any"
)

var allowedSyntheticsTestOptionsHTTPVersionEnumValues = []SyntheticsTestOptionsHTTPVersion{
	SYNTHETICSTESTOPTIONSHTTPVERSION_HTTP1,
	SYNTHETICSTESTOPTIONSHTTPVERSION_HTTP2,
	SYNTHETICSTESTOPTIONSHTTPVERSION_ANY,
}

// GetAllowedValues reeturns the list of possible values.
func (v *SyntheticsTestOptionsHTTPVersion) GetAllowedValues() []SyntheticsTestOptionsHTTPVersion {
	return allowedSyntheticsTestOptionsHTTPVersionEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *SyntheticsTestOptionsHTTPVersion) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = SyntheticsTestOptionsHTTPVersion(value)
	return nil
}

// NewSyntheticsTestOptionsHTTPVersionFromValue returns a pointer to a valid SyntheticsTestOptionsHTTPVersion
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewSyntheticsTestOptionsHTTPVersionFromValue(v string) (*SyntheticsTestOptionsHTTPVersion, error) {
	ev := SyntheticsTestOptionsHTTPVersion(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for SyntheticsTestOptionsHTTPVersion: valid values are %v", v, allowedSyntheticsTestOptionsHTTPVersionEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v SyntheticsTestOptionsHTTPVersion) IsValid() bool {
	for _, existing := range allowedSyntheticsTestOptionsHTTPVersionEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to SyntheticsTestOptionsHTTPVersion value.
func (v SyntheticsTestOptionsHTTPVersion) Ptr() *SyntheticsTestOptionsHTTPVersion {
	return &v
}

// NullableSyntheticsTestOptionsHTTPVersion handles when a null is used for SyntheticsTestOptionsHTTPVersion.
type NullableSyntheticsTestOptionsHTTPVersion struct {
	value *SyntheticsTestOptionsHTTPVersion
	isSet bool
}

// Get returns the associated value.
func (v NullableSyntheticsTestOptionsHTTPVersion) Get() *SyntheticsTestOptionsHTTPVersion {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableSyntheticsTestOptionsHTTPVersion) Set(val *SyntheticsTestOptionsHTTPVersion) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableSyntheticsTestOptionsHTTPVersion) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableSyntheticsTestOptionsHTTPVersion) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableSyntheticsTestOptionsHTTPVersion initializes the struct as if Set has been called.
func NewNullableSyntheticsTestOptionsHTTPVersion(val *SyntheticsTestOptionsHTTPVersion) *NullableSyntheticsTestOptionsHTTPVersion {
	return &NullableSyntheticsTestOptionsHTTPVersion{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableSyntheticsTestOptionsHTTPVersion) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableSyntheticsTestOptionsHTTPVersion) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
