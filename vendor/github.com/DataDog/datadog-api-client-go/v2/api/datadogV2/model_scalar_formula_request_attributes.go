// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// ScalarFormulaRequestAttributes The object describing a scalar formula request.
type ScalarFormulaRequestAttributes struct {
	// List of formulas to be calculated and returned as responses.
	Formulas []QueryFormula `json:"formulas,omitempty"`
	// Start date (inclusive) of the query in milliseconds since the Unix epoch.
	From int64 `json:"from"`
	// List of queries to be run and used as inputs to the formulas.
	Queries []ScalarQuery `json:"queries"`
	// End date (exclusive) of the query in milliseconds since the Unix epoch.
	To int64 `json:"to"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewScalarFormulaRequestAttributes instantiates a new ScalarFormulaRequestAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewScalarFormulaRequestAttributes(from int64, queries []ScalarQuery, to int64) *ScalarFormulaRequestAttributes {
	this := ScalarFormulaRequestAttributes{}
	this.From = from
	this.Queries = queries
	this.To = to
	return &this
}

// NewScalarFormulaRequestAttributesWithDefaults instantiates a new ScalarFormulaRequestAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewScalarFormulaRequestAttributesWithDefaults() *ScalarFormulaRequestAttributes {
	this := ScalarFormulaRequestAttributes{}
	return &this
}

// GetFormulas returns the Formulas field value if set, zero value otherwise.
func (o *ScalarFormulaRequestAttributes) GetFormulas() []QueryFormula {
	if o == nil || o.Formulas == nil {
		var ret []QueryFormula
		return ret
	}
	return o.Formulas
}

// GetFormulasOk returns a tuple with the Formulas field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ScalarFormulaRequestAttributes) GetFormulasOk() (*[]QueryFormula, bool) {
	if o == nil || o.Formulas == nil {
		return nil, false
	}
	return &o.Formulas, true
}

// HasFormulas returns a boolean if a field has been set.
func (o *ScalarFormulaRequestAttributes) HasFormulas() bool {
	return o != nil && o.Formulas != nil
}

// SetFormulas gets a reference to the given []QueryFormula and assigns it to the Formulas field.
func (o *ScalarFormulaRequestAttributes) SetFormulas(v []QueryFormula) {
	o.Formulas = v
}

// GetFrom returns the From field value.
func (o *ScalarFormulaRequestAttributes) GetFrom() int64 {
	if o == nil {
		var ret int64
		return ret
	}
	return o.From
}

// GetFromOk returns a tuple with the From field value
// and a boolean to check if the value has been set.
func (o *ScalarFormulaRequestAttributes) GetFromOk() (*int64, bool) {
	if o == nil {
		return nil, false
	}
	return &o.From, true
}

// SetFrom sets field value.
func (o *ScalarFormulaRequestAttributes) SetFrom(v int64) {
	o.From = v
}

// GetQueries returns the Queries field value.
func (o *ScalarFormulaRequestAttributes) GetQueries() []ScalarQuery {
	if o == nil {
		var ret []ScalarQuery
		return ret
	}
	return o.Queries
}

// GetQueriesOk returns a tuple with the Queries field value
// and a boolean to check if the value has been set.
func (o *ScalarFormulaRequestAttributes) GetQueriesOk() (*[]ScalarQuery, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Queries, true
}

// SetQueries sets field value.
func (o *ScalarFormulaRequestAttributes) SetQueries(v []ScalarQuery) {
	o.Queries = v
}

// GetTo returns the To field value.
func (o *ScalarFormulaRequestAttributes) GetTo() int64 {
	if o == nil {
		var ret int64
		return ret
	}
	return o.To
}

// GetToOk returns a tuple with the To field value
// and a boolean to check if the value has been set.
func (o *ScalarFormulaRequestAttributes) GetToOk() (*int64, bool) {
	if o == nil {
		return nil, false
	}
	return &o.To, true
}

// SetTo sets field value.
func (o *ScalarFormulaRequestAttributes) SetTo(v int64) {
	o.To = v
}

// MarshalJSON serializes the struct using spec logic.
func (o ScalarFormulaRequestAttributes) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Formulas != nil {
		toSerialize["formulas"] = o.Formulas
	}
	toSerialize["from"] = o.From
	toSerialize["queries"] = o.Queries
	toSerialize["to"] = o.To

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *ScalarFormulaRequestAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		From    *int64         `json:"from"`
		Queries *[]ScalarQuery `json:"queries"`
		To      *int64         `json:"to"`
	}{}
	all := struct {
		Formulas []QueryFormula `json:"formulas,omitempty"`
		From     int64          `json:"from"`
		Queries  []ScalarQuery  `json:"queries"`
		To       int64          `json:"to"`
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
	o.Queries = all.Queries
	o.To = all.To
	return nil
}
