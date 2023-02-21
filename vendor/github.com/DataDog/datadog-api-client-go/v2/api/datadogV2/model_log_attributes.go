// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"time"
)

// LogAttributes JSON object containing all log attributes and their associated values.
type LogAttributes struct {
	// JSON object of attributes from your log.
	Attributes map[string]interface{} `json:"attributes,omitempty"`
	// Name of the machine from where the logs are being sent.
	Host *string `json:"host,omitempty"`
	// The message [reserved attribute](https://docs.datadoghq.com/logs/log_collection/#reserved-attributes)
	// of your log. By default, Datadog ingests the value of the message attribute as the body of the log entry.
	// That value is then highlighted and displayed in the Logstream, where it is indexed for full text search.
	Message *string `json:"message,omitempty"`
	// The name of the application or service generating the log events.
	// It is used to switch from Logs to APM, so make sure you define the same
	// value when you use both products.
	Service *string `json:"service,omitempty"`
	// Status of the message associated with your log.
	Status *string `json:"status,omitempty"`
	// Array of tags associated with your log.
	Tags []string `json:"tags,omitempty"`
	// Timestamp of your log.
	Timestamp *time.Time `json:"timestamp,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewLogAttributes instantiates a new LogAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewLogAttributes() *LogAttributes {
	this := LogAttributes{}
	return &this
}

// NewLogAttributesWithDefaults instantiates a new LogAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewLogAttributesWithDefaults() *LogAttributes {
	this := LogAttributes{}
	return &this
}

// GetAttributes returns the Attributes field value if set, zero value otherwise.
func (o *LogAttributes) GetAttributes() map[string]interface{} {
	if o == nil || o.Attributes == nil {
		var ret map[string]interface{}
		return ret
	}
	return o.Attributes
}

// GetAttributesOk returns a tuple with the Attributes field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogAttributes) GetAttributesOk() (*map[string]interface{}, bool) {
	if o == nil || o.Attributes == nil {
		return nil, false
	}
	return &o.Attributes, true
}

// HasAttributes returns a boolean if a field has been set.
func (o *LogAttributes) HasAttributes() bool {
	return o != nil && o.Attributes != nil
}

// SetAttributes gets a reference to the given map[string]interface{} and assigns it to the Attributes field.
func (o *LogAttributes) SetAttributes(v map[string]interface{}) {
	o.Attributes = v
}

// GetHost returns the Host field value if set, zero value otherwise.
func (o *LogAttributes) GetHost() string {
	if o == nil || o.Host == nil {
		var ret string
		return ret
	}
	return *o.Host
}

// GetHostOk returns a tuple with the Host field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogAttributes) GetHostOk() (*string, bool) {
	if o == nil || o.Host == nil {
		return nil, false
	}
	return o.Host, true
}

// HasHost returns a boolean if a field has been set.
func (o *LogAttributes) HasHost() bool {
	return o != nil && o.Host != nil
}

// SetHost gets a reference to the given string and assigns it to the Host field.
func (o *LogAttributes) SetHost(v string) {
	o.Host = &v
}

// GetMessage returns the Message field value if set, zero value otherwise.
func (o *LogAttributes) GetMessage() string {
	if o == nil || o.Message == nil {
		var ret string
		return ret
	}
	return *o.Message
}

// GetMessageOk returns a tuple with the Message field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogAttributes) GetMessageOk() (*string, bool) {
	if o == nil || o.Message == nil {
		return nil, false
	}
	return o.Message, true
}

// HasMessage returns a boolean if a field has been set.
func (o *LogAttributes) HasMessage() bool {
	return o != nil && o.Message != nil
}

// SetMessage gets a reference to the given string and assigns it to the Message field.
func (o *LogAttributes) SetMessage(v string) {
	o.Message = &v
}

// GetService returns the Service field value if set, zero value otherwise.
func (o *LogAttributes) GetService() string {
	if o == nil || o.Service == nil {
		var ret string
		return ret
	}
	return *o.Service
}

// GetServiceOk returns a tuple with the Service field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogAttributes) GetServiceOk() (*string, bool) {
	if o == nil || o.Service == nil {
		return nil, false
	}
	return o.Service, true
}

// HasService returns a boolean if a field has been set.
func (o *LogAttributes) HasService() bool {
	return o != nil && o.Service != nil
}

// SetService gets a reference to the given string and assigns it to the Service field.
func (o *LogAttributes) SetService(v string) {
	o.Service = &v
}

// GetStatus returns the Status field value if set, zero value otherwise.
func (o *LogAttributes) GetStatus() string {
	if o == nil || o.Status == nil {
		var ret string
		return ret
	}
	return *o.Status
}

// GetStatusOk returns a tuple with the Status field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogAttributes) GetStatusOk() (*string, bool) {
	if o == nil || o.Status == nil {
		return nil, false
	}
	return o.Status, true
}

// HasStatus returns a boolean if a field has been set.
func (o *LogAttributes) HasStatus() bool {
	return o != nil && o.Status != nil
}

// SetStatus gets a reference to the given string and assigns it to the Status field.
func (o *LogAttributes) SetStatus(v string) {
	o.Status = &v
}

// GetTags returns the Tags field value if set, zero value otherwise.
func (o *LogAttributes) GetTags() []string {
	if o == nil || o.Tags == nil {
		var ret []string
		return ret
	}
	return o.Tags
}

// GetTagsOk returns a tuple with the Tags field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogAttributes) GetTagsOk() (*[]string, bool) {
	if o == nil || o.Tags == nil {
		return nil, false
	}
	return &o.Tags, true
}

// HasTags returns a boolean if a field has been set.
func (o *LogAttributes) HasTags() bool {
	return o != nil && o.Tags != nil
}

// SetTags gets a reference to the given []string and assigns it to the Tags field.
func (o *LogAttributes) SetTags(v []string) {
	o.Tags = v
}

// GetTimestamp returns the Timestamp field value if set, zero value otherwise.
func (o *LogAttributes) GetTimestamp() time.Time {
	if o == nil || o.Timestamp == nil {
		var ret time.Time
		return ret
	}
	return *o.Timestamp
}

// GetTimestampOk returns a tuple with the Timestamp field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogAttributes) GetTimestampOk() (*time.Time, bool) {
	if o == nil || o.Timestamp == nil {
		return nil, false
	}
	return o.Timestamp, true
}

// HasTimestamp returns a boolean if a field has been set.
func (o *LogAttributes) HasTimestamp() bool {
	return o != nil && o.Timestamp != nil
}

// SetTimestamp gets a reference to the given time.Time and assigns it to the Timestamp field.
func (o *LogAttributes) SetTimestamp(v time.Time) {
	o.Timestamp = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o LogAttributes) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Attributes != nil {
		toSerialize["attributes"] = o.Attributes
	}
	if o.Host != nil {
		toSerialize["host"] = o.Host
	}
	if o.Message != nil {
		toSerialize["message"] = o.Message
	}
	if o.Service != nil {
		toSerialize["service"] = o.Service
	}
	if o.Status != nil {
		toSerialize["status"] = o.Status
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
func (o *LogAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Attributes map[string]interface{} `json:"attributes,omitempty"`
		Host       *string                `json:"host,omitempty"`
		Message    *string                `json:"message,omitempty"`
		Service    *string                `json:"service,omitempty"`
		Status     *string                `json:"status,omitempty"`
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
	o.Host = all.Host
	o.Message = all.Message
	o.Service = all.Service
	o.Status = all.Status
	o.Tags = all.Tags
	o.Timestamp = all.Timestamp
	return nil
}
