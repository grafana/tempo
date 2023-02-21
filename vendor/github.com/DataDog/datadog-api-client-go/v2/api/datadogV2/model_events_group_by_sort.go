// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// EventsGroupBySort The dimension by which to sort a query's results.
type EventsGroupBySort struct {
	// The type of aggregation that can be performed on events-based queries.
	Aggregation EventsAggregation `json:"aggregation"`
	// Metric whose calculated value should be used to define the sort order of a query's results.
	Metric *string `json:"metric,omitempty"`
	// Direction of sort.
	Order *QuerySortOrder `json:"order,omitempty"`
	// The type of sort to use on the calculated value.
	Type *EventsSortType `json:"type,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewEventsGroupBySort instantiates a new EventsGroupBySort object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewEventsGroupBySort(aggregation EventsAggregation) *EventsGroupBySort {
	this := EventsGroupBySort{}
	this.Aggregation = aggregation
	var order QuerySortOrder = QUERYSORTORDER_DESC
	this.Order = &order
	return &this
}

// NewEventsGroupBySortWithDefaults instantiates a new EventsGroupBySort object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewEventsGroupBySortWithDefaults() *EventsGroupBySort {
	this := EventsGroupBySort{}
	var aggregation EventsAggregation = EVENTSAGGREGATION_COUNT
	this.Aggregation = aggregation
	var order QuerySortOrder = QUERYSORTORDER_DESC
	this.Order = &order
	return &this
}

// GetAggregation returns the Aggregation field value.
func (o *EventsGroupBySort) GetAggregation() EventsAggregation {
	if o == nil {
		var ret EventsAggregation
		return ret
	}
	return o.Aggregation
}

// GetAggregationOk returns a tuple with the Aggregation field value
// and a boolean to check if the value has been set.
func (o *EventsGroupBySort) GetAggregationOk() (*EventsAggregation, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Aggregation, true
}

// SetAggregation sets field value.
func (o *EventsGroupBySort) SetAggregation(v EventsAggregation) {
	o.Aggregation = v
}

// GetMetric returns the Metric field value if set, zero value otherwise.
func (o *EventsGroupBySort) GetMetric() string {
	if o == nil || o.Metric == nil {
		var ret string
		return ret
	}
	return *o.Metric
}

// GetMetricOk returns a tuple with the Metric field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *EventsGroupBySort) GetMetricOk() (*string, bool) {
	if o == nil || o.Metric == nil {
		return nil, false
	}
	return o.Metric, true
}

// HasMetric returns a boolean if a field has been set.
func (o *EventsGroupBySort) HasMetric() bool {
	return o != nil && o.Metric != nil
}

// SetMetric gets a reference to the given string and assigns it to the Metric field.
func (o *EventsGroupBySort) SetMetric(v string) {
	o.Metric = &v
}

// GetOrder returns the Order field value if set, zero value otherwise.
func (o *EventsGroupBySort) GetOrder() QuerySortOrder {
	if o == nil || o.Order == nil {
		var ret QuerySortOrder
		return ret
	}
	return *o.Order
}

// GetOrderOk returns a tuple with the Order field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *EventsGroupBySort) GetOrderOk() (*QuerySortOrder, bool) {
	if o == nil || o.Order == nil {
		return nil, false
	}
	return o.Order, true
}

// HasOrder returns a boolean if a field has been set.
func (o *EventsGroupBySort) HasOrder() bool {
	return o != nil && o.Order != nil
}

// SetOrder gets a reference to the given QuerySortOrder and assigns it to the Order field.
func (o *EventsGroupBySort) SetOrder(v QuerySortOrder) {
	o.Order = &v
}

// GetType returns the Type field value if set, zero value otherwise.
func (o *EventsGroupBySort) GetType() EventsSortType {
	if o == nil || o.Type == nil {
		var ret EventsSortType
		return ret
	}
	return *o.Type
}

// GetTypeOk returns a tuple with the Type field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *EventsGroupBySort) GetTypeOk() (*EventsSortType, bool) {
	if o == nil || o.Type == nil {
		return nil, false
	}
	return o.Type, true
}

// HasType returns a boolean if a field has been set.
func (o *EventsGroupBySort) HasType() bool {
	return o != nil && o.Type != nil
}

// SetType gets a reference to the given EventsSortType and assigns it to the Type field.
func (o *EventsGroupBySort) SetType(v EventsSortType) {
	o.Type = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o EventsGroupBySort) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["aggregation"] = o.Aggregation
	if o.Metric != nil {
		toSerialize["metric"] = o.Metric
	}
	if o.Order != nil {
		toSerialize["order"] = o.Order
	}
	if o.Type != nil {
		toSerialize["type"] = o.Type
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *EventsGroupBySort) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Aggregation *EventsAggregation `json:"aggregation"`
	}{}
	all := struct {
		Aggregation EventsAggregation `json:"aggregation"`
		Metric      *string           `json:"metric,omitempty"`
		Order       *QuerySortOrder   `json:"order,omitempty"`
		Type        *EventsSortType   `json:"type,omitempty"`
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
	if v := all.Order; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	if v := all.Type; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.Aggregation = all.Aggregation
	o.Metric = all.Metric
	o.Order = all.Order
	o.Type = all.Type
	return nil
}
