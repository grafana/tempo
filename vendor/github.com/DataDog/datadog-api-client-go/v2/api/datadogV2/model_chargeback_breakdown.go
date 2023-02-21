// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// ChargebackBreakdown Charges breakdown.
type ChargebackBreakdown struct {
	// The type of charge for a particular product.
	ChargeType *string `json:"charge_type,omitempty"`
	// The cost for a particular product and charge type during a given month.
	Cost *float64 `json:"cost,omitempty"`
	// The product for which cost is being reported.
	ProductName *string `json:"product_name,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewChargebackBreakdown instantiates a new ChargebackBreakdown object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewChargebackBreakdown() *ChargebackBreakdown {
	this := ChargebackBreakdown{}
	return &this
}

// NewChargebackBreakdownWithDefaults instantiates a new ChargebackBreakdown object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewChargebackBreakdownWithDefaults() *ChargebackBreakdown {
	this := ChargebackBreakdown{}
	return &this
}

// GetChargeType returns the ChargeType field value if set, zero value otherwise.
func (o *ChargebackBreakdown) GetChargeType() string {
	if o == nil || o.ChargeType == nil {
		var ret string
		return ret
	}
	return *o.ChargeType
}

// GetChargeTypeOk returns a tuple with the ChargeType field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ChargebackBreakdown) GetChargeTypeOk() (*string, bool) {
	if o == nil || o.ChargeType == nil {
		return nil, false
	}
	return o.ChargeType, true
}

// HasChargeType returns a boolean if a field has been set.
func (o *ChargebackBreakdown) HasChargeType() bool {
	return o != nil && o.ChargeType != nil
}

// SetChargeType gets a reference to the given string and assigns it to the ChargeType field.
func (o *ChargebackBreakdown) SetChargeType(v string) {
	o.ChargeType = &v
}

// GetCost returns the Cost field value if set, zero value otherwise.
func (o *ChargebackBreakdown) GetCost() float64 {
	if o == nil || o.Cost == nil {
		var ret float64
		return ret
	}
	return *o.Cost
}

// GetCostOk returns a tuple with the Cost field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ChargebackBreakdown) GetCostOk() (*float64, bool) {
	if o == nil || o.Cost == nil {
		return nil, false
	}
	return o.Cost, true
}

// HasCost returns a boolean if a field has been set.
func (o *ChargebackBreakdown) HasCost() bool {
	return o != nil && o.Cost != nil
}

// SetCost gets a reference to the given float64 and assigns it to the Cost field.
func (o *ChargebackBreakdown) SetCost(v float64) {
	o.Cost = &v
}

// GetProductName returns the ProductName field value if set, zero value otherwise.
func (o *ChargebackBreakdown) GetProductName() string {
	if o == nil || o.ProductName == nil {
		var ret string
		return ret
	}
	return *o.ProductName
}

// GetProductNameOk returns a tuple with the ProductName field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ChargebackBreakdown) GetProductNameOk() (*string, bool) {
	if o == nil || o.ProductName == nil {
		return nil, false
	}
	return o.ProductName, true
}

// HasProductName returns a boolean if a field has been set.
func (o *ChargebackBreakdown) HasProductName() bool {
	return o != nil && o.ProductName != nil
}

// SetProductName gets a reference to the given string and assigns it to the ProductName field.
func (o *ChargebackBreakdown) SetProductName(v string) {
	o.ProductName = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o ChargebackBreakdown) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.ChargeType != nil {
		toSerialize["charge_type"] = o.ChargeType
	}
	if o.Cost != nil {
		toSerialize["cost"] = o.Cost
	}
	if o.ProductName != nil {
		toSerialize["product_name"] = o.ProductName
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *ChargebackBreakdown) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		ChargeType  *string  `json:"charge_type,omitempty"`
		Cost        *float64 `json:"cost,omitempty"`
		ProductName *string  `json:"product_name,omitempty"`
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
	o.ChargeType = all.ChargeType
	o.Cost = all.Cost
	o.ProductName = all.ProductName
	return nil
}
