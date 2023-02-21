// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV1

import (
	"encoding/json"
)

// SyntheticsGlobalVariableTOTPParameters Parameters for the TOTP/MFA variable
type SyntheticsGlobalVariableTOTPParameters struct {
	// Number of digits for the OTP code.
	Digits *int32 `json:"digits,omitempty"`
	// Interval for which to refresh the token (in seconds).
	RefreshInterval *int32 `json:"refresh_interval,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewSyntheticsGlobalVariableTOTPParameters instantiates a new SyntheticsGlobalVariableTOTPParameters object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSyntheticsGlobalVariableTOTPParameters() *SyntheticsGlobalVariableTOTPParameters {
	this := SyntheticsGlobalVariableTOTPParameters{}
	return &this
}

// NewSyntheticsGlobalVariableTOTPParametersWithDefaults instantiates a new SyntheticsGlobalVariableTOTPParameters object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSyntheticsGlobalVariableTOTPParametersWithDefaults() *SyntheticsGlobalVariableTOTPParameters {
	this := SyntheticsGlobalVariableTOTPParameters{}
	return &this
}

// GetDigits returns the Digits field value if set, zero value otherwise.
func (o *SyntheticsGlobalVariableTOTPParameters) GetDigits() int32 {
	if o == nil || o.Digits == nil {
		var ret int32
		return ret
	}
	return *o.Digits
}

// GetDigitsOk returns a tuple with the Digits field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsGlobalVariableTOTPParameters) GetDigitsOk() (*int32, bool) {
	if o == nil || o.Digits == nil {
		return nil, false
	}
	return o.Digits, true
}

// HasDigits returns a boolean if a field has been set.
func (o *SyntheticsGlobalVariableTOTPParameters) HasDigits() bool {
	return o != nil && o.Digits != nil
}

// SetDigits gets a reference to the given int32 and assigns it to the Digits field.
func (o *SyntheticsGlobalVariableTOTPParameters) SetDigits(v int32) {
	o.Digits = &v
}

// GetRefreshInterval returns the RefreshInterval field value if set, zero value otherwise.
func (o *SyntheticsGlobalVariableTOTPParameters) GetRefreshInterval() int32 {
	if o == nil || o.RefreshInterval == nil {
		var ret int32
		return ret
	}
	return *o.RefreshInterval
}

// GetRefreshIntervalOk returns a tuple with the RefreshInterval field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsGlobalVariableTOTPParameters) GetRefreshIntervalOk() (*int32, bool) {
	if o == nil || o.RefreshInterval == nil {
		return nil, false
	}
	return o.RefreshInterval, true
}

// HasRefreshInterval returns a boolean if a field has been set.
func (o *SyntheticsGlobalVariableTOTPParameters) HasRefreshInterval() bool {
	return o != nil && o.RefreshInterval != nil
}

// SetRefreshInterval gets a reference to the given int32 and assigns it to the RefreshInterval field.
func (o *SyntheticsGlobalVariableTOTPParameters) SetRefreshInterval(v int32) {
	o.RefreshInterval = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SyntheticsGlobalVariableTOTPParameters) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Digits != nil {
		toSerialize["digits"] = o.Digits
	}
	if o.RefreshInterval != nil {
		toSerialize["refresh_interval"] = o.RefreshInterval
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SyntheticsGlobalVariableTOTPParameters) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Digits          *int32 `json:"digits,omitempty"`
		RefreshInterval *int32 `json:"refresh_interval,omitempty"`
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
	o.Digits = all.Digits
	o.RefreshInterval = all.RefreshInterval
	return nil
}
