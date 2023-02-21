// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// AuditLogsSort Sort parameters when querying events.
type AuditLogsSort string

// List of AuditLogsSort.
const (
	AUDITLOGSSORT_TIMESTAMP_ASCENDING  AuditLogsSort = "timestamp"
	AUDITLOGSSORT_TIMESTAMP_DESCENDING AuditLogsSort = "-timestamp"
)

var allowedAuditLogsSortEnumValues = []AuditLogsSort{
	AUDITLOGSSORT_TIMESTAMP_ASCENDING,
	AUDITLOGSSORT_TIMESTAMP_DESCENDING,
}

// GetAllowedValues reeturns the list of possible values.
func (v *AuditLogsSort) GetAllowedValues() []AuditLogsSort {
	return allowedAuditLogsSortEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *AuditLogsSort) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = AuditLogsSort(value)
	return nil
}

// NewAuditLogsSortFromValue returns a pointer to a valid AuditLogsSort
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewAuditLogsSortFromValue(v string) (*AuditLogsSort, error) {
	ev := AuditLogsSort(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for AuditLogsSort: valid values are %v", v, allowedAuditLogsSortEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v AuditLogsSort) IsValid() bool {
	for _, existing := range allowedAuditLogsSortEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to AuditLogsSort value.
func (v AuditLogsSort) Ptr() *AuditLogsSort {
	return &v
}

// NullableAuditLogsSort handles when a null is used for AuditLogsSort.
type NullableAuditLogsSort struct {
	value *AuditLogsSort
	isSet bool
}

// Get returns the associated value.
func (v NullableAuditLogsSort) Get() *AuditLogsSort {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableAuditLogsSort) Set(val *AuditLogsSort) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableAuditLogsSort) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableAuditLogsSort) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableAuditLogsSort initializes the struct as if Set has been called.
func NewNullableAuditLogsSort(val *AuditLogsSort) *NullableAuditLogsSort {
	return &NullableAuditLogsSort{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableAuditLogsSort) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableAuditLogsSort) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
