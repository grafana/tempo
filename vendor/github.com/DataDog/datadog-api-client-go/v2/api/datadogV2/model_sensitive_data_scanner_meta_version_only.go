// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// SensitiveDataScannerMetaVersionOnly Meta payload containing information about the API.
type SensitiveDataScannerMetaVersionOnly struct {
	// Version of the API (optional).
	Version *int64 `json:"version,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewSensitiveDataScannerMetaVersionOnly instantiates a new SensitiveDataScannerMetaVersionOnly object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSensitiveDataScannerMetaVersionOnly() *SensitiveDataScannerMetaVersionOnly {
	this := SensitiveDataScannerMetaVersionOnly{}
	return &this
}

// NewSensitiveDataScannerMetaVersionOnlyWithDefaults instantiates a new SensitiveDataScannerMetaVersionOnly object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSensitiveDataScannerMetaVersionOnlyWithDefaults() *SensitiveDataScannerMetaVersionOnly {
	this := SensitiveDataScannerMetaVersionOnly{}
	return &this
}

// GetVersion returns the Version field value if set, zero value otherwise.
func (o *SensitiveDataScannerMetaVersionOnly) GetVersion() int64 {
	if o == nil || o.Version == nil {
		var ret int64
		return ret
	}
	return *o.Version
}

// GetVersionOk returns a tuple with the Version field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SensitiveDataScannerMetaVersionOnly) GetVersionOk() (*int64, bool) {
	if o == nil || o.Version == nil {
		return nil, false
	}
	return o.Version, true
}

// HasVersion returns a boolean if a field has been set.
func (o *SensitiveDataScannerMetaVersionOnly) HasVersion() bool {
	return o != nil && o.Version != nil
}

// SetVersion gets a reference to the given int64 and assigns it to the Version field.
func (o *SensitiveDataScannerMetaVersionOnly) SetVersion(v int64) {
	o.Version = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SensitiveDataScannerMetaVersionOnly) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Version != nil {
		toSerialize["version"] = o.Version
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SensitiveDataScannerMetaVersionOnly) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Version *int64 `json:"version,omitempty"`
	}{}
	err = json.Unmarshal(bytes, &all)
	if err != nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.Version = all.Version
	return nil
}
