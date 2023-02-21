// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// EventsWarning A warning message indicating something is wrong with the query.
type EventsWarning struct {
	// A unique code for this type of warning.
	Code *string `json:"code,omitempty"`
	// A detailed explanation of this specific warning.
	Detail *string `json:"detail,omitempty"`
	// A short human-readable summary of the warning.
	Title *string `json:"title,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewEventsWarning instantiates a new EventsWarning object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewEventsWarning() *EventsWarning {
	this := EventsWarning{}
	return &this
}

// NewEventsWarningWithDefaults instantiates a new EventsWarning object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewEventsWarningWithDefaults() *EventsWarning {
	this := EventsWarning{}
	return &this
}

// GetCode returns the Code field value if set, zero value otherwise.
func (o *EventsWarning) GetCode() string {
	if o == nil || o.Code == nil {
		var ret string
		return ret
	}
	return *o.Code
}

// GetCodeOk returns a tuple with the Code field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *EventsWarning) GetCodeOk() (*string, bool) {
	if o == nil || o.Code == nil {
		return nil, false
	}
	return o.Code, true
}

// HasCode returns a boolean if a field has been set.
func (o *EventsWarning) HasCode() bool {
	return o != nil && o.Code != nil
}

// SetCode gets a reference to the given string and assigns it to the Code field.
func (o *EventsWarning) SetCode(v string) {
	o.Code = &v
}

// GetDetail returns the Detail field value if set, zero value otherwise.
func (o *EventsWarning) GetDetail() string {
	if o == nil || o.Detail == nil {
		var ret string
		return ret
	}
	return *o.Detail
}

// GetDetailOk returns a tuple with the Detail field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *EventsWarning) GetDetailOk() (*string, bool) {
	if o == nil || o.Detail == nil {
		return nil, false
	}
	return o.Detail, true
}

// HasDetail returns a boolean if a field has been set.
func (o *EventsWarning) HasDetail() bool {
	return o != nil && o.Detail != nil
}

// SetDetail gets a reference to the given string and assigns it to the Detail field.
func (o *EventsWarning) SetDetail(v string) {
	o.Detail = &v
}

// GetTitle returns the Title field value if set, zero value otherwise.
func (o *EventsWarning) GetTitle() string {
	if o == nil || o.Title == nil {
		var ret string
		return ret
	}
	return *o.Title
}

// GetTitleOk returns a tuple with the Title field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *EventsWarning) GetTitleOk() (*string, bool) {
	if o == nil || o.Title == nil {
		return nil, false
	}
	return o.Title, true
}

// HasTitle returns a boolean if a field has been set.
func (o *EventsWarning) HasTitle() bool {
	return o != nil && o.Title != nil
}

// SetTitle gets a reference to the given string and assigns it to the Title field.
func (o *EventsWarning) SetTitle(v string) {
	o.Title = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o EventsWarning) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Code != nil {
		toSerialize["code"] = o.Code
	}
	if o.Detail != nil {
		toSerialize["detail"] = o.Detail
	}
	if o.Title != nil {
		toSerialize["title"] = o.Title
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *EventsWarning) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Code   *string `json:"code,omitempty"`
		Detail *string `json:"detail,omitempty"`
		Title  *string `json:"title,omitempty"`
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
	o.Code = all.Code
	o.Detail = all.Detail
	o.Title = all.Title
	return nil
}
