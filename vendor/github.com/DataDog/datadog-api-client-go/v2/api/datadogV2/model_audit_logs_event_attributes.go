// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"time"
)

// AuditLogsEventAttributes JSON object containing all event attributes and their associated values.
type AuditLogsEventAttributes struct {
	// JSON object of attributes from Audit Logs events.
	Attributes map[string]interface{} `json:"attributes,omitempty"`
	// Name of the application or service generating Audit Logs events.
	// This name is used to correlate Audit Logs to APM, so make sure you specify the same
	// value when you use both products.
	Service *string `json:"service,omitempty"`
	// Array of tags associated with your event.
	Tags []string `json:"tags,omitempty"`
	// Timestamp of your event.
	Timestamp *time.Time `json:"timestamp,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewAuditLogsEventAttributes instantiates a new AuditLogsEventAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewAuditLogsEventAttributes() *AuditLogsEventAttributes {
	this := AuditLogsEventAttributes{}
	return &this
}

// NewAuditLogsEventAttributesWithDefaults instantiates a new AuditLogsEventAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewAuditLogsEventAttributesWithDefaults() *AuditLogsEventAttributes {
	this := AuditLogsEventAttributes{}
	return &this
}

// GetAttributes returns the Attributes field value if set, zero value otherwise.
func (o *AuditLogsEventAttributes) GetAttributes() map[string]interface{} {
	if o == nil || o.Attributes == nil {
		var ret map[string]interface{}
		return ret
	}
	return o.Attributes
}

// GetAttributesOk returns a tuple with the Attributes field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *AuditLogsEventAttributes) GetAttributesOk() (*map[string]interface{}, bool) {
	if o == nil || o.Attributes == nil {
		return nil, false
	}
	return &o.Attributes, true
}

// HasAttributes returns a boolean if a field has been set.
func (o *AuditLogsEventAttributes) HasAttributes() bool {
	return o != nil && o.Attributes != nil
}

// SetAttributes gets a reference to the given map[string]interface{} and assigns it to the Attributes field.
func (o *AuditLogsEventAttributes) SetAttributes(v map[string]interface{}) {
	o.Attributes = v
}

// GetService returns the Service field value if set, zero value otherwise.
func (o *AuditLogsEventAttributes) GetService() string {
	if o == nil || o.Service == nil {
		var ret string
		return ret
	}
	return *o.Service
}

// GetServiceOk returns a tuple with the Service field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *AuditLogsEventAttributes) GetServiceOk() (*string, bool) {
	if o == nil || o.Service == nil {
		return nil, false
	}
	return o.Service, true
}

// HasService returns a boolean if a field has been set.
func (o *AuditLogsEventAttributes) HasService() bool {
	return o != nil && o.Service != nil
}

// SetService gets a reference to the given string and assigns it to the Service field.
func (o *AuditLogsEventAttributes) SetService(v string) {
	o.Service = &v
}

// GetTags returns the Tags field value if set, zero value otherwise.
func (o *AuditLogsEventAttributes) GetTags() []string {
	if o == nil || o.Tags == nil {
		var ret []string
		return ret
	}
	return o.Tags
}

// GetTagsOk returns a tuple with the Tags field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *AuditLogsEventAttributes) GetTagsOk() (*[]string, bool) {
	if o == nil || o.Tags == nil {
		return nil, false
	}
	return &o.Tags, true
}

// HasTags returns a boolean if a field has been set.
func (o *AuditLogsEventAttributes) HasTags() bool {
	return o != nil && o.Tags != nil
}

// SetTags gets a reference to the given []string and assigns it to the Tags field.
func (o *AuditLogsEventAttributes) SetTags(v []string) {
	o.Tags = v
}

// GetTimestamp returns the Timestamp field value if set, zero value otherwise.
func (o *AuditLogsEventAttributes) GetTimestamp() time.Time {
	if o == nil || o.Timestamp == nil {
		var ret time.Time
		return ret
	}
	return *o.Timestamp
}

// GetTimestampOk returns a tuple with the Timestamp field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *AuditLogsEventAttributes) GetTimestampOk() (*time.Time, bool) {
	if o == nil || o.Timestamp == nil {
		return nil, false
	}
	return o.Timestamp, true
}

// HasTimestamp returns a boolean if a field has been set.
func (o *AuditLogsEventAttributes) HasTimestamp() bool {
	return o != nil && o.Timestamp != nil
}

// SetTimestamp gets a reference to the given time.Time and assigns it to the Timestamp field.
func (o *AuditLogsEventAttributes) SetTimestamp(v time.Time) {
	o.Timestamp = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o AuditLogsEventAttributes) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Attributes != nil {
		toSerialize["attributes"] = o.Attributes
	}
	if o.Service != nil {
		toSerialize["service"] = o.Service
	}
	if o.Tags != nil {
		toSerialize["tags"] = o.Tags
	}
	if o.Timestamp != nil {
		if o.Timestamp.Nanosecond() == 0 {
			toSerialize["timestamp"] = o.Timestamp.Format("2006-01-02T15:04:05Z07:00")
		} else {
			toSerialize["timestamp"] = o.Timestamp.Format("2006-01-02T15:04:05.000Z07:00")
		}
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *AuditLogsEventAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Attributes map[string]interface{} `json:"attributes,omitempty"`
		Service    *string                `json:"service,omitempty"`
		Tags       []string               `json:"tags,omitempty"`
		Timestamp  *time.Time             `json:"timestamp,omitempty"`
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
	o.Attributes = all.Attributes
	o.Service = all.Service
	o.Tags = all.Tags
	o.Timestamp = all.Timestamp
	return nil
}
