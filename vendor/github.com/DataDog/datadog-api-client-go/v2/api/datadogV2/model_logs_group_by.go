// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// LogsGroupBy A group by rule
type LogsGroupBy struct {
	// The name of the facet to use (required)
	Facet string `json:"facet"`
	// Used to perform a histogram computation (only for measure facets).
	// Note: At most 100 buckets are allowed, the number of buckets is (max - min)/interval.
	Histogram *LogsGroupByHistogram `json:"histogram,omitempty"`
	// The maximum buckets to return for this group by
	Limit *int64 `json:"limit,omitempty"`
	// The value to use for logs that don't have the facet used to group by
	Missing *LogsGroupByMissing `json:"missing,omitempty"`
	// A sort rule
	Sort *LogsAggregateSort `json:"sort,omitempty"`
	// A resulting object to put the given computes in over all the matching records.
	Total *LogsGroupByTotal `json:"total,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewLogsGroupBy instantiates a new LogsGroupBy object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewLogsGroupBy(facet string) *LogsGroupBy {
	this := LogsGroupBy{}
	this.Facet = facet
	var limit int64 = 10
	this.Limit = &limit
	return &this
}

// NewLogsGroupByWithDefaults instantiates a new LogsGroupBy object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewLogsGroupByWithDefaults() *LogsGroupBy {
	this := LogsGroupBy{}
	var limit int64 = 10
	this.Limit = &limit
	return &this
}

// GetFacet returns the Facet field value.
func (o *LogsGroupBy) GetFacet() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Facet
}

// GetFacetOk returns a tuple with the Facet field value
// and a boolean to check if the value has been set.
func (o *LogsGroupBy) GetFacetOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Facet, true
}

// SetFacet sets field value.
func (o *LogsGroupBy) SetFacet(v string) {
	o.Facet = v
}

// GetHistogram returns the Histogram field value if set, zero value otherwise.
func (o *LogsGroupBy) GetHistogram() LogsGroupByHistogram {
	if o == nil || o.Histogram == nil {
		var ret LogsGroupByHistogram
		return ret
	}
	return *o.Histogram
}

// GetHistogramOk returns a tuple with the Histogram field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsGroupBy) GetHistogramOk() (*LogsGroupByHistogram, bool) {
	if o == nil || o.Histogram == nil {
		return nil, false
	}
	return o.Histogram, true
}

// HasHistogram returns a boolean if a field has been set.
func (o *LogsGroupBy) HasHistogram() bool {
	return o != nil && o.Histogram != nil
}

// SetHistogram gets a reference to the given LogsGroupByHistogram and assigns it to the Histogram field.
func (o *LogsGroupBy) SetHistogram(v LogsGroupByHistogram) {
	o.Histogram = &v
}

// GetLimit returns the Limit field value if set, zero value otherwise.
func (o *LogsGroupBy) GetLimit() int64 {
	if o == nil || o.Limit == nil {
		var ret int64
		return ret
	}
	return *o.Limit
}

// GetLimitOk returns a tuple with the Limit field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsGroupBy) GetLimitOk() (*int64, bool) {
	if o == nil || o.Limit == nil {
		return nil, false
	}
	return o.Limit, true
}

// HasLimit returns a boolean if a field has been set.
func (o *LogsGroupBy) HasLimit() bool {
	return o != nil && o.Limit != nil
}

// SetLimit gets a reference to the given int64 and assigns it to the Limit field.
func (o *LogsGroupBy) SetLimit(v int64) {
	o.Limit = &v
}

// GetMissing returns the Missing field value if set, zero value otherwise.
func (o *LogsGroupBy) GetMissing() LogsGroupByMissing {
	if o == nil || o.Missing == nil {
		var ret LogsGroupByMissing
		return ret
	}
	return *o.Missing
}

// GetMissingOk returns a tuple with the Missing field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsGroupBy) GetMissingOk() (*LogsGroupByMissing, bool) {
	if o == nil || o.Missing == nil {
		return nil, false
	}
	return o.Missing, true
}

// HasMissing returns a boolean if a field has been set.
func (o *LogsGroupBy) HasMissing() bool {
	return o != nil && o.Missing != nil
}

// SetMissing gets a reference to the given LogsGroupByMissing and assigns it to the Missing field.
func (o *LogsGroupBy) SetMissing(v LogsGroupByMissing) {
	o.Missing = &v
}

// GetSort returns the Sort field value if set, zero value otherwise.
func (o *LogsGroupBy) GetSort() LogsAggregateSort {
	if o == nil || o.Sort == nil {
		var ret LogsAggregateSort
		return ret
	}
	return *o.Sort
}

// GetSortOk returns a tuple with the Sort field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsGroupBy) GetSortOk() (*LogsAggregateSort, bool) {
	if o == nil || o.Sort == nil {
		return nil, false
	}
	return o.Sort, true
}

// HasSort returns a boolean if a field has been set.
func (o *LogsGroupBy) HasSort() bool {
	return o != nil && o.Sort != nil
}

// SetSort gets a reference to the given LogsAggregateSort and assigns it to the Sort field.
func (o *LogsGroupBy) SetSort(v LogsAggregateSort) {
	o.Sort = &v
}

// GetTotal returns the Total field value if set, zero value otherwise.
func (o *LogsGroupBy) GetTotal() LogsGroupByTotal {
	if o == nil || o.Total == nil {
		var ret LogsGroupByTotal
		return ret
	}
	return *o.Total
}

// GetTotalOk returns a tuple with the Total field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsGroupBy) GetTotalOk() (*LogsGroupByTotal, bool) {
	if o == nil || o.Total == nil {
		return nil, false
	}
	return o.Total, true
}

// HasTotal returns a boolean if a field has been set.
func (o *LogsGroupBy) HasTotal() bool {
	return o != nil && o.Total != nil
}

// SetTotal gets a reference to the given LogsGroupByTotal and assigns it to the Total field.
func (o *LogsGroupBy) SetTotal(v LogsGroupByTotal) {
	o.Total = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o LogsGroupBy) MarshalJSON() ([]byte, error) {
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
func (o *LogsGroupBy) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Facet *string `json:"facet"`
	}{}
	all := struct {
		Facet     string                `json:"facet"`
		Histogram *LogsGroupByHistogram `json:"histogram,omitempty"`
		Limit     *int64                `json:"limit,omitempty"`
		Missing   *LogsGroupByMissing   `json:"missing,omitempty"`
		Sort      *LogsAggregateSort    `json:"sort,omitempty"`
		Total     *LogsGroupByTotal     `json:"total,omitempty"`
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
