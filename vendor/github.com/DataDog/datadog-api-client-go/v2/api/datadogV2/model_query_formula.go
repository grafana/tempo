// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// QueryFormula A formula for calculation based on one or more queries.
type QueryFormula struct {
	// Formula string, referencing one or more queries with their name property.
	Formula string `json:"formula"`
	// Message for specifying limits to the number of values returned by a query.
	Limit *FormulaLimit `json:"limit,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewQueryFormula instantiates a new QueryFormula object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewQueryFormula(formula string) *QueryFormula {
	this := QueryFormula{}
	this.Formula = formula
	return &this
}

// NewQueryFormulaWithDefaults instantiates a new QueryFormula object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewQueryFormulaWithDefaults() *QueryFormula {
	this := QueryFormula{}
	return &this
}

// GetFormula returns the Formula field value.
func (o *QueryFormula) GetFormula() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Formula
}

// GetFormulaOk returns a tuple with the Formula field value
// and a boolean to check if the value has been set.
func (o *QueryFormula) GetFormulaOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Formula, true
}

// SetFormula sets field value.
func (o *QueryFormula) SetFormula(v string) {
	o.Formula = v
}

// GetLimit returns the Limit field value if set, zero value otherwise.
func (o *QueryFormula) GetLimit() FormulaLimit {
	if o == nil || o.Limit == nil {
		var ret FormulaLimit
		return ret
	}
	return *o.Limit
}

// GetLimitOk returns a tuple with the Limit field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *QueryFormula) GetLimitOk() (*FormulaLimit, bool) {
	if o == nil || o.Limit == nil {
		return nil, false
	}
	return o.Limit, true
}

// HasLimit returns a boolean if a field has been set.
func (o *QueryFormula) HasLimit() bool {
	return o != nil && o.Limit != nil
}

// SetLimit gets a reference to the given FormulaLimit and assigns it to the Limit field.
func (o *QueryFormula) SetLimit(v FormulaLimit) {
	o.Limit = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o QueryFormula) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["formula"] = o.Formula
	if o.Limit != nil {
		toSerialize["limit"] = o.Limit
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *QueryFormula) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Formula *string `json:"formula"`
	}{}
	all := struct {
		Formula string        `json:"formula"`
		Limit   *FormulaLimit `json:"limit,omitempty"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Formula == nil {
		return fmt.Errorf("required field formula missing")
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
	o.Formula = all.Formula
	if all.Limit != nil && all.Limit.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Limit = all.Limit
	return nil
}
