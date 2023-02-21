// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV1

import (
	"encoding/json"
)

// TopologyQuery Query to service-based topology data sources like the service map or data streams.
type TopologyQuery struct {
	// Name of the data source
	DataSource *TopologyQueryDataSource `json:"data_source,omitempty"`
	// Your environment and primary tag (or * if enabled for your account).
	Filters []string `json:"filters,omitempty"`
	// Name of the service
	Service *string `json:"service,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewTopologyQuery instantiates a new TopologyQuery object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewTopologyQuery() *TopologyQuery {
	this := TopologyQuery{}
	return &this
}

// NewTopologyQueryWithDefaults instantiates a new TopologyQuery object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewTopologyQueryWithDefaults() *TopologyQuery {
	this := TopologyQuery{}
	return &this
}

// GetDataSource returns the DataSource field value if set, zero value otherwise.
func (o *TopologyQuery) GetDataSource() TopologyQueryDataSource {
	if o == nil || o.DataSource == nil {
		var ret TopologyQueryDataSource
		return ret
	}
	return *o.DataSource
}

// GetDataSourceOk returns a tuple with the DataSource field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *TopologyQuery) GetDataSourceOk() (*TopologyQueryDataSource, bool) {
	if o == nil || o.DataSource == nil {
		return nil, false
	}
	return o.DataSource, true
}

// HasDataSource returns a boolean if a field has been set.
func (o *TopologyQuery) HasDataSource() bool {
	return o != nil && o.DataSource != nil
}

// SetDataSource gets a reference to the given TopologyQueryDataSource and assigns it to the DataSource field.
func (o *TopologyQuery) SetDataSource(v TopologyQueryDataSource) {
	o.DataSource = &v
}

// GetFilters returns the Filters field value if set, zero value otherwise.
func (o *TopologyQuery) GetFilters() []string {
	if o == nil || o.Filters == nil {
		var ret []string
		return ret
	}
	return o.Filters
}

// GetFiltersOk returns a tuple with the Filters field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *TopologyQuery) GetFiltersOk() (*[]string, bool) {
	if o == nil || o.Filters == nil {
		return nil, false
	}
	return &o.Filters, true
}

// HasFilters returns a boolean if a field has been set.
func (o *TopologyQuery) HasFilters() bool {
	return o != nil && o.Filters != nil
}

// SetFilters gets a reference to the given []string and assigns it to the Filters field.
func (o *TopologyQuery) SetFilters(v []string) {
	o.Filters = v
}

// GetService returns the Service field value if set, zero value otherwise.
func (o *TopologyQuery) GetService() string {
	if o == nil || o.Service == nil {
		var ret string
		return ret
	}
	return *o.Service
}

// GetServiceOk returns a tuple with the Service field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *TopologyQuery) GetServiceOk() (*string, bool) {
	if o == nil || o.Service == nil {
		return nil, false
	}
	return o.Service, true
}

// HasService returns a boolean if a field has been set.
func (o *TopologyQuery) HasService() bool {
	return o != nil && o.Service != nil
}

// SetService gets a reference to the given string and assigns it to the Service field.
func (o *TopologyQuery) SetService(v string) {
	o.Service = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o TopologyQuery) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.DataSource != nil {
		toSerialize["data_source"] = o.DataSource
	}
	if o.Filters != nil {
		toSerialize["filters"] = o.Filters
	}
	if o.Service != nil {
		toSerialize["service"] = o.Service
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *TopologyQuery) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		DataSource *TopologyQueryDataSource `json:"data_source,omitempty"`
		Filters    []string                 `json:"filters,omitempty"`
		Service    *string                  `json:"service,omitempty"`
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
	if v := all.DataSource; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.DataSource = all.DataSource
	o.Filters = all.Filters
	o.Service = all.Service
	return nil
}
