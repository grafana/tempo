// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// MetricBulkTagConfigDeleteAttributes Optional parameters for bulk deleting metric tag configurations.
type MetricBulkTagConfigDeleteAttributes struct {
	// A list of account emails to notify when the configuration is applied.
	Emails []string `json:"emails,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewMetricBulkTagConfigDeleteAttributes instantiates a new MetricBulkTagConfigDeleteAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewMetricBulkTagConfigDeleteAttributes() *MetricBulkTagConfigDeleteAttributes {
	this := MetricBulkTagConfigDeleteAttributes{}
	return &this
}

// NewMetricBulkTagConfigDeleteAttributesWithDefaults instantiates a new MetricBulkTagConfigDeleteAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewMetricBulkTagConfigDeleteAttributesWithDefaults() *MetricBulkTagConfigDeleteAttributes {
	this := MetricBulkTagConfigDeleteAttributes{}
	return &this
}

// GetEmails returns the Emails field value if set, zero value otherwise.
func (o *MetricBulkTagConfigDeleteAttributes) GetEmails() []string {
	if o == nil || o.Emails == nil {
		var ret []string
		return ret
	}
	return o.Emails
}

// GetEmailsOk returns a tuple with the Emails field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MetricBulkTagConfigDeleteAttributes) GetEmailsOk() (*[]string, bool) {
	if o == nil || o.Emails == nil {
		return nil, false
	}
	return &o.Emails, true
}

// HasEmails returns a boolean if a field has been set.
func (o *MetricBulkTagConfigDeleteAttributes) HasEmails() bool {
	return o != nil && o.Emails != nil
}

// SetEmails gets a reference to the given []string and assigns it to the Emails field.
func (o *MetricBulkTagConfigDeleteAttributes) SetEmails(v []string) {
	o.Emails = v
}

// MarshalJSON serializes the struct using spec logic.
func (o MetricBulkTagConfigDeleteAttributes) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Emails != nil {
		toSerialize["emails"] = o.Emails
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *MetricBulkTagConfigDeleteAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Emails []string `json:"emails,omitempty"`
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
	return nil
}
