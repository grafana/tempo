// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV1

import (
	"encoding/json"
	"fmt"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
)

// SearchSLOThreshold SLO thresholds (target and optionally warning) for a single time window.
type SearchSLOThreshold struct {
	// The target value for the service level indicator within the corresponding
	// timeframe.
	Target float64 `json:"target"`
	// A string representation of the target that indicates its precision.
	// It uses trailing zeros to show significant decimal places (for example `98.00`).
	//
	// Always included in service level objective responses. Ignored in
	// create/update requests.
	TargetDisplay *string `json:"target_display,omitempty"`
	// The SLO time window options.
	Timeframe SearchSLOTimeframe `json:"timeframe"`
	// The warning value for the service level objective.
	Warning datadog.NullableFloat64 `json:"warning,omitempty"`
	// A string representation of the warning target (see the description of
	// the `target_display` field for details).
	//
	// Included in service level objective responses if a warning target exists.
	// Ignored in create/update requests.
	WarningDisplay datadog.NullableString `json:"warning_display,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewSearchSLOThreshold instantiates a new SearchSLOThreshold object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSearchSLOThreshold(target float64, timeframe SearchSLOTimeframe) *SearchSLOThreshold {
	this := SearchSLOThreshold{}
	this.Target = target
	this.Timeframe = timeframe
	return &this
}

// NewSearchSLOThresholdWithDefaults instantiates a new SearchSLOThreshold object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSearchSLOThresholdWithDefaults() *SearchSLOThreshold {
	this := SearchSLOThreshold{}
	return &this
}

// GetTarget returns the Target field value.
func (o *SearchSLOThreshold) GetTarget() float64 {
	if o == nil {
		var ret float64
		return ret
	}
	return o.Target
}

// GetTargetOk returns a tuple with the Target field value
// and a boolean to check if the value has been set.
func (o *SearchSLOThreshold) GetTargetOk() (*float64, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Target, true
}

// SetTarget sets field value.
func (o *SearchSLOThreshold) SetTarget(v float64) {
	o.Target = v
}

// GetTargetDisplay returns the TargetDisplay field value if set, zero value otherwise.
func (o *SearchSLOThreshold) GetTargetDisplay() string {
	if o == nil || o.TargetDisplay == nil {
		var ret string
		return ret
	}
	return *o.TargetDisplay
}

// GetTargetDisplayOk returns a tuple with the TargetDisplay field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SearchSLOThreshold) GetTargetDisplayOk() (*string, bool) {
	if o == nil || o.TargetDisplay == nil {
		return nil, false
	}
	return o.TargetDisplay, true
}

// HasTargetDisplay returns a boolean if a field has been set.
func (o *SearchSLOThreshold) HasTargetDisplay() bool {
	return o != nil && o.TargetDisplay != nil
}

// SetTargetDisplay gets a reference to the given string and assigns it to the TargetDisplay field.
func (o *SearchSLOThreshold) SetTargetDisplay(v string) {
	o.TargetDisplay = &v
}

// GetTimeframe returns the Timeframe field value.
func (o *SearchSLOThreshold) GetTimeframe() SearchSLOTimeframe {
	if o == nil {
		var ret SearchSLOTimeframe
		return ret
	}
	return o.Timeframe
}

// GetTimeframeOk returns a tuple with the Timeframe field value
// and a boolean to check if the value has been set.
func (o *SearchSLOThreshold) GetTimeframeOk() (*SearchSLOTimeframe, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Timeframe, true
}

// SetTimeframe sets field value.
func (o *SearchSLOThreshold) SetTimeframe(v SearchSLOTimeframe) {
	o.Timeframe = v
}

// GetWarning returns the Warning field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *SearchSLOThreshold) GetWarning() float64 {
	if o == nil || o.Warning.Get() == nil {
		var ret float64
		return ret
	}
	return *o.Warning.Get()
}

// GetWarningOk returns a tuple with the Warning field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *SearchSLOThreshold) GetWarningOk() (*float64, bool) {
	if o == nil {
		return nil, false
	}
	return o.Warning.Get(), o.Warning.IsSet()
}

// HasWarning returns a boolean if a field has been set.
func (o *SearchSLOThreshold) HasWarning() bool {
	return o != nil && o.Warning.IsSet()
}

// SetWarning gets a reference to the given datadog.NullableFloat64 and assigns it to the Warning field.
func (o *SearchSLOThreshold) SetWarning(v float64) {
	o.Warning.Set(&v)
}

// SetWarningNil sets the value for Warning to be an explicit nil.
func (o *SearchSLOThreshold) SetWarningNil() {
	o.Warning.Set(nil)
}

// UnsetWarning ensures that no value is present for Warning, not even an explicit nil.
func (o *SearchSLOThreshold) UnsetWarning() {
	o.Warning.Unset()
}

// GetWarningDisplay returns the WarningDisplay field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *SearchSLOThreshold) GetWarningDisplay() string {
	if o == nil || o.WarningDisplay.Get() == nil {
		var ret string
		return ret
	}
	return *o.WarningDisplay.Get()
}

// GetWarningDisplayOk returns a tuple with the WarningDisplay field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *SearchSLOThreshold) GetWarningDisplayOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return o.WarningDisplay.Get(), o.WarningDisplay.IsSet()
}

// HasWarningDisplay returns a boolean if a field has been set.
func (o *SearchSLOThreshold) HasWarningDisplay() bool {
	return o != nil && o.WarningDisplay.IsSet()
}

// SetWarningDisplay gets a reference to the given datadog.NullableString and assigns it to the WarningDisplay field.
func (o *SearchSLOThreshold) SetWarningDisplay(v string) {
	o.WarningDisplay.Set(&v)
}

// SetWarningDisplayNil sets the value for WarningDisplay to be an explicit nil.
func (o *SearchSLOThreshold) SetWarningDisplayNil() {
	o.WarningDisplay.Set(nil)
}

// UnsetWarningDisplay ensures that no value is present for WarningDisplay, not even an explicit nil.
func (o *SearchSLOThreshold) UnsetWarningDisplay() {
	o.WarningDisplay.Unset()
}

// MarshalJSON serializes the struct using spec logic.
func (o SearchSLOThreshold) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["target"] = o.Target
	if o.TargetDisplay != nil {
		toSerialize["target_display"] = o.TargetDisplay
	}
	toSerialize["timeframe"] = o.Timeframe
	if o.Warning.IsSet() {
		toSerialize["warning"] = o.Warning.Get()
	}
	if o.WarningDisplay.IsSet() {
		toSerialize["warning_display"] = o.WarningDisplay.Get()
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SearchSLOThreshold) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Target    *float64            `json:"target"`
		Timeframe *SearchSLOTimeframe `json:"timeframe"`
	}{}
	all := struct {
		Target         float64                 `json:"target"`
		TargetDisplay  *string                 `json:"target_display,omitempty"`
		Timeframe      SearchSLOTimeframe      `json:"timeframe"`
		Warning        datadog.NullableFloat64 `json:"warning,omitempty"`
		WarningDisplay datadog.NullableString  `json:"warning_display,omitempty"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Target == nil {
		return fmt.Errorf("required field target missing")
	}
	if required.Timeframe == nil {
		return fmt.Errorf("required field timeframe missing")
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
	if v := all.Timeframe; !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.Target = all.Target
	o.TargetDisplay = all.TargetDisplay
	o.Timeframe = all.Timeframe
	o.Warning = all.Warning
	o.WarningDisplay = all.WarningDisplay
	return nil
}
