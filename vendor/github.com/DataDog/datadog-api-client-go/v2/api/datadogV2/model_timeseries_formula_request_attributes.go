// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// TimeseriesFormulaRequestAttributes The object describing a timeseries formula request.
type TimeseriesFormulaRequestAttributes struct {
	// List of formulas to be calculated and returned as responses.
	Formulas []QueryFormula `json:"formulas,omitempty"`
	// Start date (inclusive) of the query in milliseconds since the Unix epoch.
	From int64 `json:"from"`
	// A time interval in milliseconds.
	// May be overridden by a larger interval if the query would result in
	// too many points for the specified timeframe.
	// Defaults to a reasonable interval for the given timeframe.
	Interval *int64 `json:"interval,omitempty"`
	// List of queries to be run and used as inputs to the formulas.
	Queries []TimeseriesQuery `json:"queries"`
	// End date (exclusive) of the query in milliseconds since the Unix epoch.
	To int64 `json:"to"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewTimeseriesFormulaRequestAttributes instantiates a new TimeseriesFormulaRequestAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewTimeseriesFormulaRequestAttributes(from int64, queries []TimeseriesQuery, to int64) *TimeseriesFormulaRequestAttributes {
	this := TimeseriesFormulaRequestAttributes{}
	this.From = from
	this.Queries = queries
	this.To = to
	return &this
}

// NewTimeseriesFormulaRequestAttributesWithDefaults instantiates a new TimeseriesFormulaRequestAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewTimeseriesFormulaRequestAttributesWithDefaults() *TimeseriesFormulaRequestAttributes {
	this := TimeseriesFormulaRequestAttributes{}
	return &this
}

// GetFormulas returns the Formulas field value if set, zero value otherwise.
func (o *TimeseriesFormulaRequestAttributes) GetFormulas() []QueryFormula {
	if o == nil || o.Formulas == nil {
		var ret []QueryFormula
		return ret
	}
	return o.Formulas
}

// GetFormulasOk returns a tuple with the Formulas field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *TimeseriesFormulaRequestAttributes) GetFormulasOk() (*[]QueryFormula, bool) {
	if o == nil || o.Formulas == nil {
		return nil, false
	}
	return &o.Formulas, true
}

// HasFormulas returns a boolean if a field has been set.
func (o *TimeseriesFormulaRequestAttributes) HasFormulas() bool {
	return o != nil && o.Formulas != nil
}

// SetFormulas gets a reference to the given []QueryFormula and assigns it to the Formulas field.
func (o *TimeseriesFormulaRequestAttributes) SetFormulas(v []QueryFormula) {
	o.Formulas = v
}

// GetFrom returns the From field value.
func (o *TimeseriesFormulaRequestAttributes) GetFrom() int64 {
	if o == nil {
		var ret int64
		return ret
	}
	return o.From
}

// GetFromOk returns a tuple with the From field value
// and a boolean to check if the value has been set.
func (o *TimeseriesFormulaRequestAttributes) GetFromOk() (*int64, bool) {
	if o == nil {
		return nil, false
	}
	return &o.From, true
}

// SetFrom sets field value.
func (o *TimeseriesFormulaRequestAttributes) SetFrom(v int64) {
	o.From = v
}

// GetInterval returns the Interval field value if set, zero value otherwise.
func (o *TimeseriesFormulaRequestAttributes) GetInterval() int64 {
	if o == nil || o.Interval == nil {
		var ret int64
		return ret
	}
	return *o.Interval
}

// GetIntervalOk returns a tuple with the Interval field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *TimeseriesFormulaRequestAttributes) GetIntervalOk() (*int64, bool) {
	if o == nil || o.Interval == nil {
		return nil, false
	}
	return o.Interval, true
}

// HasInterval returns a boolean if a field has been set.
func (o *TimeseriesFormulaRequestAttributes) HasInterval() bool {
	return o != nil && o.Interval != nil
}

// SetInterval gets a reference to the given int64 and assigns it to the Interval field.
func (o *TimeseriesFormulaRequestAttributes) SetInterval(v int64) {
	o.Interval = &v
}

// GetQueries returns the Queries field value.
func (o *TimeseriesFormulaRequestAttributes) GetQueries() []TimeseriesQuery {
	if o == nil {
		var ret []TimeseriesQuery
		return ret
	}
	return o.Queries
}

// GetQueriesOk returns a tuple with the Queries field value
// and a boolean to check if the value has been set.
func (o *TimeseriesFormulaRequestAttributes) GetQueriesOk() (*[]TimeseriesQuery, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Queries, true
}

// SetQueries sets field value.
func (o *TimeseriesFormulaRequestAttributes) SetQueries(v []TimeseriesQuery) {
	o.Queries = v
}

// GetTo returns the To field value.
func (o *TimeseriesFormulaRequestAttributes) GetTo() int64 {
	if o == nil {
		var ret int64
		return ret
	}
	return o.To
}

// GetToOk returns a tuple with the To field value
// and a boolean to check if the value has been set.
func (o *TimeseriesFormulaRequestAttributes) GetToOk() (*int64, bool) {
	if o == nil {
		return nil, false
	}
	return &o.To, true
}

// SetTo sets field value.
func (o *TimeseriesFormulaRequestAttributes) SetTo(v int64) {
	o.To = v
}

// MarshalJSON serializes the struct using spec logic.
func (o TimeseriesFormulaRequestAttributes) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Formulas != nil {
		toSerialize["formulas"] = o.Formulas
	}
	toSerialize["from"] = o.From
	if o.Interval != nil {
		toSerialize["interval"] = o.Interval
	}
	toSerialize["queries"] = o.Queries
	toSerialize["to"] = o.To

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *TimeseriesFormulaRequestAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		From    *int64             `json:"from"`
		Queries *[]TimeseriesQuery `json:"queries"`
		To      *int64             `json:"to"`
	}{}
	all := struct {
		Formulas []QueryFormula    `json:"formulas,omitempty"`
		From     int64             `json:"from"`
		Interval *int64            `json:"interval,omitempty"`
		Queries  []TimeseriesQuery `json:"queries"`
		To       int64             `json:"to"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.From == nil {
		return fmt.Errorf("required field from missing")
	}
	if required.Queries == nil {
		return fmt.Errorf("required field queries missing")
	}
	if required.To == nil {
		return fmt.Errorf("required field to missing")
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
	o.Formulas = all.Formulas
	o.From = all.From
	o.Interval = all.Interval
	o.Queries = all.Queries
	o.To = all.To
	return nil
}
