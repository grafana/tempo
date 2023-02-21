// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV1

import (
	"encoding/json"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
)

// SLOStatus Status of the SLO's primary timeframe.
type SLOStatus struct {
	// Error message if SLO status or error budget could not be calculated.
	CalculationError datadog.NullableString `json:"calculation_error,omitempty"`
	// Remaining error budget of the SLO in percentage.
	ErrorBudgetRemaining datadog.NullableFloat64 `json:"error_budget_remaining,omitempty"`
	// timestamp (UNIX time in seconds) of when the SLO status and error budget
	// were calculated.
	IndexedAt *int64 `json:"indexed_at,omitempty"`
	// Error budget remaining for an SLO.
	RawErrorBudgetRemaining NullableSLORawErrorBudgetRemaining `json:"raw_error_budget_remaining,omitempty"`
	// The current service level indicator (SLI) of the SLO, also known as 'status'. This is a percentage value from 0-100 (inclusive).
	Sli datadog.NullableFloat64 `json:"sli,omitempty"`
	// The number of decimal places the SLI value is accurate to.
	SpanPrecision datadog.NullableInt64 `json:"span_precision,omitempty"`
	// State of the SLO.
	State *SLOState `json:"state,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewSLOStatus instantiates a new SLOStatus object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSLOStatus() *SLOStatus {
	this := SLOStatus{}
	return &this
}

// NewSLOStatusWithDefaults instantiates a new SLOStatus object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSLOStatusWithDefaults() *SLOStatus {
	this := SLOStatus{}
	return &this
}

// GetCalculationError returns the CalculationError field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *SLOStatus) GetCalculationError() string {
	if o == nil || o.CalculationError.Get() == nil {
		var ret string
		return ret
	}
	return *o.CalculationError.Get()
}

// GetCalculationErrorOk returns a tuple with the CalculationError field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *SLOStatus) GetCalculationErrorOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return o.CalculationError.Get(), o.CalculationError.IsSet()
}

// HasCalculationError returns a boolean if a field has been set.
func (o *SLOStatus) HasCalculationError() bool {
	return o != nil && o.CalculationError.IsSet()
}

// SetCalculationError gets a reference to the given datadog.NullableString and assigns it to the CalculationError field.
func (o *SLOStatus) SetCalculationError(v string) {
	o.CalculationError.Set(&v)
}

// SetCalculationErrorNil sets the value for CalculationError to be an explicit nil.
func (o *SLOStatus) SetCalculationErrorNil() {
	o.CalculationError.Set(nil)
}

// UnsetCalculationError ensures that no value is present for CalculationError, not even an explicit nil.
func (o *SLOStatus) UnsetCalculationError() {
	o.CalculationError.Unset()
}

// GetErrorBudgetRemaining returns the ErrorBudgetRemaining field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *SLOStatus) GetErrorBudgetRemaining() float64 {
	if o == nil || o.ErrorBudgetRemaining.Get() == nil {
		var ret float64
		return ret
	}
	return *o.ErrorBudgetRemaining.Get()
}

// GetErrorBudgetRemainingOk returns a tuple with the ErrorBudgetRemaining field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *SLOStatus) GetErrorBudgetRemainingOk() (*float64, bool) {
	if o == nil {
		return nil, false
	}
	return o.ErrorBudgetRemaining.Get(), o.ErrorBudgetRemaining.IsSet()
}

// HasErrorBudgetRemaining returns a boolean if a field has been set.
func (o *SLOStatus) HasErrorBudgetRemaining() bool {
	return o != nil && o.ErrorBudgetRemaining.IsSet()
}

// SetErrorBudgetRemaining gets a reference to the given datadog.NullableFloat64 and assigns it to the ErrorBudgetRemaining field.
func (o *SLOStatus) SetErrorBudgetRemaining(v float64) {
	o.ErrorBudgetRemaining.Set(&v)
}

// SetErrorBudgetRemainingNil sets the value for ErrorBudgetRemaining to be an explicit nil.
func (o *SLOStatus) SetErrorBudgetRemainingNil() {
	o.ErrorBudgetRemaining.Set(nil)
}

// UnsetErrorBudgetRemaining ensures that no value is present for ErrorBudgetRemaining, not even an explicit nil.
func (o *SLOStatus) UnsetErrorBudgetRemaining() {
	o.ErrorBudgetRemaining.Unset()
}

// GetIndexedAt returns the IndexedAt field value if set, zero value otherwise.
func (o *SLOStatus) GetIndexedAt() int64 {
	if o == nil || o.IndexedAt == nil {
		var ret int64
		return ret
	}
	return *o.IndexedAt
}

// GetIndexedAtOk returns a tuple with the IndexedAt field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOStatus) GetIndexedAtOk() (*int64, bool) {
	if o == nil || o.IndexedAt == nil {
		return nil, false
	}
	return o.IndexedAt, true
}

// HasIndexedAt returns a boolean if a field has been set.
func (o *SLOStatus) HasIndexedAt() bool {
	return o != nil && o.IndexedAt != nil
}

// SetIndexedAt gets a reference to the given int64 and assigns it to the IndexedAt field.
func (o *SLOStatus) SetIndexedAt(v int64) {
	o.IndexedAt = &v
}

// GetRawErrorBudgetRemaining returns the RawErrorBudgetRemaining field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *SLOStatus) GetRawErrorBudgetRemaining() SLORawErrorBudgetRemaining {
	if o == nil || o.RawErrorBudgetRemaining.Get() == nil {
		var ret SLORawErrorBudgetRemaining
		return ret
	}
	return *o.RawErrorBudgetRemaining.Get()
}

// GetRawErrorBudgetRemainingOk returns a tuple with the RawErrorBudgetRemaining field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *SLOStatus) GetRawErrorBudgetRemainingOk() (*SLORawErrorBudgetRemaining, bool) {
	if o == nil {
		return nil, false
	}
	return o.RawErrorBudgetRemaining.Get(), o.RawErrorBudgetRemaining.IsSet()
}

// HasRawErrorBudgetRemaining returns a boolean if a field has been set.
func (o *SLOStatus) HasRawErrorBudgetRemaining() bool {
	return o != nil && o.RawErrorBudgetRemaining.IsSet()
}

// SetRawErrorBudgetRemaining gets a reference to the given NullableSLORawErrorBudgetRemaining and assigns it to the RawErrorBudgetRemaining field.
func (o *SLOStatus) SetRawErrorBudgetRemaining(v SLORawErrorBudgetRemaining) {
	o.RawErrorBudgetRemaining.Set(&v)
}

// SetRawErrorBudgetRemainingNil sets the value for RawErrorBudgetRemaining to be an explicit nil.
func (o *SLOStatus) SetRawErrorBudgetRemainingNil() {
	o.RawErrorBudgetRemaining.Set(nil)
}

// UnsetRawErrorBudgetRemaining ensures that no value is present for RawErrorBudgetRemaining, not even an explicit nil.
func (o *SLOStatus) UnsetRawErrorBudgetRemaining() {
	o.RawErrorBudgetRemaining.Unset()
}

// GetSli returns the Sli field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *SLOStatus) GetSli() float64 {
	if o == nil || o.Sli.Get() == nil {
		var ret float64
		return ret
	}
	return *o.Sli.Get()
}

// GetSliOk returns a tuple with the Sli field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *SLOStatus) GetSliOk() (*float64, bool) {
	if o == nil {
		return nil, false
	}
	return o.Sli.Get(), o.Sli.IsSet()
}

// HasSli returns a boolean if a field has been set.
func (o *SLOStatus) HasSli() bool {
	return o != nil && o.Sli.IsSet()
}

// SetSli gets a reference to the given datadog.NullableFloat64 and assigns it to the Sli field.
func (o *SLOStatus) SetSli(v float64) {
	o.Sli.Set(&v)
}

// SetSliNil sets the value for Sli to be an explicit nil.
func (o *SLOStatus) SetSliNil() {
	o.Sli.Set(nil)
}

// UnsetSli ensures that no value is present for Sli, not even an explicit nil.
func (o *SLOStatus) UnsetSli() {
	o.Sli.Unset()
}

// GetSpanPrecision returns the SpanPrecision field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *SLOStatus) GetSpanPrecision() int64 {
	if o == nil || o.SpanPrecision.Get() == nil {
		var ret int64
		return ret
	}
	return *o.SpanPrecision.Get()
}

// GetSpanPrecisionOk returns a tuple with the SpanPrecision field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *SLOStatus) GetSpanPrecisionOk() (*int64, bool) {
	if o == nil {
		return nil, false
	}
	return o.SpanPrecision.Get(), o.SpanPrecision.IsSet()
}

// HasSpanPrecision returns a boolean if a field has been set.
func (o *SLOStatus) HasSpanPrecision() bool {
	return o != nil && o.SpanPrecision.IsSet()
}

// SetSpanPrecision gets a reference to the given datadog.NullableInt64 and assigns it to the SpanPrecision field.
func (o *SLOStatus) SetSpanPrecision(v int64) {
	o.SpanPrecision.Set(&v)
}

// SetSpanPrecisionNil sets the value for SpanPrecision to be an explicit nil.
func (o *SLOStatus) SetSpanPrecisionNil() {
	o.SpanPrecision.Set(nil)
}

// UnsetSpanPrecision ensures that no value is present for SpanPrecision, not even an explicit nil.
func (o *SLOStatus) UnsetSpanPrecision() {
	o.SpanPrecision.Unset()
}

// GetState returns the State field value if set, zero value otherwise.
func (o *SLOStatus) GetState() SLOState {
	if o == nil || o.State == nil {
		var ret SLOState
		return ret
	}
	return *o.State
}

// GetStateOk returns a tuple with the State field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOStatus) GetStateOk() (*SLOState, bool) {
	if o == nil || o.State == nil {
		return nil, false
	}
	return o.State, true
}

// HasState returns a boolean if a field has been set.
func (o *SLOStatus) HasState() bool {
	return o != nil && o.State != nil
}

// SetState gets a reference to the given SLOState and assigns it to the State field.
func (o *SLOStatus) SetState(v SLOState) {
	o.State = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SLOStatus) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.CalculationError.IsSet() {
		toSerialize["calculation_error"] = o.CalculationError.Get()
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
	if o.Sli.IsSet() {
		toSerialize["sli"] = o.Sli.Get()
	}
	if o.SpanPrecision.IsSet() {
		toSerialize["span_precision"] = o.SpanPrecision.Get()
	}
	if o.State != nil {
		toSerialize["state"] = o.State
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SLOStatus) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		CalculationError        datadog.NullableString             `json:"calculation_error,omitempty"`
		ErrorBudgetRemaining    datadog.NullableFloat64            `json:"error_budget_remaining,omitempty"`
		IndexedAt               *int64                             `json:"indexed_at,omitempty"`
		RawErrorBudgetRemaining NullableSLORawErrorBudgetRemaining `json:"raw_error_budget_remaining,omitempty"`
		Sli                     datadog.NullableFloat64            `json:"sli,omitempty"`
		SpanPrecision           datadog.NullableInt64              `json:"span_precision,omitempty"`
		State                   *SLOState                          `json:"state,omitempty"`
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
	o.CalculationError = all.CalculationError
	o.ErrorBudgetRemaining = all.ErrorBudgetRemaining
	o.IndexedAt = all.IndexedAt
	o.RawErrorBudgetRemaining = all.RawErrorBudgetRemaining
	o.Sli = all.Sli
	o.SpanPrecision = all.SpanPrecision
	o.State = all.State
	return nil
}
