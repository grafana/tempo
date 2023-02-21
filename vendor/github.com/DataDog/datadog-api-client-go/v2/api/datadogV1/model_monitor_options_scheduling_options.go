// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV1

import (
	"encoding/json"
)

// MonitorOptionsSchedulingOptions Configuration options for scheduling.
type MonitorOptionsSchedulingOptions struct {
	// Configuration options for the evaluation window. If `hour_starts` is set, no other fields may be set. Otherwise, `day_starts` and `month_starts` must be set together.
	EvaluationWindow *MonitorOptionsSchedulingOptionsEvaluationWindow `json:"evaluation_window,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewMonitorOptionsSchedulingOptions instantiates a new MonitorOptionsSchedulingOptions object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewMonitorOptionsSchedulingOptions() *MonitorOptionsSchedulingOptions {
	this := MonitorOptionsSchedulingOptions{}
	return &this
}

// NewMonitorOptionsSchedulingOptionsWithDefaults instantiates a new MonitorOptionsSchedulingOptions object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewMonitorOptionsSchedulingOptionsWithDefaults() *MonitorOptionsSchedulingOptions {
	this := MonitorOptionsSchedulingOptions{}
	return &this
}

// GetEvaluationWindow returns the EvaluationWindow field value if set, zero value otherwise.
func (o *MonitorOptionsSchedulingOptions) GetEvaluationWindow() MonitorOptionsSchedulingOptionsEvaluationWindow {
	if o == nil || o.EvaluationWindow == nil {
		var ret MonitorOptionsSchedulingOptionsEvaluationWindow
		return ret
	}
	return *o.EvaluationWindow
}

// GetEvaluationWindowOk returns a tuple with the EvaluationWindow field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorOptionsSchedulingOptions) GetEvaluationWindowOk() (*MonitorOptionsSchedulingOptionsEvaluationWindow, bool) {
	if o == nil || o.EvaluationWindow == nil {
		return nil, false
	}
	return o.EvaluationWindow, true
}

// HasEvaluationWindow returns a boolean if a field has been set.
func (o *MonitorOptionsSchedulingOptions) HasEvaluationWindow() bool {
	return o != nil && o.EvaluationWindow != nil
}

// SetEvaluationWindow gets a reference to the given MonitorOptionsSchedulingOptionsEvaluationWindow and assigns it to the EvaluationWindow field.
func (o *MonitorOptionsSchedulingOptions) SetEvaluationWindow(v MonitorOptionsSchedulingOptionsEvaluationWindow) {
	o.EvaluationWindow = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o MonitorOptionsSchedulingOptions) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.EvaluationWindow != nil {
		toSerialize["evaluation_window"] = o.EvaluationWindow
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *MonitorOptionsSchedulingOptions) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		EvaluationWindow *MonitorOptionsSchedulingOptionsEvaluationWindow `json:"evaluation_window,omitempty"`
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
	if all.EvaluationWindow != nil && all.EvaluationWindow.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.EvaluationWindow = all.EvaluationWindow
	return nil
}
