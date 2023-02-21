// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// LogsMetricResponseGroupBy A group by rule.
type LogsMetricResponseGroupBy struct {
	// The path to the value the log-based metric will be aggregated over.
	Path *string `json:"path,omitempty"`
	// Eventual name of the tag that gets created. By default, the path attribute is used as the tag name.
	TagName *string `json:"tag_name,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewLogsMetricResponseGroupBy instantiates a new LogsMetricResponseGroupBy object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewLogsMetricResponseGroupBy() *LogsMetricResponseGroupBy {
	this := LogsMetricResponseGroupBy{}
	return &this
}

// NewLogsMetricResponseGroupByWithDefaults instantiates a new LogsMetricResponseGroupBy object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewLogsMetricResponseGroupByWithDefaults() *LogsMetricResponseGroupBy {
	this := LogsMetricResponseGroupBy{}
	return &this
}

// GetPath returns the Path field value if set, zero value otherwise.
func (o *LogsMetricResponseGroupBy) GetPath() string {
	if o == nil || o.Path == nil {
		var ret string
		return ret
	}
	return *o.Path
}

// GetPathOk returns a tuple with the Path field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsMetricResponseGroupBy) GetPathOk() (*string, bool) {
	if o == nil || o.Path == nil {
		return nil, false
	}
	return o.Path, true
}

// HasPath returns a boolean if a field has been set.
func (o *LogsMetricResponseGroupBy) HasPath() bool {
	return o != nil && o.Path != nil
}

// SetPath gets a reference to the given string and assigns it to the Path field.
func (o *LogsMetricResponseGroupBy) SetPath(v string) {
	o.Path = &v
}

// GetTagName returns the TagName field value if set, zero value otherwise.
func (o *LogsMetricResponseGroupBy) GetTagName() string {
	if o == nil || o.TagName == nil {
		var ret string
		return ret
	}
	return *o.TagName
}

// GetTagNameOk returns a tuple with the TagName field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsMetricResponseGroupBy) GetTagNameOk() (*string, bool) {
	if o == nil || o.TagName == nil {
		return nil, false
	}
	return o.TagName, true
}

// HasTagName returns a boolean if a field has been set.
func (o *LogsMetricResponseGroupBy) HasTagName() bool {
	return o != nil && o.TagName != nil
}

// SetTagName gets a reference to the given string and assigns it to the TagName field.
func (o *LogsMetricResponseGroupBy) SetTagName(v string) {
	o.TagName = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o LogsMetricResponseGroupBy) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Path != nil {
		toSerialize["path"] = o.Path
	}
	if o.TagName != nil {
		toSerialize["tag_name"] = o.TagName
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *LogsMetricResponseGroupBy) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Path    *string `json:"path,omitempty"`
		TagName *string `json:"tag_name,omitempty"`
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
	o.Path = all.Path
	o.TagName = all.TagName
	return nil
}
