// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// CIAppResponseMetadata The metadata associated with a request.
type CIAppResponseMetadata struct {
	// The time elapsed in milliseconds.
	Elapsed *int64 `json:"elapsed,omitempty"`
	// The identifier of the request.
	RequestId *string `json:"request_id,omitempty"`
	// The status of the response.
	Status *CIAppResponseStatus `json:"status,omitempty"`
	// A list of warnings (non-fatal errors) encountered. Partial results may return if
	// warnings are present in the response.
	Warnings []CIAppWarning `json:"warnings,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewCIAppResponseMetadata instantiates a new CIAppResponseMetadata object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewCIAppResponseMetadata() *CIAppResponseMetadata {
	this := CIAppResponseMetadata{}
	return &this
}

// NewCIAppResponseMetadataWithDefaults instantiates a new CIAppResponseMetadata object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewCIAppResponseMetadataWithDefaults() *CIAppResponseMetadata {
	this := CIAppResponseMetadata{}
	return &this
}

// GetElapsed returns the Elapsed field value if set, zero value otherwise.
func (o *CIAppResponseMetadata) GetElapsed() int64 {
	if o == nil || o.Elapsed == nil {
		var ret int64
		return ret
	}
	return *o.Elapsed
}

// GetElapsedOk returns a tuple with the Elapsed field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *CIAppResponseMetadata) GetElapsedOk() (*int64, bool) {
	if o == nil || o.Elapsed == nil {
		return nil, false
	}
	return o.Elapsed, true
}

// HasElapsed returns a boolean if a field has been set.
func (o *CIAppResponseMetadata) HasElapsed() bool {
	return o != nil && o.Elapsed != nil
}

// SetElapsed gets a reference to the given int64 and assigns it to the Elapsed field.
func (o *CIAppResponseMetadata) SetElapsed(v int64) {
	o.Elapsed = &v
}

// GetRequestId returns the RequestId field value if set, zero value otherwise.
func (o *CIAppResponseMetadata) GetRequestId() string {
	if o == nil || o.RequestId == nil {
		var ret string
		return ret
	}
	return *o.RequestId
}

// GetRequestIdOk returns a tuple with the RequestId field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *CIAppResponseMetadata) GetRequestIdOk() (*string, bool) {
	if o == nil || o.RequestId == nil {
		return nil, false
	}
	return o.RequestId, true
}

// HasRequestId returns a boolean if a field has been set.
func (o *CIAppResponseMetadata) HasRequestId() bool {
	return o != nil && o.RequestId != nil
}

// SetRequestId gets a reference to the given string and assigns it to the RequestId field.
func (o *CIAppResponseMetadata) SetRequestId(v string) {
	o.RequestId = &v
}

// GetStatus returns the Status field value if set, zero value otherwise.
func (o *CIAppResponseMetadata) GetStatus() CIAppResponseStatus {
	if o == nil || o.Status == nil {
		var ret CIAppResponseStatus
		return ret
	}
	return *o.Status
}

// GetStatusOk returns a tuple with the Status field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *CIAppResponseMetadata) GetStatusOk() (*CIAppResponseStatus, bool) {
	if o == nil || o.Status == nil {
		return nil, false
	}
	return o.Status, true
}

// HasStatus returns a boolean if a field has been set.
func (o *CIAppResponseMetadata) HasStatus() bool {
	return o != nil && o.Status != nil
}

// SetStatus gets a reference to the given CIAppResponseStatus and assigns it to the Status field.
func (o *CIAppResponseMetadata) SetStatus(v CIAppResponseStatus) {
	o.Status = &v
}

// GetWarnings returns the Warnings field value if set, zero value otherwise.
func (o *CIAppResponseMetadata) GetWarnings() []CIAppWarning {
	if o == nil || o.Warnings == nil {
		var ret []CIAppWarning
		return ret
	}
	return o.Warnings
}

// GetWarningsOk returns a tuple with the Warnings field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *CIAppResponseMetadata) GetWarningsOk() (*[]CIAppWarning, bool) {
	if o == nil || o.Warnings == nil {
		return nil, false
	}
	return &o.Warnings, true
}

// HasWarnings returns a boolean if a field has been set.
func (o *CIAppResponseMetadata) HasWarnings() bool {
	return o != nil && o.Warnings != nil
}

// SetWarnings gets a reference to the given []CIAppWarning and assigns it to the Warnings field.
func (o *CIAppResponseMetadata) SetWarnings(v []CIAppWarning) {
	o.Warnings = v
}

// MarshalJSON serializes the struct using spec logic.
func (o CIAppResponseMetadata) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Elapsed != nil {
		toSerialize["elapsed"] = o.Elapsed
	}
	if o.RequestId != nil {
		toSerialize["request_id"] = o.RequestId
	}
	if o.Status != nil {
		toSerialize["status"] = o.Status
	}
	if o.Warnings != nil {
		toSerialize["warnings"] = o.Warnings
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *CIAppResponseMetadata) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Elapsed   *int64               `json:"elapsed,omitempty"`
		RequestId *string              `json:"request_id,omitempty"`
		Status    *CIAppResponseStatus `json:"status,omitempty"`
		Warnings  []CIAppWarning       `json:"warnings,omitempty"`
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
	if v := all.Status; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.Elapsed = all.Elapsed
	o.RequestId = all.RequestId
	o.Status = all.Status
	o.Warnings = all.Warnings
	return nil
}
