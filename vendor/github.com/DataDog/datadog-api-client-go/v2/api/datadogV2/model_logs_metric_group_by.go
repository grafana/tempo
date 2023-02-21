// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// LogsMetricGroupBy A group by rule.
type LogsMetricGroupBy struct {
	// The path to the value the log-based metric will be aggregated over.
	Path string `json:"path"`
	// Eventual name of the tag that gets created. By default, the path attribute is used as the tag name.
	TagName *string `json:"tag_name,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewLogsMetricGroupBy instantiates a new LogsMetricGroupBy object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewLogsMetricGroupBy(path string) *LogsMetricGroupBy {
	this := LogsMetricGroupBy{}
	this.Path = path
	return &this
}

// NewLogsMetricGroupByWithDefaults instantiates a new LogsMetricGroupBy object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewLogsMetricGroupByWithDefaults() *LogsMetricGroupBy {
	this := LogsMetricGroupBy{}
	return &this
}

// GetPath returns the Path field value.
func (o *LogsMetricGroupBy) GetPath() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Path
}

// GetPathOk returns a tuple with the Path field value
// and a boolean to check if the value has been set.
func (o *LogsMetricGroupBy) GetPathOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Path, true
}

// SetPath sets field value.
func (o *LogsMetricGroupBy) SetPath(v string) {
	o.Path = v
}

// GetTagName returns the TagName field value if set, zero value otherwise.
func (o *LogsMetricGroupBy) GetTagName() string {
	if o == nil || o.TagName == nil {
		var ret string
		return ret
	}
	return *o.TagName
}

// GetTagNameOk returns a tuple with the TagName field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsMetricGroupBy) GetTagNameOk() (*string, bool) {
	if o == nil || o.TagName == nil {
		return nil, false
	}
	return o.TagName, true
}

// HasTagName returns a boolean if a field has been set.
func (o *LogsMetricGroupBy) HasTagName() bool {
	return o != nil && o.TagName != nil
}

// SetTagName gets a reference to the given string and assigns it to the TagName field.
func (o *LogsMetricGroupBy) SetTagName(v string) {
	o.TagName = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o LogsMetricGroupBy) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["path"] = o.Path
	if o.TagName != nil {
		toSerialize["tag_name"] = o.TagName
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *LogsMetricGroupBy) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Path *string `json:"path"`
	}{}
	all := struct {
		Path    string  `json:"path"`
		TagName *string `json:"tag_name,omitempty"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Path == nil {
		return fmt.Errorf("required field path missing")
	}
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
