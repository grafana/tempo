// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// HTTPLogErrors Invalid query performed.
type HTTPLogErrors struct {
	// Structured errors.
	Errors []HTTPLogError `json:"errors,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewHTTPLogErrors instantiates a new HTTPLogErrors object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewHTTPLogErrors() *HTTPLogErrors {
	this := HTTPLogErrors{}
	return &this
}

// NewHTTPLogErrorsWithDefaults instantiates a new HTTPLogErrors object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewHTTPLogErrorsWithDefaults() *HTTPLogErrors {
	this := HTTPLogErrors{}
	return &this
}

// GetErrors returns the Errors field value if set, zero value otherwise.
func (o *HTTPLogErrors) GetErrors() []HTTPLogError {
	if o == nil || o.Errors == nil {
		var ret []HTTPLogError
		return ret
	}
	return o.Errors
}

// GetErrorsOk returns a tuple with the Errors field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *HTTPLogErrors) GetErrorsOk() (*[]HTTPLogError, bool) {
	if o == nil || o.Errors == nil {
		return nil, false
	}
	return &o.Errors, true
}

// HasErrors returns a boolean if a field has been set.
func (o *HTTPLogErrors) HasErrors() bool {
	return o != nil && o.Errors != nil
}

// SetErrors gets a reference to the given []HTTPLogError and assigns it to the Errors field.
func (o *HTTPLogErrors) SetErrors(v []HTTPLogError) {
	o.Errors = v
}

// MarshalJSON serializes the struct using spec logic.
func (o HTTPLogErrors) MarshalJSON() ([]byte, error) {
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
func (o *HTTPLogErrors) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Errors []HTTPLogError `json:"errors,omitempty"`
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
