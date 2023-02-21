// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// AuditLogsResponsePage Paging attributes.
type AuditLogsResponsePage struct {
	// The cursor to use to get the next results, if any. To make the next request, use the same parameters with the addition of `page[cursor]`.
	After *string `json:"after,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewAuditLogsResponsePage instantiates a new AuditLogsResponsePage object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewAuditLogsResponsePage() *AuditLogsResponsePage {
	this := AuditLogsResponsePage{}
	return &this
}

// NewAuditLogsResponsePageWithDefaults instantiates a new AuditLogsResponsePage object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewAuditLogsResponsePageWithDefaults() *AuditLogsResponsePage {
	this := AuditLogsResponsePage{}
	return &this
}

// GetAfter returns the After field value if set, zero value otherwise.
func (o *AuditLogsResponsePage) GetAfter() string {
	if o == nil || o.After == nil {
		var ret string
		return ret
	}
	return *o.After
}

// GetAfterOk returns a tuple with the After field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *AuditLogsResponsePage) GetAfterOk() (*string, bool) {
	if o == nil || o.After == nil {
		return nil, false
	}
	return o.After, true
}

// HasAfter returns a boolean if a field has been set.
func (o *AuditLogsResponsePage) HasAfter() bool {
	return o != nil && o.After != nil
}

// SetAfter gets a reference to the given string and assigns it to the After field.
func (o *AuditLogsResponsePage) SetAfter(v string) {
	o.After = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o AuditLogsResponsePage) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.After != nil {
		toSerialize["after"] = o.After
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *AuditLogsResponsePage) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		After *string `json:"after,omitempty"`
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
	o.After = all.After
	return nil
}
