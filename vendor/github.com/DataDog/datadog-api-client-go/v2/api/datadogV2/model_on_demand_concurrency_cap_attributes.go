// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// OnDemandConcurrencyCapAttributes On-demand concurrency cap attributes.
type OnDemandConcurrencyCapAttributes struct {
	// Value of the on-demand concurrency cap.
	OnDemandConcurrencyCap *float64 `json:"on_demand_concurrency_cap,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewOnDemandConcurrencyCapAttributes instantiates a new OnDemandConcurrencyCapAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewOnDemandConcurrencyCapAttributes() *OnDemandConcurrencyCapAttributes {
	this := OnDemandConcurrencyCapAttributes{}
	return &this
}

// NewOnDemandConcurrencyCapAttributesWithDefaults instantiates a new OnDemandConcurrencyCapAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewOnDemandConcurrencyCapAttributesWithDefaults() *OnDemandConcurrencyCapAttributes {
	this := OnDemandConcurrencyCapAttributes{}
	return &this
}

// GetOnDemandConcurrencyCap returns the OnDemandConcurrencyCap field value if set, zero value otherwise.
func (o *OnDemandConcurrencyCapAttributes) GetOnDemandConcurrencyCap() float64 {
	if o == nil || o.OnDemandConcurrencyCap == nil {
		var ret float64
		return ret
	}
	return *o.OnDemandConcurrencyCap
}

// GetOnDemandConcurrencyCapOk returns a tuple with the OnDemandConcurrencyCap field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *OnDemandConcurrencyCapAttributes) GetOnDemandConcurrencyCapOk() (*float64, bool) {
	if o == nil || o.OnDemandConcurrencyCap == nil {
		return nil, false
	}
	return o.OnDemandConcurrencyCap, true
}

// HasOnDemandConcurrencyCap returns a boolean if a field has been set.
func (o *OnDemandConcurrencyCapAttributes) HasOnDemandConcurrencyCap() bool {
	return o != nil && o.OnDemandConcurrencyCap != nil
}

// SetOnDemandConcurrencyCap gets a reference to the given float64 and assigns it to the OnDemandConcurrencyCap field.
func (o *OnDemandConcurrencyCapAttributes) SetOnDemandConcurrencyCap(v float64) {
	o.OnDemandConcurrencyCap = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o OnDemandConcurrencyCapAttributes) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.OnDemandConcurrencyCap != nil {
		toSerialize["on_demand_concurrency_cap"] = o.OnDemandConcurrencyCap
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *OnDemandConcurrencyCapAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		OnDemandConcurrencyCap *float64 `json:"on_demand_concurrency_cap,omitempty"`
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
	o.OnDemandConcurrencyCap = all.OnDemandConcurrencyCap
	return nil
}
