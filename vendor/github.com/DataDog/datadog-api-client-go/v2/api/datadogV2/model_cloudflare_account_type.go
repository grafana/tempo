// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// CloudflareAccountType The JSON:API type for this API. Should always be `cloudflare-accounts`.
type CloudflareAccountType string

// List of CloudflareAccountType.
const (
	CLOUDFLAREACCOUNTTYPE_CLOUDFLARE_ACCOUNTS CloudflareAccountType = "cloudflare-accounts"
)

var allowedCloudflareAccountTypeEnumValues = []CloudflareAccountType{
	CLOUDFLAREACCOUNTTYPE_CLOUDFLARE_ACCOUNTS,
}

// GetAllowedValues reeturns the list of possible values.
func (v *CloudflareAccountType) GetAllowedValues() []CloudflareAccountType {
	return allowedCloudflareAccountTypeEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *CloudflareAccountType) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = CloudflareAccountType(value)
	return nil
}

// NewCloudflareAccountTypeFromValue returns a pointer to a valid CloudflareAccountType
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewCloudflareAccountTypeFromValue(v string) (*CloudflareAccountType, error) {
	ev := CloudflareAccountType(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for CloudflareAccountType: valid values are %v", v, allowedCloudflareAccountTypeEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v CloudflareAccountType) IsValid() bool {
	for _, existing := range allowedCloudflareAccountTypeEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to CloudflareAccountType value.
func (v CloudflareAccountType) Ptr() *CloudflareAccountType {
	return &v
}

// NullableCloudflareAccountType handles when a null is used for CloudflareAccountType.
type NullableCloudflareAccountType struct {
	value *CloudflareAccountType
	isSet bool
}

// Get returns the associated value.
func (v NullableCloudflareAccountType) Get() *CloudflareAccountType {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableCloudflareAccountType) Set(val *CloudflareAccountType) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableCloudflareAccountType) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableCloudflareAccountType) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableCloudflareAccountType initializes the struct as if Set has been called.
func NewNullableCloudflareAccountType(val *CloudflareAccountType) *NullableCloudflareAccountType {
	return &NullableCloudflareAccountType{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableCloudflareAccountType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableCloudflareAccountType) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
