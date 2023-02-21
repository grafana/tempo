// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// RUMResponseMetadata The metadata associated with a request.
type RUMResponseMetadata struct {
	// The time elapsed in milliseconds.
	Elapsed *int64 `json:"elapsed,omitempty"`
	// Paging attributes.
	Page *RUMResponsePage `json:"page,omitempty"`
	// The identifier of the request.
	RequestId *string `json:"request_id,omitempty"`
	// The status of the response.
	Status *RUMResponseStatus `json:"status,omitempty"`
	// A list of warnings (non-fatal errors) encountered. Partial results may return if
	// warnings are present in the response.
	Warnings []RUMWarning `json:"warnings,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewRUMResponseMetadata instantiates a new RUMResponseMetadata object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewRUMResponseMetadata() *RUMResponseMetadata {
	this := RUMResponseMetadata{}
	return &this
}

// NewRUMResponseMetadataWithDefaults instantiates a new RUMResponseMetadata object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewRUMResponseMetadataWithDefaults() *RUMResponseMetadata {
	this := RUMResponseMetadata{}
	return &this
}

// GetElapsed returns the Elapsed field value if set, zero value otherwise.
func (o *RUMResponseMetadata) GetElapsed() int64 {
	if o == nil || o.Elapsed == nil {
		var ret int64
		return ret
	}
	return *o.Elapsed
}

// GetElapsedOk returns a tuple with the Elapsed field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *RUMResponseMetadata) GetElapsedOk() (*int64, bool) {
	if o == nil || o.Elapsed == nil {
		return nil, false
	}
	return o.Elapsed, true
}

// HasElapsed returns a boolean if a field has been set.
func (o *RUMResponseMetadata) HasElapsed() bool {
	return o != nil && o.Elapsed != nil
}

// SetElapsed gets a reference to the given int64 and assigns it to the Elapsed field.
func (o *RUMResponseMetadata) SetElapsed(v int64) {
	o.Elapsed = &v
}

// GetPage returns the Page field value if set, zero value otherwise.
func (o *RUMResponseMetadata) GetPage() RUMResponsePage {
	if o == nil || o.Page == nil {
		var ret RUMResponsePage
		return ret
	}
	return *o.Page
}

// GetPageOk returns a tuple with the Page field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *RUMResponseMetadata) GetPageOk() (*RUMResponsePage, bool) {
	if o == nil || o.Page == nil {
		return nil, false
	}
	return o.Page, true
}

// HasPage returns a boolean if a field has been set.
func (o *RUMResponseMetadata) HasPage() bool {
	return o != nil && o.Page != nil
}

// SetPage gets a reference to the given RUMResponsePage and assigns it to the Page field.
func (o *RUMResponseMetadata) SetPage(v RUMResponsePage) {
	o.Page = &v
}

// GetRequestId returns the RequestId field value if set, zero value otherwise.
func (o *RUMResponseMetadata) GetRequestId() string {
	if o == nil || o.RequestId == nil {
		var ret string
		return ret
	}
	return *o.RequestId
}

// GetRequestIdOk returns a tuple with the RequestId field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *RUMResponseMetadata) GetRequestIdOk() (*string, bool) {
	if o == nil || o.RequestId == nil {
		return nil, false
	}
	return o.RequestId, true
}

// HasRequestId returns a boolean if a field has been set.
func (o *RUMResponseMetadata) HasRequestId() bool {
	return o != nil && o.RequestId != nil
}

// SetRequestId gets a reference to the given string and assigns it to the RequestId field.
func (o *RUMResponseMetadata) SetRequestId(v string) {
	o.RequestId = &v
}

// GetStatus returns the Status field value if set, zero value otherwise.
func (o *RUMResponseMetadata) GetStatus() RUMResponseStatus {
	if o == nil || o.Status == nil {
		var ret RUMResponseStatus
		return ret
	}
	return *o.Status
}

// GetStatusOk returns a tuple with the Status field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *RUMResponseMetadata) GetStatusOk() (*RUMResponseStatus, bool) {
	if o == nil || o.Status == nil {
		return nil, false
	}
	return o.Status, true
}

// HasStatus returns a boolean if a field has been set.
func (o *RUMResponseMetadata) HasStatus() bool {
	return o != nil && o.Status != nil
}

// SetStatus gets a reference to the given RUMResponseStatus and assigns it to the Status field.
func (o *RUMResponseMetadata) SetStatus(v RUMResponseStatus) {
	o.Status = &v
}

// GetWarnings returns the Warnings field value if set, zero value otherwise.
func (o *RUMResponseMetadata) GetWarnings() []RUMWarning {
	if o == nil || o.Warnings == nil {
		var ret []RUMWarning
		return ret
	}
	return o.Warnings
}

// GetWarningsOk returns a tuple with the Warnings field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *RUMResponseMetadata) GetWarningsOk() (*[]RUMWarning, bool) {
	if o == nil || o.Warnings == nil {
		return nil, false
	}
	return &o.Warnings, true
}

// HasWarnings returns a boolean if a field has been set.
func (o *RUMResponseMetadata) HasWarnings() bool {
	return o != nil && o.Warnings != nil
}

// SetWarnings gets a reference to the given []RUMWarning and assigns it to the Warnings field.
func (o *RUMResponseMetadata) SetWarnings(v []RUMWarning) {
	o.Warnings = v
}

// MarshalJSON serializes the struct using spec logic.
func (o RUMResponseMetadata) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Elapsed != nil {
		toSerialize["elapsed"] = o.Elapsed
	}
	if o.Page != nil {
		toSerialize["page"] = o.Page
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
func (o *RUMResponseMetadata) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Elapsed   *int64             `json:"elapsed,omitempty"`
		Page      *RUMResponsePage   `json:"page,omitempty"`
		RequestId *string            `json:"request_id,omitempty"`
		Status    *RUMResponseStatus `json:"status,omitempty"`
		Warnings  []RUMWarning       `json:"warnings,omitempty"`
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
	if all.Page != nil && all.Page.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Page = all.Page
	o.RequestId = all.RequestId
	o.Status = all.Status
	o.Warnings = all.Warnings
	return nil
}
