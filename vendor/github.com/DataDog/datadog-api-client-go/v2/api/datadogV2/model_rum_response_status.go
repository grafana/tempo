// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// RUMResponseStatus The status of the response.
type RUMResponseStatus string

// List of RUMResponseStatus.
const (
	RUMRESPONSESTATUS_DONE    RUMResponseStatus = "done"
	RUMRESPONSESTATUS_TIMEOUT RUMResponseStatus = "timeout"
)

var allowedRUMResponseStatusEnumValues = []RUMResponseStatus{
	RUMRESPONSESTATUS_DONE,
	RUMRESPONSESTATUS_TIMEOUT,
}

// GetAllowedValues reeturns the list of possible values.
func (v *RUMResponseStatus) GetAllowedValues() []RUMResponseStatus {
	return allowedRUMResponseStatusEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *RUMResponseStatus) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = RUMResponseStatus(value)
	return nil
}

// NewRUMResponseStatusFromValue returns a pointer to a valid RUMResponseStatus
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewRUMResponseStatusFromValue(v string) (*RUMResponseStatus, error) {
	ev := RUMResponseStatus(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for RUMResponseStatus: valid values are %v", v, allowedRUMResponseStatusEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v RUMResponseStatus) IsValid() bool {
	for _, existing := range allowedRUMResponseStatusEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to RUMResponseStatus value.
func (v RUMResponseStatus) Ptr() *RUMResponseStatus {
	return &v
}

// NullableRUMResponseStatus handles when a null is used for RUMResponseStatus.
type NullableRUMResponseStatus struct {
	value *RUMResponseStatus
	isSet bool
}

// Get returns the associated value.
func (v NullableRUMResponseStatus) Get() *RUMResponseStatus {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableRUMResponseStatus) Set(val *RUMResponseStatus) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableRUMResponseStatus) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableRUMResponseStatus) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableRUMResponseStatus initializes the struct as if Set has been called.
func NewNullableRUMResponseStatus(val *RUMResponseStatus) *NullableRUMResponseStatus {
	return &NullableRUMResponseStatus{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableRUMResponseStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableRUMResponseStatus) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
