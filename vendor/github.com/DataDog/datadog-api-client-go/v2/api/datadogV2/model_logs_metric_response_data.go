// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// LogsMetricResponseData The log-based metric properties.
type LogsMetricResponseData struct {
	// The object describing a Datadog log-based metric.
	Attributes *LogsMetricResponseAttributes `json:"attributes,omitempty"`
	// The name of the log-based metric.
	Id *string `json:"id,omitempty"`
	// The type of the resource. The value should always be logs_metrics.
	Type *LogsMetricType `json:"type,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewLogsMetricResponseData instantiates a new LogsMetricResponseData object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewLogsMetricResponseData() *LogsMetricResponseData {
	this := LogsMetricResponseData{}
	var typeVar LogsMetricType = LOGSMETRICTYPE_LOGS_METRICS
	this.Type = &typeVar
	return &this
}

// NewLogsMetricResponseDataWithDefaults instantiates a new LogsMetricResponseData object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewLogsMetricResponseDataWithDefaults() *LogsMetricResponseData {
	this := LogsMetricResponseData{}
	var typeVar LogsMetricType = LOGSMETRICTYPE_LOGS_METRICS
	this.Type = &typeVar
	return &this
}

// GetAttributes returns the Attributes field value if set, zero value otherwise.
func (o *LogsMetricResponseData) GetAttributes() LogsMetricResponseAttributes {
	if o == nil || o.Attributes == nil {
		var ret LogsMetricResponseAttributes
		return ret
	}
	return *o.Attributes
}

// GetAttributesOk returns a tuple with the Attributes field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsMetricResponseData) GetAttributesOk() (*LogsMetricResponseAttributes, bool) {
	if o == nil || o.Attributes == nil {
		return nil, false
	}
	return o.Attributes, true
}

// HasAttributes returns a boolean if a field has been set.
func (o *LogsMetricResponseData) HasAttributes() bool {
	return o != nil && o.Attributes != nil
}

// SetAttributes gets a reference to the given LogsMetricResponseAttributes and assigns it to the Attributes field.
func (o *LogsMetricResponseData) SetAttributes(v LogsMetricResponseAttributes) {
	o.Attributes = &v
}

// GetId returns the Id field value if set, zero value otherwise.
func (o *LogsMetricResponseData) GetId() string {
	if o == nil || o.Id == nil {
		var ret string
		return ret
	}
	return *o.Id
}

// GetIdOk returns a tuple with the Id field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsMetricResponseData) GetIdOk() (*string, bool) {
	if o == nil || o.Id == nil {
		return nil, false
	}
	return o.Id, true
}

// HasId returns a boolean if a field has been set.
func (o *LogsMetricResponseData) HasId() bool {
	return o != nil && o.Id != nil
}

// SetId gets a reference to the given string and assigns it to the Id field.
func (o *LogsMetricResponseData) SetId(v string) {
	o.Id = &v
}

// GetType returns the Type field value if set, zero value otherwise.
func (o *LogsMetricResponseData) GetType() LogsMetricType {
	if o == nil || o.Type == nil {
		var ret LogsMetricType
		return ret
	}
	return *o.Type
}

// GetTypeOk returns a tuple with the Type field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsMetricResponseData) GetTypeOk() (*LogsMetricType, bool) {
	if o == nil || o.Type == nil {
		return nil, false
	}
	return o.Type, true
}

// HasType returns a boolean if a field has been set.
func (o *LogsMetricResponseData) HasType() bool {
	return o != nil && o.Type != nil
}

// SetType gets a reference to the given LogsMetricType and assigns it to the Type field.
func (o *LogsMetricResponseData) SetType(v LogsMetricType) {
	o.Type = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o LogsMetricResponseData) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Attributes != nil {
		toSerialize["attributes"] = o.Attributes
	}
	if o.Id != nil {
		toSerialize["id"] = o.Id
	}
	if o.Type != nil {
		toSerialize["type"] = o.Type
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *LogsMetricResponseData) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Attributes *LogsMetricResponseAttributes `json:"attributes,omitempty"`
		Id         *string                       `json:"id,omitempty"`
		Type       *LogsMetricType               `json:"type,omitempty"`
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
	if v := all.Type; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	if all.Attributes != nil && all.Attributes.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Attributes = all.Attributes
	o.Id = all.Id
	o.Type = all.Type
	return nil
}
