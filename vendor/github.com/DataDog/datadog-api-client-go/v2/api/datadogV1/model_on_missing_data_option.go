// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV1

import (
	"encoding/json"
	"fmt"
)

// OnMissingDataOption Controls how groups or monitors are treated if an evaluation does not return any data points.
// The default option results in different behavior depending on the monitor query type.
// For monitors using Count queries, an empty monitor evaluation is treated as 0 and is compared to the threshold conditions.
// For monitors using any query type other than Count, for example Gauge, Measure, or Rate, the monitor shows the last known status.
// This option is only available for APM Trace Analytics, Audit Trail, CI, Error Tracking, Event, Logs, and RUM monitors.
type OnMissingDataOption string

// List of OnMissingDataOption.
const (
	ONMISSINGDATAOPTION_DEFAULT                 OnMissingDataOption = "default"
	ONMISSINGDATAOPTION_SHOW_NO_DATA            OnMissingDataOption = "show_no_data"
	ONMISSINGDATAOPTION_SHOW_AND_NOTIFY_NO_DATA OnMissingDataOption = "show_and_notify_no_data"
	ONMISSINGDATAOPTION_RESOLVE                 OnMissingDataOption = "resolve"
)

var allowedOnMissingDataOptionEnumValues = []OnMissingDataOption{
	ONMISSINGDATAOPTION_DEFAULT,
	ONMISSINGDATAOPTION_SHOW_NO_DATA,
	ONMISSINGDATAOPTION_SHOW_AND_NOTIFY_NO_DATA,
	ONMISSINGDATAOPTION_RESOLVE,
}

// GetAllowedValues reeturns the list of possible values.
func (v *OnMissingDataOption) GetAllowedValues() []OnMissingDataOption {
	return allowedOnMissingDataOptionEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *OnMissingDataOption) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = OnMissingDataOption(value)
	return nil
}

// NewOnMissingDataOptionFromValue returns a pointer to a valid OnMissingDataOption
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewOnMissingDataOptionFromValue(v string) (*OnMissingDataOption, error) {
	ev := OnMissingDataOption(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for OnMissingDataOption: valid values are %v", v, allowedOnMissingDataOptionEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v OnMissingDataOption) IsValid() bool {
	for _, existing := range allowedOnMissingDataOptionEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to OnMissingDataOption value.
func (v OnMissingDataOption) Ptr() *OnMissingDataOption {
	return &v
}

// NullableOnMissingDataOption handles when a null is used for OnMissingDataOption.
type NullableOnMissingDataOption struct {
	value *OnMissingDataOption
	isSet bool
}

// Get returns the associated value.
func (v NullableOnMissingDataOption) Get() *OnMissingDataOption {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableOnMissingDataOption) Set(val *OnMissingDataOption) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableOnMissingDataOption) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableOnMissingDataOption) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableOnMissingDataOption initializes the struct as if Set has been called.
func NewNullableOnMissingDataOption(val *OnMissingDataOption) *NullableOnMissingDataOption {
	return &NullableOnMissingDataOption{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableOnMissingDataOption) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableOnMissingDataOption) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
