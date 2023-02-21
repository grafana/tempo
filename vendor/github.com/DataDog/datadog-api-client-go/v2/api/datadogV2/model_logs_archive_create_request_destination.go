// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// LogsArchiveCreateRequestDestination - An archive's destination.
type LogsArchiveCreateRequestDestination struct {
	LogsArchiveDestinationAzure *LogsArchiveDestinationAzure
	LogsArchiveDestinationGCS   *LogsArchiveDestinationGCS
	LogsArchiveDestinationS3    *LogsArchiveDestinationS3

	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject interface{}
}

// LogsArchiveDestinationAzureAsLogsArchiveCreateRequestDestination is a convenience function that returns LogsArchiveDestinationAzure wrapped in LogsArchiveCreateRequestDestination.
func LogsArchiveDestinationAzureAsLogsArchiveCreateRequestDestination(v *LogsArchiveDestinationAzure) LogsArchiveCreateRequestDestination {
	return LogsArchiveCreateRequestDestination{LogsArchiveDestinationAzure: v}
}

// LogsArchiveDestinationGCSAsLogsArchiveCreateRequestDestination is a convenience function that returns LogsArchiveDestinationGCS wrapped in LogsArchiveCreateRequestDestination.
func LogsArchiveDestinationGCSAsLogsArchiveCreateRequestDestination(v *LogsArchiveDestinationGCS) LogsArchiveCreateRequestDestination {
	return LogsArchiveCreateRequestDestination{LogsArchiveDestinationGCS: v}
}

// LogsArchiveDestinationS3AsLogsArchiveCreateRequestDestination is a convenience function that returns LogsArchiveDestinationS3 wrapped in LogsArchiveCreateRequestDestination.
func LogsArchiveDestinationS3AsLogsArchiveCreateRequestDestination(v *LogsArchiveDestinationS3) LogsArchiveCreateRequestDestination {
	return LogsArchiveCreateRequestDestination{LogsArchiveDestinationS3: v}
}

// UnmarshalJSON turns data into one of the pointers in the struct.
func (obj *LogsArchiveCreateRequestDestination) UnmarshalJSON(data []byte) error {
	var err error
	match := 0
	// try to unmarshal data into LogsArchiveDestinationAzure
	err = json.Unmarshal(data, &obj.LogsArchiveDestinationAzure)
	if err == nil {
		if obj.LogsArchiveDestinationAzure != nil && obj.LogsArchiveDestinationAzure.UnparsedObject == nil {
			jsonLogsArchiveDestinationAzure, _ := json.Marshal(obj.LogsArchiveDestinationAzure)
			if string(jsonLogsArchiveDestinationAzure) == "{}" { // empty struct
				obj.LogsArchiveDestinationAzure = nil
			} else {
				match++
			}
		} else {
			obj.LogsArchiveDestinationAzure = nil
		}
	} else {
		obj.LogsArchiveDestinationAzure = nil
	}

	// try to unmarshal data into LogsArchiveDestinationGCS
	err = json.Unmarshal(data, &obj.LogsArchiveDestinationGCS)
	if err == nil {
		if obj.LogsArchiveDestinationGCS != nil && obj.LogsArchiveDestinationGCS.UnparsedObject == nil {
			jsonLogsArchiveDestinationGCS, _ := json.Marshal(obj.LogsArchiveDestinationGCS)
			if string(jsonLogsArchiveDestinationGCS) == "{}" { // empty struct
				obj.LogsArchiveDestinationGCS = nil
			} else {
				match++
			}
		} else {
			obj.LogsArchiveDestinationGCS = nil
		}
	} else {
		obj.LogsArchiveDestinationGCS = nil
	}

	// try to unmarshal data into LogsArchiveDestinationS3
	err = json.Unmarshal(data, &obj.LogsArchiveDestinationS3)
	if err == nil {
		if obj.LogsArchiveDestinationS3 != nil && obj.LogsArchiveDestinationS3.UnparsedObject == nil {
			jsonLogsArchiveDestinationS3, _ := json.Marshal(obj.LogsArchiveDestinationS3)
			if string(jsonLogsArchiveDestinationS3) == "{}" { // empty struct
				obj.LogsArchiveDestinationS3 = nil
			} else {
				match++
			}
		} else {
			obj.LogsArchiveDestinationS3 = nil
		}
	} else {
		obj.LogsArchiveDestinationS3 = nil
	}

	if match != 1 { // more than 1 match
		// reset to nil
		obj.LogsArchiveDestinationAzure = nil
		obj.LogsArchiveDestinationGCS = nil
		obj.LogsArchiveDestinationS3 = nil
		return json.Unmarshal(data, &obj.UnparsedObject)
	}
	return nil // exactly one match
}

// MarshalJSON turns data from the first non-nil pointers in the struct to JSON.
func (obj LogsArchiveCreateRequestDestination) MarshalJSON() ([]byte, error) {
	if obj.LogsArchiveDestinationAzure != nil {
		return json.Marshal(&obj.LogsArchiveDestinationAzure)
	}

	if obj.LogsArchiveDestinationGCS != nil {
		return json.Marshal(&obj.LogsArchiveDestinationGCS)
	}

	if obj.LogsArchiveDestinationS3 != nil {
		return json.Marshal(&obj.LogsArchiveDestinationS3)
	}

	if obj.UnparsedObject != nil {
		return json.Marshal(obj.UnparsedObject)
	}
	return nil, nil // no data in oneOf schemas
}

// GetActualInstance returns the actual instance.
func (obj *LogsArchiveCreateRequestDestination) GetActualInstance() interface{} {
	if obj.LogsArchiveDestinationAzure != nil {
		return obj.LogsArchiveDestinationAzure
	}

	if obj.LogsArchiveDestinationGCS != nil {
		return obj.LogsArchiveDestinationGCS
	}

	if obj.LogsArchiveDestinationS3 != nil {
		return obj.LogsArchiveDestinationS3
	}

	// all schemas are nil
	return nil
}

// NullableLogsArchiveCreateRequestDestination handles when a null is used for LogsArchiveCreateRequestDestination.
type NullableLogsArchiveCreateRequestDestination struct {
	value *LogsArchiveCreateRequestDestination
	isSet bool
}

// Get returns the associated value.
func (v NullableLogsArchiveCreateRequestDestination) Get() *LogsArchiveCreateRequestDestination {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableLogsArchiveCreateRequestDestination) Set(val *LogsArchiveCreateRequestDestination) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableLogsArchiveCreateRequestDestination) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag/
func (v *NullableLogsArchiveCreateRequestDestination) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableLogsArchiveCreateRequestDestination initializes the struct as if Set has been called.
func NewNullableLogsArchiveCreateRequestDestination(val *LogsArchiveCreateRequestDestination) *NullableLogsArchiveCreateRequestDestination {
	return &NullableLogsArchiveCreateRequestDestination{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableLogsArchiveCreateRequestDestination) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableLogsArchiveCreateRequestDestination) UnmarshalJSON(src []byte) error {
	v.isSet = true

	// this object is nullable so check if the payload is null or empty string
	if string(src) == "" || string(src) == "{}" {
		return nil
	}

	return json.Unmarshal(src, &v.value)
}
