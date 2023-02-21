// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV1

import (
	"encoding/json"
	"fmt"
)

// ListStreamQuery Updated list stream widget.
type ListStreamQuery struct {
	// Compute configuration for the List Stream Widget. Compute can be used only with the logs_transaction_stream (from 1 to 5 items) list stream source.
	Compute []ListStreamComputeItems `json:"compute,omitempty"`
	// Source from which to query items to display in the stream.
	DataSource ListStreamSource `json:"data_source"`
	// Size to use to display an event.
	EventSize *WidgetEventSize `json:"event_size,omitempty"`
	// Group by configuration for the List Stream Widget. Group by can be used only with logs_pattern_stream (up to 3 items) or logs_transaction_stream (one group by item is required) list stream source.
	GroupBy []ListStreamGroupByItems `json:"group_by,omitempty"`
	// List of indexes.
	Indexes []string `json:"indexes,omitempty"`
	// Widget query.
	QueryString string `json:"query_string"`
	// Option for storage location. Feature in Private Beta.
	Storage *string `json:"storage,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewListStreamQuery instantiates a new ListStreamQuery object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewListStreamQuery(dataSource ListStreamSource, queryString string) *ListStreamQuery {
	this := ListStreamQuery{}
	this.DataSource = dataSource
	this.QueryString = queryString
	return &this
}

// NewListStreamQueryWithDefaults instantiates a new ListStreamQuery object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewListStreamQueryWithDefaults() *ListStreamQuery {
	this := ListStreamQuery{}
	var dataSource ListStreamSource = LISTSTREAMSOURCE_APM_ISSUE_STREAM
	this.DataSource = dataSource
	return &this
}

// GetCompute returns the Compute field value if set, zero value otherwise.
func (o *ListStreamQuery) GetCompute() []ListStreamComputeItems {
	if o == nil || o.Compute == nil {
		var ret []ListStreamComputeItems
		return ret
	}
	return o.Compute
}

// GetComputeOk returns a tuple with the Compute field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ListStreamQuery) GetComputeOk() (*[]ListStreamComputeItems, bool) {
	if o == nil || o.Compute == nil {
		return nil, false
	}
	return &o.Compute, true
}

// HasCompute returns a boolean if a field has been set.
func (o *ListStreamQuery) HasCompute() bool {
	return o != nil && o.Compute != nil
}

// SetCompute gets a reference to the given []ListStreamComputeItems and assigns it to the Compute field.
func (o *ListStreamQuery) SetCompute(v []ListStreamComputeItems) {
	o.Compute = v
}

// GetDataSource returns the DataSource field value.
func (o *ListStreamQuery) GetDataSource() ListStreamSource {
	if o == nil {
		var ret ListStreamSource
		return ret
	}
	return o.DataSource
}

// GetDataSourceOk returns a tuple with the DataSource field value
// and a boolean to check if the value has been set.
func (o *ListStreamQuery) GetDataSourceOk() (*ListStreamSource, bool) {
	if o == nil {
		return nil, false
	}
	return &o.DataSource, true
}

// SetDataSource sets field value.
func (o *ListStreamQuery) SetDataSource(v ListStreamSource) {
	o.DataSource = v
}

// GetEventSize returns the EventSize field value if set, zero value otherwise.
func (o *ListStreamQuery) GetEventSize() WidgetEventSize {
	if o == nil || o.EventSize == nil {
		var ret WidgetEventSize
		return ret
	}
	return *o.EventSize
}

// GetEventSizeOk returns a tuple with the EventSize field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ListStreamQuery) GetEventSizeOk() (*WidgetEventSize, bool) {
	if o == nil || o.EventSize == nil {
		return nil, false
	}
	return o.EventSize, true
}

// HasEventSize returns a boolean if a field has been set.
func (o *ListStreamQuery) HasEventSize() bool {
	return o != nil && o.EventSize != nil
}

// SetEventSize gets a reference to the given WidgetEventSize and assigns it to the EventSize field.
func (o *ListStreamQuery) SetEventSize(v WidgetEventSize) {
	o.EventSize = &v
}

// GetGroupBy returns the GroupBy field value if set, zero value otherwise.
func (o *ListStreamQuery) GetGroupBy() []ListStreamGroupByItems {
	if o == nil || o.GroupBy == nil {
		var ret []ListStreamGroupByItems
		return ret
	}
	return o.GroupBy
}

// GetGroupByOk returns a tuple with the GroupBy field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ListStreamQuery) GetGroupByOk() (*[]ListStreamGroupByItems, bool) {
	if o == nil || o.GroupBy == nil {
		return nil, false
	}
	return &o.GroupBy, true
}

// HasGroupBy returns a boolean if a field has been set.
func (o *ListStreamQuery) HasGroupBy() bool {
	return o != nil && o.GroupBy != nil
}

// SetGroupBy gets a reference to the given []ListStreamGroupByItems and assigns it to the GroupBy field.
func (o *ListStreamQuery) SetGroupBy(v []ListStreamGroupByItems) {
	o.GroupBy = v
}

// GetIndexes returns the Indexes field value if set, zero value otherwise.
func (o *ListStreamQuery) GetIndexes() []string {
	if o == nil || o.Indexes == nil {
		var ret []string
		return ret
	}
	return o.Indexes
}

// GetIndexesOk returns a tuple with the Indexes field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ListStreamQuery) GetIndexesOk() (*[]string, bool) {
	if o == nil || o.Indexes == nil {
		return nil, false
	}
	return &o.Indexes, true
}

// HasIndexes returns a boolean if a field has been set.
func (o *ListStreamQuery) HasIndexes() bool {
	return o != nil && o.Indexes != nil
}

// SetIndexes gets a reference to the given []string and assigns it to the Indexes field.
func (o *ListStreamQuery) SetIndexes(v []string) {
	o.Indexes = v
}

// GetQueryString returns the QueryString field value.
func (o *ListStreamQuery) GetQueryString() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.QueryString
}

// GetQueryStringOk returns a tuple with the QueryString field value
// and a boolean to check if the value has been set.
func (o *ListStreamQuery) GetQueryStringOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.QueryString, true
}

// SetQueryString sets field value.
func (o *ListStreamQuery) SetQueryString(v string) {
	o.QueryString = v
}

// GetStorage returns the Storage field value if set, zero value otherwise.
func (o *ListStreamQuery) GetStorage() string {
	if o == nil || o.Storage == nil {
		var ret string
		return ret
	}
	return *o.Storage
}

// GetStorageOk returns a tuple with the Storage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ListStreamQuery) GetStorageOk() (*string, bool) {
	if o == nil || o.Storage == nil {
		return nil, false
	}
	return o.Storage, true
}

// HasStorage returns a boolean if a field has been set.
func (o *ListStreamQuery) HasStorage() bool {
	return o != nil && o.Storage != nil
}

// SetStorage gets a reference to the given string and assigns it to the Storage field.
func (o *ListStreamQuery) SetStorage(v string) {
	o.Storage = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o ListStreamQuery) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Compute != nil {
		toSerialize["compute"] = o.Compute
	}
	toSerialize["data_source"] = o.DataSource
	if o.EventSize != nil {
		toSerialize["event_size"] = o.EventSize
	}
	if o.GroupBy != nil {
		toSerialize["group_by"] = o.GroupBy
	}
	if o.Indexes != nil {
		toSerialize["indexes"] = o.Indexes
	}
	toSerialize["query_string"] = o.QueryString
	if o.Storage != nil {
		toSerialize["storage"] = o.Storage
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *ListStreamQuery) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		DataSource  *ListStreamSource `json:"data_source"`
		QueryString *string           `json:"query_string"`
	}{}
	all := struct {
		Compute     []ListStreamComputeItems `json:"compute,omitempty"`
		DataSource  ListStreamSource         `json:"data_source"`
		EventSize   *WidgetEventSize         `json:"event_size,omitempty"`
		GroupBy     []ListStreamGroupByItems `json:"group_by,omitempty"`
		Indexes     []string                 `json:"indexes,omitempty"`
		QueryString string                   `json:"query_string"`
		Storage     *string                  `json:"storage,omitempty"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.DataSource == nil {
		return fmt.Errorf("required field data_source missing")
	}
	if required.QueryString == nil {
		return fmt.Errorf("required field query_string missing")
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
	if v := all.EventSize; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.Compute = all.Compute
	o.DataSource = all.DataSource
	o.EventSize = all.EventSize
	o.GroupBy = all.GroupBy
	o.Indexes = all.Indexes
	o.QueryString = all.QueryString
	o.Storage = all.Storage
	return nil
}
