// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// EventsScalarQuery An individual scalar events query.
type EventsScalarQuery struct {
	// The instructions for what to compute for this query.
	Compute EventsCompute `json:"compute"`
	// A data source that is powered by the Events Platform.
	DataSource EventsDataSource `json:"data_source"`
	// The list of facets on which to split results.
	GroupBy []EventsGroupBy `json:"group_by,omitempty"`
	// The indexes in which to search.
	Indexes []string `json:"indexes,omitempty"`
	// The variable name for use in formulas.
	Name *string `json:"name,omitempty"`
	// Configuration of the search/filter for an events query.
	Search *EventsSearch `json:"search,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewEventsScalarQuery instantiates a new EventsScalarQuery object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewEventsScalarQuery(compute EventsCompute, dataSource EventsDataSource) *EventsScalarQuery {
	this := EventsScalarQuery{}
	this.Compute = compute
	this.DataSource = dataSource
	return &this
}

// NewEventsScalarQueryWithDefaults instantiates a new EventsScalarQuery object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewEventsScalarQueryWithDefaults() *EventsScalarQuery {
	this := EventsScalarQuery{}
	var dataSource EventsDataSource = EVENTSDATASOURCE_LOGS
	this.DataSource = dataSource
	return &this
}

// GetCompute returns the Compute field value.
func (o *EventsScalarQuery) GetCompute() EventsCompute {
	if o == nil {
		var ret EventsCompute
		return ret
	}
	return o.Compute
}

// GetComputeOk returns a tuple with the Compute field value
// and a boolean to check if the value has been set.
func (o *EventsScalarQuery) GetComputeOk() (*EventsCompute, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Compute, true
}

// SetCompute sets field value.
func (o *EventsScalarQuery) SetCompute(v EventsCompute) {
	o.Compute = v
}

// GetDataSource returns the DataSource field value.
func (o *EventsScalarQuery) GetDataSource() EventsDataSource {
	if o == nil {
		var ret EventsDataSource
		return ret
	}
	return o.DataSource
}

// GetDataSourceOk returns a tuple with the DataSource field value
// and a boolean to check if the value has been set.
func (o *EventsScalarQuery) GetDataSourceOk() (*EventsDataSource, bool) {
	if o == nil {
		return nil, false
	}
	return &o.DataSource, true
}

// SetDataSource sets field value.
func (o *EventsScalarQuery) SetDataSource(v EventsDataSource) {
	o.DataSource = v
}

// GetGroupBy returns the GroupBy field value if set, zero value otherwise.
func (o *EventsScalarQuery) GetGroupBy() []EventsGroupBy {
	if o == nil || o.GroupBy == nil {
		var ret []EventsGroupBy
		return ret
	}
	return o.GroupBy
}

// GetGroupByOk returns a tuple with the GroupBy field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *EventsScalarQuery) GetGroupByOk() (*[]EventsGroupBy, bool) {
	if o == nil || o.GroupBy == nil {
		return nil, false
	}
	return &o.GroupBy, true
}

// HasGroupBy returns a boolean if a field has been set.
func (o *EventsScalarQuery) HasGroupBy() bool {
	return o != nil && o.GroupBy != nil
}

// SetGroupBy gets a reference to the given []EventsGroupBy and assigns it to the GroupBy field.
func (o *EventsScalarQuery) SetGroupBy(v []EventsGroupBy) {
	o.GroupBy = v
}

// GetIndexes returns the Indexes field value if set, zero value otherwise.
func (o *EventsScalarQuery) GetIndexes() []string {
	if o == nil || o.Indexes == nil {
		var ret []string
		return ret
	}
	return o.Indexes
}

// GetIndexesOk returns a tuple with the Indexes field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *EventsScalarQuery) GetIndexesOk() (*[]string, bool) {
	if o == nil || o.Indexes == nil {
		return nil, false
	}
	return &o.Indexes, true
}

// HasIndexes returns a boolean if a field has been set.
func (o *EventsScalarQuery) HasIndexes() bool {
	return o != nil && o.Indexes != nil
}

// SetIndexes gets a reference to the given []string and assigns it to the Indexes field.
func (o *EventsScalarQuery) SetIndexes(v []string) {
	o.Indexes = v
}

// GetName returns the Name field value if set, zero value otherwise.
func (o *EventsScalarQuery) GetName() string {
	if o == nil || o.Name == nil {
		var ret string
		return ret
	}
	return *o.Name
}

// GetNameOk returns a tuple with the Name field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *EventsScalarQuery) GetNameOk() (*string, bool) {
	if o == nil || o.Name == nil {
		return nil, false
	}
	return o.Name, true
}

// HasName returns a boolean if a field has been set.
func (o *EventsScalarQuery) HasName() bool {
	return o != nil && o.Name != nil
}

// SetName gets a reference to the given string and assigns it to the Name field.
func (o *EventsScalarQuery) SetName(v string) {
	o.Name = &v
}

// GetSearch returns the Search field value if set, zero value otherwise.
func (o *EventsScalarQuery) GetSearch() EventsSearch {
	if o == nil || o.Search == nil {
		var ret EventsSearch
		return ret
	}
	return *o.Search
}

// GetSearchOk returns a tuple with the Search field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *EventsScalarQuery) GetSearchOk() (*EventsSearch, bool) {
	if o == nil || o.Search == nil {
		return nil, false
	}
	return o.Search, true
}

// HasSearch returns a boolean if a field has been set.
func (o *EventsScalarQuery) HasSearch() bool {
	return o != nil && o.Search != nil
}

// SetSearch gets a reference to the given EventsSearch and assigns it to the Search field.
func (o *EventsScalarQuery) SetSearch(v EventsSearch) {
	o.Search = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o EventsScalarQuery) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["compute"] = o.Compute
	toSerialize["data_source"] = o.DataSource
	if o.GroupBy != nil {
		toSerialize["group_by"] = o.GroupBy
	}
	if o.Indexes != nil {
		toSerialize["indexes"] = o.Indexes
	}
	if o.Name != nil {
		toSerialize["name"] = o.Name
	}
	if o.Search != nil {
		toSerialize["search"] = o.Search
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *EventsScalarQuery) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Compute    *EventsCompute    `json:"compute"`
		DataSource *EventsDataSource `json:"data_source"`
	}{}
	all := struct {
		Compute    EventsCompute    `json:"compute"`
		DataSource EventsDataSource `json:"data_source"`
		GroupBy    []EventsGroupBy  `json:"group_by,omitempty"`
		Indexes    []string         `json:"indexes,omitempty"`
		Name       *string          `json:"name,omitempty"`
		Search     *EventsSearch    `json:"search,omitempty"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Compute == nil {
		return fmt.Errorf("required field compute missing")
	}
	if required.DataSource == nil {
		return fmt.Errorf("required field data_source missing")
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
	if v := all.DataSource; !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	if all.Compute.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Compute = all.Compute
	o.DataSource = all.DataSource
	o.GroupBy = all.GroupBy
	o.Indexes = all.Indexes
	o.Name = all.Name
	if all.Search != nil && all.Search.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Search = all.Search
	return nil
}
