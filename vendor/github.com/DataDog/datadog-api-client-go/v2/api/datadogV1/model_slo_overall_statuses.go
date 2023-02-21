// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV1

import (
	"encoding/json"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
)

// SLOOverallStatuses Overall status of the SLO by timeframes.
type SLOOverallStatuses struct {
	// Error message if SLO status or error budget could not be calculated.
	Error datadog.NullableString `json:"error,omitempty"`
	// Remaining error budget of the SLO in percentage.
	ErrorBudgetRemaining datadog.NullableFloat64 `json:"error_budget_remaining,omitempty"`
	// timestamp (UNIX time in seconds) of when the SLO status and error budget
	// were calculated.
	IndexedAt *int64 `json:"indexed_at,omitempty"`
	// Error budget remaining for an SLO.
	RawErrorBudgetRemaining NullableSLORawErrorBudgetRemaining `json:"raw_error_budget_remaining,omitempty"`
	// The amount of decimal places the SLI value is accurate to.
	SpanPrecision datadog.NullableInt64 `json:"span_precision,omitempty"`
	// State of the SLO.
	State *SLOState `json:"state,omitempty"`
	// The status of the SLO.
	Status datadog.NullableFloat64 `json:"status,omitempty"`
	// The target of the SLO.
	Target *float64 `json:"target,omitempty"`
	// The SLO time window options.
	Timeframe *SLOTimeframe `json:"timeframe,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewSLOOverallStatuses instantiates a new SLOOverallStatuses object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSLOOverallStatuses() *SLOOverallStatuses {
	this := SLOOverallStatuses{}
	return &this
}

// NewSLOOverallStatusesWithDefaults instantiates a new SLOOverallStatuses object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSLOOverallStatusesWithDefaults() *SLOOverallStatuses {
	this := SLOOverallStatuses{}
	return &this
}

// GetError returns the Error field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *SLOOverallStatuses) GetError() string {
	if o == nil || o.Error.Get() == nil {
		var ret string
		return ret
	}
	return *o.Error.Get()
}

// GetErrorOk returns a tuple with the Error field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *SLOOverallStatuses) GetErrorOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return o.Error.Get(), o.Error.IsSet()
}

// HasError returns a boolean if a field has been set.
func (o *SLOOverallStatuses) HasError() bool {
	return o != nil && o.Error.IsSet()
}

// SetError gets a reference to the given datadog.NullableString and assigns it to the Error field.
func (o *SLOOverallStatuses) SetError(v string) {
	o.Error.Set(&v)
}

// SetErrorNil sets the value for Error to be an explicit nil.
func (o *SLOOverallStatuses) SetErrorNil() {
	o.Error.Set(nil)
}

// UnsetError ensures that no value is present for Error, not even an explicit nil.
func (o *SLOOverallStatuses) UnsetError() {
	o.Error.Unset()
}

// GetErrorBudgetRemaining returns the ErrorBudgetRemaining field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *SLOOverallStatuses) GetErrorBudgetRemaining() float64 {
	if o == nil || o.ErrorBudgetRemaining.Get() == nil {
		var ret float64
		return ret
	}
	return *o.ErrorBudgetRemaining.Get()
}

// GetErrorBudgetRemainingOk returns a tuple with the ErrorBudgetRemaining field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *SLOOverallStatuses) GetErrorBudgetRemainingOk() (*float64, bool) {
	if o == nil {
		return nil, false
	}
	return o.ErrorBudgetRemaining.Get(), o.ErrorBudgetRemaining.IsSet()
}

// HasErrorBudgetRemaining returns a boolean if a field has been set.
func (o *SLOOverallStatuses) HasErrorBudgetRemaining() bool {
	return o != nil && o.ErrorBudgetRemaining.IsSet()
}

// SetErrorBudgetRemaining gets a reference to the given datadog.NullableFloat64 and assigns it to the ErrorBudgetRemaining field.
func (o *SLOOverallStatuses) SetErrorBudgetRemaining(v float64) {
	o.ErrorBudgetRemaining.Set(&v)
}

// SetErrorBudgetRemainingNil sets the value for ErrorBudgetRemaining to be an explicit nil.
func (o *SLOOverallStatuses) SetErrorBudgetRemainingNil() {
	o.ErrorBudgetRemaining.Set(nil)
}

// UnsetErrorBudgetRemaining ensures that no value is present for ErrorBudgetRemaining, not even an explicit nil.
func (o *SLOOverallStatuses) UnsetErrorBudgetRemaining() {
	o.ErrorBudgetRemaining.Unset()
}

// GetIndexedAt returns the IndexedAt field value if set, zero value otherwise.
func (o *SLOOverallStatuses) GetIndexedAt() int64 {
	if o == nil || o.IndexedAt == nil {
		var ret int64
		return ret
	}
	return *o.IndexedAt
}

// GetIndexedAtOk returns a tuple with the IndexedAt field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOOverallStatuses) GetIndexedAtOk() (*int64, bool) {
	if o == nil || o.IndexedAt == nil {
		return nil, false
	}
	return o.IndexedAt, true
}

// HasIndexedAt returns a boolean if a field has been set.
func (o *SLOOverallStatuses) HasIndexedAt() bool {
	return o != nil && o.IndexedAt != nil
}

// SetIndexedAt gets a reference to the given int64 and assigns it to the IndexedAt field.
func (o *SLOOverallStatuses) SetIndexedAt(v int64) {
	o.IndexedAt = &v
}

// GetRawErrorBudgetRemaining returns the RawErrorBudgetRemaining field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *SLOOverallStatuses) GetRawErrorBudgetRemaining() SLORawErrorBudgetRemaining {
	if o == nil || o.RawErrorBudgetRemaining.Get() == nil {
		var ret SLORawErrorBudgetRemaining
		return ret
	}
	return *o.RawErrorBudgetRemaining.Get()
}

// GetRawErrorBudgetRemainingOk returns a tuple with the RawErrorBudgetRemaining field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *SLOOverallStatuses) GetRawErrorBudgetRemainingOk() (*SLORawErrorBudgetRemaining, bool) {
	if o == nil {
		return nil, false
	}
	return o.RawErrorBudgetRemaining.Get(), o.RawErrorBudgetRemaining.IsSet()
}

// HasRawErrorBudgetRemaining returns a boolean if a field has been set.
func (o *SLOOverallStatuses) HasRawErrorBudgetRemaining() bool {
	return o != nil && o.RawErrorBudgetRemaining.IsSet()
}

// SetRawErrorBudgetRemaining gets a reference to the given NullableSLORawErrorBudgetRemaining and assigns it to the RawErrorBudgetRemaining field.
func (o *SLOOverallStatuses) SetRawErrorBudgetRemaining(v SLORawErrorBudgetRemaining) {
	o.RawErrorBudgetRemaining.Set(&v)
}

// SetRawErrorBudgetRemainingNil sets the value for RawErrorBudgetRemaining to be an explicit nil.
func (o *SLOOverallStatuses) SetRawErrorBudgetRemainingNil() {
	o.RawErrorBudgetRemaining.Set(nil)
}

// UnsetRawErrorBudgetRemaining ensures that no value is present for RawErrorBudgetRemaining, not even an explicit nil.
func (o *SLOOverallStatuses) UnsetRawErrorBudgetRemaining() {
	o.RawErrorBudgetRemaining.Unset()
}

// GetSpanPrecision returns the SpanPrecision field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *SLOOverallStatuses) GetSpanPrecision() int64 {
	if o == nil || o.SpanPrecision.Get() == nil {
		var ret int64
		return ret
	}
	return *o.SpanPrecision.Get()
}

// GetSpanPrecisionOk returns a tuple with the SpanPrecision field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *SLOOverallStatuses) GetSpanPrecisionOk() (*int64, bool) {
	if o == nil {
		return nil, false
	}
	return o.SpanPrecision.Get(), o.SpanPrecision.IsSet()
}

// HasSpanPrecision returns a boolean if a field has been set.
func (o *SLOOverallStatuses) HasSpanPrecision() bool {
	return o != nil && o.SpanPrecision.IsSet()
}

// SetSpanPrecision gets a reference to the given datadog.NullableInt64 and assigns it to the SpanPrecision field.
func (o *SLOOverallStatuses) SetSpanPrecision(v int64) {
	o.SpanPrecision.Set(&v)
}

// SetSpanPrecisionNil sets the value for SpanPrecision to be an explicit nil.
func (o *SLOOverallStatuses) SetSpanPrecisionNil() {
	o.SpanPrecision.Set(nil)
}

// UnsetSpanPrecision ensures that no value is present for SpanPrecision, not even an explicit nil.
func (o *SLOOverallStatuses) UnsetSpanPrecision() {
	o.SpanPrecision.Unset()
}

// GetState returns the State field value if set, zero value otherwise.
func (o *SLOOverallStatuses) GetState() SLOState {
	if o == nil || o.State == nil {
		var ret SLOState
		return ret
	}
	return *o.State
}

// GetStateOk returns a tuple with the State field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOOverallStatuses) GetStateOk() (*SLOState, bool) {
	if o == nil || o.State == nil {
		return nil, false
	}
	return o.State, true
}

// HasState returns a boolean if a field has been set.
func (o *SLOOverallStatuses) HasState() bool {
	return o != nil && o.State != nil
}

// SetState gets a reference to the given SLOState and assigns it to the State field.
func (o *SLOOverallStatuses) SetState(v SLOState) {
	o.State = &v
}

// GetStatus returns the Status field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *SLOOverallStatuses) GetStatus() float64 {
	if o == nil || o.Status.Get() == nil {
		var ret float64
		return ret
	}
	return *o.Status.Get()
}

// GetStatusOk returns a tuple with the Status field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *SLOOverallStatuses) GetStatusOk() (*float64, bool) {
	if o == nil {
		return nil, false
	}
	return o.Status.Get(), o.Status.IsSet()
}

// HasStatus returns a boolean if a field has been set.
func (o *SLOOverallStatuses) HasStatus() bool {
	return o != nil && o.Status.IsSet()
}

// SetStatus gets a reference to the given datadog.NullableFloat64 and assigns it to the Status field.
func (o *SLOOverallStatuses) SetStatus(v float64) {
	o.Status.Set(&v)
}

// SetStatusNil sets the value for Status to be an explicit nil.
func (o *SLOOverallStatuses) SetStatusNil() {
	o.Status.Set(nil)
}

// UnsetStatus ensures that no value is present for Status, not even an explicit nil.
func (o *SLOOverallStatuses) UnsetStatus() {
	o.Status.Unset()
}

// GetTarget returns the Target field value if set, zero value otherwise.
func (o *SLOOverallStatuses) GetTarget() float64 {
	if o == nil || o.Target == nil {
		var ret float64
		return ret
	}
	return *o.Target
}

// GetTargetOk returns a tuple with the Target field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOOverallStatuses) GetTargetOk() (*float64, bool) {
	if o == nil || o.Target == nil {
		return nil, false
	}
	return o.Target, true
}

// HasTarget returns a boolean if a field has been set.
func (o *SLOOverallStatuses) HasTarget() bool {
	return o != nil && o.Target != nil
}

// SetTarget gets a reference to the given float64 and assigns it to the Target field.
func (o *SLOOverallStatuses) SetTarget(v float64) {
	o.Target = &v
}

// GetTimeframe returns the Timeframe field value if set, zero value otherwise.
func (o *SLOOverallStatuses) GetTimeframe() SLOTimeframe {
	if o == nil || o.Timeframe == nil {
		var ret SLOTimeframe
		return ret
	}
	return *o.Timeframe
}

// GetTimeframeOk returns a tuple with the Timeframe field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOOverallStatuses) GetTimeframeOk() (*SLOTimeframe, bool) {
	if o == nil || o.Timeframe == nil {
		return nil, false
	}
	return o.Timeframe, true
}

// HasTimeframe returns a boolean if a field has been set.
func (o *SLOOverallStatuses) HasTimeframe() bool {
	return o != nil && o.Timeframe != nil
}

// SetTimeframe gets a reference to the given SLOTimeframe and assigns it to the Timeframe field.
func (o *SLOOverallStatuses) SetTimeframe(v SLOTimeframe) {
	o.Timeframe = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SLOOverallStatuses) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Error.IsSet() {
		toSerialize["error"] = o.Error.Get()
	}
	if o.ErrorBudgetRemaining.IsSet() {
		toSerialize["error_budget_remaining"] = o.ErrorBudgetRemaining.Get()
	}
	if o.IndexedAt != nil {
		toSerialize["indexed_at"] = o.IndexedAt
	}
	if o.RawErrorBudgetRemaining.IsSet() {
		toSerialize["raw_error_budget_remaining"] = o.RawErrorBudgetRemaining.Get()
	}
	if o.SpanPrecision.IsSet() {
		toSerialize["span_precision"] = o.SpanPrecision.Get()
	}
	if o.State != nil {
		toSerialize["state"] = o.State
	}
	if o.Status.IsSet() {
		toSerialize["status"] = o.Status.Get()
	}
	if o.Target != nil {
		toSerialize["target"] = o.Target
	}
	if o.Timeframe != nil {
		toSerialize["timeframe"] = o.Timeframe
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SLOOverallStatuses) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Error                   datadog.NullableString             `json:"error,omitempty"`
		ErrorBudgetRemaining    datadog.NullableFloat64            `json:"error_budget_remaining,omitempty"`
		IndexedAt               *int64                             `json:"indexed_at,omitempty"`
		RawErrorBudgetRemaining NullableSLORawErrorBudgetRemaining `json:"raw_error_budget_remaining,omitempty"`
		SpanPrecision           datadog.NullableInt64              `json:"span_precision,omitempty"`
		State                   *SLOState                          `json:"state,omitempty"`
		Status                  datadog.NullableFloat64            `json:"status,omitempty"`
		Target                  *float64                           `json:"target,omitempty"`
		Timeframe               *SLOTimeframe                      `json:"timeframe,omitempty"`
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
	if v := all.State; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	if v := all.Timeframe; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.Error = all.Error
	o.ErrorBudgetRemaining = all.ErrorBudgetRemaining
	o.IndexedAt = all.IndexedAt
	o.RawErrorBudgetRemaining = all.RawErrorBudgetRemaining
	o.SpanPrecision = all.SpanPrecision
	o.State = all.State
	o.Status = all.Status
	o.Target = all.Target
	o.Timeframe = all.Timeframe
	return nil
}
