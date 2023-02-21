// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// SensitiveDataScannerGetConfigIncludedItem - An object related to the configuration.
type SensitiveDataScannerGetConfigIncludedItem struct {
	SensitiveDataScannerRuleIncludedItem  *SensitiveDataScannerRuleIncludedItem
	SensitiveDataScannerGroupIncludedItem *SensitiveDataScannerGroupIncludedItem

	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject interface{}
}

// SensitiveDataScannerRuleIncludedItemAsSensitiveDataScannerGetConfigIncludedItem is a convenience function that returns SensitiveDataScannerRuleIncludedItem wrapped in SensitiveDataScannerGetConfigIncludedItem.
func SensitiveDataScannerRuleIncludedItemAsSensitiveDataScannerGetConfigIncludedItem(v *SensitiveDataScannerRuleIncludedItem) SensitiveDataScannerGetConfigIncludedItem {
	return SensitiveDataScannerGetConfigIncludedItem{SensitiveDataScannerRuleIncludedItem: v}
}

// SensitiveDataScannerGroupIncludedItemAsSensitiveDataScannerGetConfigIncludedItem is a convenience function that returns SensitiveDataScannerGroupIncludedItem wrapped in SensitiveDataScannerGetConfigIncludedItem.
func SensitiveDataScannerGroupIncludedItemAsSensitiveDataScannerGetConfigIncludedItem(v *SensitiveDataScannerGroupIncludedItem) SensitiveDataScannerGetConfigIncludedItem {
	return SensitiveDataScannerGetConfigIncludedItem{SensitiveDataScannerGroupIncludedItem: v}
}

// UnmarshalJSON turns data into one of the pointers in the struct.
func (obj *SensitiveDataScannerGetConfigIncludedItem) UnmarshalJSON(data []byte) error {
	var err error
	match := 0
	// try to unmarshal data into SensitiveDataScannerRuleIncludedItem
	err = json.Unmarshal(data, &obj.SensitiveDataScannerRuleIncludedItem)
	if err == nil {
		if obj.SensitiveDataScannerRuleIncludedItem != nil && obj.SensitiveDataScannerRuleIncludedItem.UnparsedObject == nil {
			jsonSensitiveDataScannerRuleIncludedItem, _ := json.Marshal(obj.SensitiveDataScannerRuleIncludedItem)
			if string(jsonSensitiveDataScannerRuleIncludedItem) == "{}" { // empty struct
				obj.SensitiveDataScannerRuleIncludedItem = nil
			} else {
				match++
			}
		} else {
			obj.SensitiveDataScannerRuleIncludedItem = nil
		}
	} else {
		obj.SensitiveDataScannerRuleIncludedItem = nil
	}

	// try to unmarshal data into SensitiveDataScannerGroupIncludedItem
	err = json.Unmarshal(data, &obj.SensitiveDataScannerGroupIncludedItem)
	if err == nil {
		if obj.SensitiveDataScannerGroupIncludedItem != nil && obj.SensitiveDataScannerGroupIncludedItem.UnparsedObject == nil {
			jsonSensitiveDataScannerGroupIncludedItem, _ := json.Marshal(obj.SensitiveDataScannerGroupIncludedItem)
			if string(jsonSensitiveDataScannerGroupIncludedItem) == "{}" { // empty struct
				obj.SensitiveDataScannerGroupIncludedItem = nil
			} else {
				match++
			}
		} else {
			obj.SensitiveDataScannerGroupIncludedItem = nil
		}
	} else {
		obj.SensitiveDataScannerGroupIncludedItem = nil
	}

	if match != 1 { // more than 1 match
		// reset to nil
		obj.SensitiveDataScannerRuleIncludedItem = nil
		obj.SensitiveDataScannerGroupIncludedItem = nil
		return json.Unmarshal(data, &obj.UnparsedObject)
	}
	return nil // exactly one match
}

// MarshalJSON turns data from the first non-nil pointers in the struct to JSON.
func (obj SensitiveDataScannerGetConfigIncludedItem) MarshalJSON() ([]byte, error) {
	if obj.SensitiveDataScannerRuleIncludedItem != nil {
		return json.Marshal(&obj.SensitiveDataScannerRuleIncludedItem)
	}

	if obj.SensitiveDataScannerGroupIncludedItem != nil {
		return json.Marshal(&obj.SensitiveDataScannerGroupIncludedItem)
	}

	if obj.UnparsedObject != nil {
		return json.Marshal(obj.UnparsedObject)
	}
	return nil, nil // no data in oneOf schemas
}

// GetActualInstance returns the actual instance.
func (obj *SensitiveDataScannerGetConfigIncludedItem) GetActualInstance() interface{} {
	if obj.SensitiveDataScannerRuleIncludedItem != nil {
		return obj.SensitiveDataScannerRuleIncludedItem
	}

	if obj.SensitiveDataScannerGroupIncludedItem != nil {
		return obj.SensitiveDataScannerGroupIncludedItem
	}

	// all schemas are nil
	return nil
}

// NullableSensitiveDataScannerGetConfigIncludedItem handles when a null is used for SensitiveDataScannerGetConfigIncludedItem.
type NullableSensitiveDataScannerGetConfigIncludedItem struct {
	value *SensitiveDataScannerGetConfigIncludedItem
	isSet bool
}

// Get returns the associated value.
func (v NullableSensitiveDataScannerGetConfigIncludedItem) Get() *SensitiveDataScannerGetConfigIncludedItem {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableSensitiveDataScannerGetConfigIncludedItem) Set(val *SensitiveDataScannerGetConfigIncludedItem) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableSensitiveDataScannerGetConfigIncludedItem) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag/
func (v *NullableSensitiveDataScannerGetConfigIncludedItem) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableSensitiveDataScannerGetConfigIncludedItem initializes the struct as if Set has been called.
func NewNullableSensitiveDataScannerGetConfigIncludedItem(val *SensitiveDataScannerGetConfigIncludedItem) *NullableSensitiveDataScannerGetConfigIncludedItem {
	return &NullableSensitiveDataScannerGetConfigIncludedItem{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableSensitiveDataScannerGetConfigIncludedItem) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableSensitiveDataScannerGetConfigIncludedItem) UnmarshalJSON(src []byte) error {
	v.isSet = true

	// this object is nullable so check if the payload is null or empty string
	if string(src) == "" || string(src) == "{}" {
		return nil
	}

	return json.Unmarshal(src, &v.value)
}
