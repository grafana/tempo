// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// MetricBulkTagConfigStatusAttributes Optional attributes for the status of a bulk tag configuration request.
type MetricBulkTagConfigStatusAttributes struct {
	// A list of account emails to notify when the configuration is applied.
	Emails []string `json:"emails,omitempty"`
	// The status of the request.
	Status *string `json:"status,omitempty"`
	// A list of tag names to apply to the configuration.
	Tags []string `json:"tags,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewMetricBulkTagConfigStatusAttributes instantiates a new MetricBulkTagConfigStatusAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewMetricBulkTagConfigStatusAttributes() *MetricBulkTagConfigStatusAttributes {
	this := MetricBulkTagConfigStatusAttributes{}
	return &this
}

// NewMetricBulkTagConfigStatusAttributesWithDefaults instantiates a new MetricBulkTagConfigStatusAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewMetricBulkTagConfigStatusAttributesWithDefaults() *MetricBulkTagConfigStatusAttributes {
	this := MetricBulkTagConfigStatusAttributes{}
	return &this
}

// GetEmails returns the Emails field value if set, zero value otherwise.
func (o *MetricBulkTagConfigStatusAttributes) GetEmails() []string {
	if o == nil || o.Emails == nil {
		var ret []string
		return ret
	}
	return o.Emails
}

// GetEmailsOk returns a tuple with the Emails field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MetricBulkTagConfigStatusAttributes) GetEmailsOk() (*[]string, bool) {
	if o == nil || o.Emails == nil {
		return nil, false
	}
	return &o.Emails, true
}

// HasEmails returns a boolean if a field has been set.
func (o *MetricBulkTagConfigStatusAttributes) HasEmails() bool {
	return o != nil && o.Emails != nil
}

// SetEmails gets a reference to the given []string and assigns it to the Emails field.
func (o *MetricBulkTagConfigStatusAttributes) SetEmails(v []string) {
	o.Emails = v
}

// GetStatus returns the Status field value if set, zero value otherwise.
func (o *MetricBulkTagConfigStatusAttributes) GetStatus() string {
	if o == nil || o.Status == nil {
		var ret string
		return ret
	}
	return *o.Status
}

// GetStatusOk returns a tuple with the Status field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MetricBulkTagConfigStatusAttributes) GetStatusOk() (*string, bool) {
	if o == nil || o.Status == nil {
		return nil, false
	}
	return o.Status, true
}

// HasStatus returns a boolean if a field has been set.
func (o *MetricBulkTagConfigStatusAttributes) HasStatus() bool {
	return o != nil && o.Status != nil
}

// SetStatus gets a reference to the given string and assigns it to the Status field.
func (o *MetricBulkTagConfigStatusAttributes) SetStatus(v string) {
	o.Status = &v
}

// GetTags returns the Tags field value if set, zero value otherwise.
func (o *MetricBulkTagConfigStatusAttributes) GetTags() []string {
	if o == nil || o.Tags == nil {
		var ret []string
		return ret
	}
	return o.Tags
}

// GetTagsOk returns a tuple with the Tags field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MetricBulkTagConfigStatusAttributes) GetTagsOk() (*[]string, bool) {
	if o == nil || o.Tags == nil {
		return nil, false
	}
	return &o.Tags, true
}

// HasTags returns a boolean if a field has been set.
func (o *MetricBulkTagConfigStatusAttributes) HasTags() bool {
	return o != nil && o.Tags != nil
}

// SetTags gets a reference to the given []string and assigns it to the Tags field.
func (o *MetricBulkTagConfigStatusAttributes) SetTags(v []string) {
	o.Tags = v
}

// MarshalJSON serializes the struct using spec logic.
func (o MetricBulkTagConfigStatusAttributes) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Emails != nil {
		toSerialize["emails"] = o.Emails
	}
	if o.Status != nil {
		toSerialize["status"] = o.Status
	}
	if o.Tags != nil {
		toSerialize["tags"] = o.Tags
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *MetricBulkTagConfigStatusAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Emails []string `json:"emails,omitempty"`
		Status *string  `json:"status,omitempty"`
		Tags   []string `json:"tags,omitempty"`
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
	o.Emails = all.Emails
	o.Status = all.Status
	o.Tags = all.Tags
	return nil
}
