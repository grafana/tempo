// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// IntakePayloadAccepted The payload accepted for intake.
type IntakePayloadAccepted struct {
	// A list of errors.
	Errors []string `json:"errors,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewIntakePayloadAccepted instantiates a new IntakePayloadAccepted object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewIntakePayloadAccepted() *IntakePayloadAccepted {
	this := IntakePayloadAccepted{}
	return &this
}

// NewIntakePayloadAcceptedWithDefaults instantiates a new IntakePayloadAccepted object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewIntakePayloadAcceptedWithDefaults() *IntakePayloadAccepted {
	this := IntakePayloadAccepted{}
	return &this
}

// GetErrors returns the Errors field value if set, zero value otherwise.
func (o *IntakePayloadAccepted) GetErrors() []string {
	if o == nil || o.Errors == nil {
		var ret []string
		return ret
	}
	return o.Errors
}

// GetErrorsOk returns a tuple with the Errors field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IntakePayloadAccepted) GetErrorsOk() (*[]string, bool) {
	if o == nil || o.Errors == nil {
		return nil, false
	}
	return &o.Errors, true
}

// HasErrors returns a boolean if a field has been set.
func (o *IntakePayloadAccepted) HasErrors() bool {
	return o != nil && o.Errors != nil
}

// SetErrors gets a reference to the given []string and assigns it to the Errors field.
func (o *IntakePayloadAccepted) SetErrors(v []string) {
	o.Errors = v
}

// MarshalJSON serializes the struct using spec logic.
func (o IntakePayloadAccepted) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Errors != nil {
		toSerialize["errors"] = o.Errors
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *IntakePayloadAccepted) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Errors []string `json:"errors,omitempty"`
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
	o.Errors = all.Errors
	return nil
}
