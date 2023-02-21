// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// ScalarFormulaResponseAtrributes The object describing a scalar response.
type ScalarFormulaResponseAtrributes struct {
	// List of response columns, each corresponding to an individual formula or query in the request and with values in parallel arrays matching the series list.
	Columns []ScalarColumn `json:"columns,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewScalarFormulaResponseAtrributes instantiates a new ScalarFormulaResponseAtrributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewScalarFormulaResponseAtrributes() *ScalarFormulaResponseAtrributes {
	this := ScalarFormulaResponseAtrributes{}
	return &this
}

// NewScalarFormulaResponseAtrributesWithDefaults instantiates a new ScalarFormulaResponseAtrributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewScalarFormulaResponseAtrributesWithDefaults() *ScalarFormulaResponseAtrributes {
	this := ScalarFormulaResponseAtrributes{}
	return &this
}

// GetColumns returns the Columns field value if set, zero value otherwise.
func (o *ScalarFormulaResponseAtrributes) GetColumns() []ScalarColumn {
	if o == nil || o.Columns == nil {
		var ret []ScalarColumn
		return ret
	}
	return o.Columns
}

// GetColumnsOk returns a tuple with the Columns field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ScalarFormulaResponseAtrributes) GetColumnsOk() (*[]ScalarColumn, bool) {
	if o == nil || o.Columns == nil {
		return nil, false
	}
	return &o.Columns, true
}

// HasColumns returns a boolean if a field has been set.
func (o *ScalarFormulaResponseAtrributes) HasColumns() bool {
	return o != nil && o.Columns != nil
}

// SetColumns gets a reference to the given []ScalarColumn and assigns it to the Columns field.
func (o *ScalarFormulaResponseAtrributes) SetColumns(v []ScalarColumn) {
	o.Columns = v
}

// MarshalJSON serializes the struct using spec logic.
func (o ScalarFormulaResponseAtrributes) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Columns != nil {
		toSerialize["columns"] = o.Columns
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *ScalarFormulaResponseAtrributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Columns []ScalarColumn `json:"columns,omitempty"`
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
	o.Columns = all.Columns
	return nil
}
