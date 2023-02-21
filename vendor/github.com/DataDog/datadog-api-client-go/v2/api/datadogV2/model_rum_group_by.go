// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// RUMGroupBy A group-by rule.
type RUMGroupBy struct {
	// The name of the facet to use (required).
	Facet string `json:"facet"`
	// Used to perform a histogram computation (only for measure facets).
	// Note: At most 100 buckets are allowed, the number of buckets is (max - min)/interval.
	Histogram *RUMGroupByHistogram `json:"histogram,omitempty"`
	// The maximum buckets to return for this group-by.
	Limit *int64 `json:"limit,omitempty"`
	// The value to use for logs that don't have the facet used to group by.
	Missing *RUMGroupByMissing `json:"missing,omitempty"`
	// A sort rule.
	Sort *RUMAggregateSort `json:"sort,omitempty"`
	// A resulting object to put the given computes in over all the matching records.
	Total *RUMGroupByTotal `json:"total,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewRUMGroupBy instantiates a new RUMGroupBy object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewRUMGroupBy(facet string) *RUMGroupBy {
	this := RUMGroupBy{}
	this.Facet = facet
	var limit int64 = 10
	this.Limit = &limit
	return &this
}

// NewRUMGroupByWithDefaults instantiates a new RUMGroupBy object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewRUMGroupByWithDefaults() *RUMGroupBy {
	this := RUMGroupBy{}
	var limit int64 = 10
	this.Limit = &limit
	return &this
}

// GetFacet returns the Facet field value.
func (o *RUMGroupBy) GetFacet() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Facet
}

// GetFacetOk returns a tuple with the Facet field value
// and a boolean to check if the value has been set.
func (o *RUMGroupBy) GetFacetOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Facet, true
}

// SetFacet sets field value.
func (o *RUMGroupBy) SetFacet(v string) {
	o.Facet = v
}

// GetHistogram returns the Histogram field value if set, zero value otherwise.
func (o *RUMGroupBy) GetHistogram() RUMGroupByHistogram {
	if o == nil || o.Histogram == nil {
		var ret RUMGroupByHistogram
		return ret
	}
	return *o.Histogram
}

// GetHistogramOk returns a tuple with the Histogram field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *RUMGroupBy) GetHistogramOk() (*RUMGroupByHistogram, bool) {
	if o == nil || o.Histogram == nil {
		return nil, false
	}
	return o.Histogram, true
}

// HasHistogram returns a boolean if a field has been set.
func (o *RUMGroupBy) HasHistogram() bool {
	return o != nil && o.Histogram != nil
}

// SetHistogram gets a reference to the given RUMGroupByHistogram and assigns it to the Histogram field.
func (o *RUMGroupBy) SetHistogram(v RUMGroupByHistogram) {
	o.Histogram = &v
}

// GetLimit returns the Limit field value if set, zero value otherwise.
func (o *RUMGroupBy) GetLimit() int64 {
	if o == nil || o.Limit == nil {
		var ret int64
		return ret
	}
	return *o.Limit
}

// GetLimitOk returns a tuple with the Limit field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *RUMGroupBy) GetLimitOk() (*int64, bool) {
	if o == nil || o.Limit == nil {
		return nil, false
	}
	return o.Limit, true
}

// HasLimit returns a boolean if a field has been set.
func (o *RUMGroupBy) HasLimit() bool {
	return o != nil && o.Limit != nil
}

// SetLimit gets a reference to the given int64 and assigns it to the Limit field.
func (o *RUMGroupBy) SetLimit(v int64) {
	o.Limit = &v
}

// GetMissing returns the Missing field value if set, zero value otherwise.
func (o *RUMGroupBy) GetMissing() RUMGroupByMissing {
	if o == nil || o.Missing == nil {
		var ret RUMGroupByMissing
		return ret
	}
	return *o.Missing
}

// GetMissingOk returns a tuple with the Missing field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *RUMGroupBy) GetMissingOk() (*RUMGroupByMissing, bool) {
	if o == nil || o.Missing == nil {
		return nil, false
	}
	return o.Missing, true
}

// HasMissing returns a boolean if a field has been set.
func (o *RUMGroupBy) HasMissing() bool {
	return o != nil && o.Missing != nil
}

// SetMissing gets a reference to the given RUMGroupByMissing and assigns it to the Missing field.
func (o *RUMGroupBy) SetMissing(v RUMGroupByMissing) {
	o.Missing = &v
}

// GetSort returns the Sort field value if set, zero value otherwise.
func (o *RUMGroupBy) GetSort() RUMAggregateSort {
	if o == nil || o.Sort == nil {
		var ret RUMAggregateSort
		return ret
	}
	return *o.Sort
}

// GetSortOk returns a tuple with the Sort field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *RUMGroupBy) GetSortOk() (*RUMAggregateSort, bool) {
	if o == nil || o.Sort == nil {
		return nil, false
	}
	return o.Sort, true
}

// HasSort returns a boolean if a field has been set.
func (o *RUMGroupBy) HasSort() bool {
	return o != nil && o.Sort != nil
}

// SetSort gets a reference to the given RUMAggregateSort and assigns it to the Sort field.
func (o *RUMGroupBy) SetSort(v RUMAggregateSort) {
	o.Sort = &v
}

// GetTotal returns the Total field value if set, zero value otherwise.
func (o *RUMGroupBy) GetTotal() RUMGroupByTotal {
	if o == nil || o.Total == nil {
		var ret RUMGroupByTotal
		return ret
	}
	return *o.Total
}

// GetTotalOk returns a tuple with the Total field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *RUMGroupBy) GetTotalOk() (*RUMGroupByTotal, bool) {
	if o == nil || o.Total == nil {
		return nil, false
	}
	return o.Total, true
}

// HasTotal returns a boolean if a field has been set.
func (o *RUMGroupBy) HasTotal() bool {
	return o != nil && o.Total != nil
}

// SetTotal gets a reference to the given RUMGroupByTotal and assigns it to the Total field.
func (o *RUMGroupBy) SetTotal(v RUMGroupByTotal) {
	o.Total = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o RUMGroupBy) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["facet"] = o.Facet
	if o.Histogram != nil {
		toSerialize["histogram"] = o.Histogram
	}
	if o.Limit != nil {
		toSerialize["limit"] = o.Limit
	}
	if o.Missing != nil {
		toSerialize["missing"] = o.Missing
	}
	if o.Sort != nil {
		toSerialize["sort"] = o.Sort
	}
	if o.Total != nil {
		toSerialize["total"] = o.Total
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *RUMGroupBy) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Facet *string `json:"facet"`
	}{}
	all := struct {
		Facet     string               `json:"facet"`
		Histogram *RUMGroupByHistogram `json:"histogram,omitempty"`
		Limit     *int64               `json:"limit,omitempty"`
		Missing   *RUMGroupByMissing   `json:"missing,omitempty"`
		Sort      *RUMAggregateSort    `json:"sort,omitempty"`
		Total     *RUMGroupByTotal     `json:"total,omitempty"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Facet == nil {
		return fmt.Errorf("required field facet missing")
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
	o.Facet = all.Facet
	if all.Histogram != nil && all.Histogram.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Histogram = all.Histogram
	o.Limit = all.Limit
	o.Missing = all.Missing
	if all.Sort != nil && all.Sort.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Sort = all.Sort
	o.Total = all.Total
	return nil
}
