// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// AuditLogsResponseStatus The status of the response.
type AuditLogsResponseStatus string

// List of AuditLogsResponseStatus.
const (
	AUDITLOGSRESPONSESTATUS_DONE    AuditLogsResponseStatus = "done"
	AUDITLOGSRESPONSESTATUS_TIMEOUT AuditLogsResponseStatus = "timeout"
)

var allowedAuditLogsResponseStatusEnumValues = []AuditLogsResponseStatus{
	AUDITLOGSRESPONSESTATUS_DONE,
	AUDITLOGSRESPONSESTATUS_TIMEOUT,
}

// GetAllowedValues reeturns the list of possible values.
func (v *AuditLogsResponseStatus) GetAllowedValues() []AuditLogsResponseStatus {
	return allowedAuditLogsResponseStatusEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *AuditLogsResponseStatus) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = AuditLogsResponseStatus(value)
	return nil
}

// NewAuditLogsResponseStatusFromValue returns a pointer to a valid AuditLogsResponseStatus
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewAuditLogsResponseStatusFromValue(v string) (*AuditLogsResponseStatus, error) {
	ev := AuditLogsResponseStatus(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for AuditLogsResponseStatus: valid values are %v", v, allowedAuditLogsResponseStatusEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v AuditLogsResponseStatus) IsValid() bool {
	for _, existing := range allowedAuditLogsResponseStatusEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to AuditLogsResponseStatus value.
func (v AuditLogsResponseStatus) Ptr() *AuditLogsResponseStatus {
	return &v
}

// NullableAuditLogsResponseStatus handles when a null is used for AuditLogsResponseStatus.
type NullableAuditLogsResponseStatus struct {
	value *AuditLogsResponseStatus
	isSet bool
}

// Get returns the associated value.
func (v NullableAuditLogsResponseStatus) Get() *AuditLogsResponseStatus {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableAuditLogsResponseStatus) Set(val *AuditLogsResponseStatus) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableAuditLogsResponseStatus) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableAuditLogsResponseStatus) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableAuditLogsResponseStatus initializes the struct as if Set has been called.
func NewNullableAuditLogsResponseStatus(val *AuditLogsResponseStatus) *NullableAuditLogsResponseStatus {
	return &NullableAuditLogsResponseStatus{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableAuditLogsResponseStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableAuditLogsResponseStatus) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
