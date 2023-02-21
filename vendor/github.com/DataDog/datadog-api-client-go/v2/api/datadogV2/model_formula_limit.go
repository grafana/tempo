// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// FormulaLimit Message for specifying limits to the number of values returned by a query.
type FormulaLimit struct {
	// The number of results to which to limit.
	Count *int32 `json:"count,omitempty"`
	// Direction of sort.
	Order *QuerySortOrder `json:"order,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewFormulaLimit instantiates a new FormulaLimit object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewFormulaLimit() *FormulaLimit {
	this := FormulaLimit{}
	var order QuerySortOrder = QUERYSORTORDER_DESC
	this.Order = &order
	return &this
}

// NewFormulaLimitWithDefaults instantiates a new FormulaLimit object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewFormulaLimitWithDefaults() *FormulaLimit {
	this := FormulaLimit{}
	var order QuerySortOrder = QUERYSORTORDER_DESC
	this.Order = &order
	return &this
}

// GetCount returns the Count field value if set, zero value otherwise.
func (o *FormulaLimit) GetCount() int32 {
	if o == nil || o.Count == nil {
		var ret int32
		return ret
	}
	return *o.Count
}

// GetCountOk returns a tuple with the Count field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *FormulaLimit) GetCountOk() (*int32, bool) {
	if o == nil || o.Count == nil {
		return nil, false
	}
	return o.Count, true
}

// HasCount returns a boolean if a field has been set.
func (o *FormulaLimit) HasCount() bool {
	return o != nil && o.Count != nil
}

// SetCount gets a reference to the given int32 and assigns it to the Count field.
func (o *FormulaLimit) SetCount(v int32) {
	o.Count = &v
}

// GetOrder returns the Order field value if set, zero value otherwise.
func (o *FormulaLimit) GetOrder() QuerySortOrder {
	if o == nil || o.Order == nil {
		var ret QuerySortOrder
		return ret
	}
	return *o.Order
}

// GetOrderOk returns a tuple with the Order field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *FormulaLimit) GetOrderOk() (*QuerySortOrder, bool) {
	if o == nil || o.Order == nil {
		return nil, false
	}
	return o.Order, true
}

// HasOrder returns a boolean if a field has been set.
func (o *FormulaLimit) HasOrder() bool {
	return o != nil && o.Order != nil
}

// SetOrder gets a reference to the given QuerySortOrder and assigns it to the Order field.
func (o *FormulaLimit) SetOrder(v QuerySortOrder) {
	o.Order = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o FormulaLimit) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Count != nil {
		toSerialize["count"] = o.Count
	}
	if o.Order != nil {
		toSerialize["order"] = o.Order
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *FormulaLimit) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Count *int32          `json:"count,omitempty"`
		Order *QuerySortOrder `json:"order,omitempty"`
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
	if v := all.Order; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.Count = all.Count
	o.Order = all.Order
	return nil
}
