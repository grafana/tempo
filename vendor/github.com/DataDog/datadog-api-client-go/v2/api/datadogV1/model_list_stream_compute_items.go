// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV1

import (
	"encoding/json"
	"fmt"
)

// ListStreamComputeItems List of facets and aggregations which to compute.
type ListStreamComputeItems struct {
	// Aggregation value.
	Aggregation ListStreamComputeAggregation `json:"aggregation"`
	// Facet name.
	Facet *string `json:"facet,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewListStreamComputeItems instantiates a new ListStreamComputeItems object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewListStreamComputeItems(aggregation ListStreamComputeAggregation) *ListStreamComputeItems {
	this := ListStreamComputeItems{}
	this.Aggregation = aggregation
	return &this
}

// NewListStreamComputeItemsWithDefaults instantiates a new ListStreamComputeItems object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewListStreamComputeItemsWithDefaults() *ListStreamComputeItems {
	this := ListStreamComputeItems{}
	return &this
}

// GetAggregation returns the Aggregation field value.
func (o *ListStreamComputeItems) GetAggregation() ListStreamComputeAggregation {
	if o == nil {
		var ret ListStreamComputeAggregation
		return ret
	}
	return o.Aggregation
}

// GetAggregationOk returns a tuple with the Aggregation field value
// and a boolean to check if the value has been set.
func (o *ListStreamComputeItems) GetAggregationOk() (*ListStreamComputeAggregation, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Aggregation, true
}

// SetAggregation sets field value.
func (o *ListStreamComputeItems) SetAggregation(v ListStreamComputeAggregation) {
	o.Aggregation = v
}

// GetFacet returns the Facet field value if set, zero value otherwise.
func (o *ListStreamComputeItems) GetFacet() string {
	if o == nil || o.Facet == nil {
		var ret string
		return ret
	}
	return *o.Facet
}

// GetFacetOk returns a tuple with the Facet field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ListStreamComputeItems) GetFacetOk() (*string, bool) {
	if o == nil || o.Facet == nil {
		return nil, false
	}
	return o.Facet, true
}

// HasFacet returns a boolean if a field has been set.
func (o *ListStreamComputeItems) HasFacet() bool {
	return o != nil && o.Facet != nil
}

// SetFacet gets a reference to the given string and assigns it to the Facet field.
func (o *ListStreamComputeItems) SetFacet(v string) {
	o.Facet = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o ListStreamComputeItems) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["aggregation"] = o.Aggregation
	if o.Facet != nil {
		toSerialize["facet"] = o.Facet
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *ListStreamComputeItems) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Aggregation *ListStreamComputeAggregation `json:"aggregation"`
	}{}
	all := struct {
		Aggregation ListStreamComputeAggregation `json:"aggregation"`
		Facet       *string                      `json:"facet,omitempty"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Aggregation == nil {
		return fmt.Errorf("required field aggregation missing")
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
	if v := all.Aggregation; !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.Aggregation = all.Aggregation
	o.Facet = all.Facet
	return nil
}
