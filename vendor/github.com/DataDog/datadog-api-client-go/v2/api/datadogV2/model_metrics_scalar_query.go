// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// MetricsScalarQuery An individual scalar metrics query.
type MetricsScalarQuery struct {
	// The type of aggregation that can be performed on metrics queries.
	Aggregator MetricsAggregator `json:"aggregator"`
	// A data source that is powered by the Metrics platform.
	DataSource MetricsDataSource `json:"data_source"`
	// The variable name for use in formulas.
	Name *string `json:"name,omitempty"`
	// A classic metrics query string.
	Query string `json:"query"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewMetricsScalarQuery instantiates a new MetricsScalarQuery object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewMetricsScalarQuery(aggregator MetricsAggregator, dataSource MetricsDataSource, query string) *MetricsScalarQuery {
	this := MetricsScalarQuery{}
	this.Aggregator = aggregator
	this.DataSource = dataSource
	this.Query = query
	return &this
}

// NewMetricsScalarQueryWithDefaults instantiates a new MetricsScalarQuery object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewMetricsScalarQueryWithDefaults() *MetricsScalarQuery {
	this := MetricsScalarQuery{}
	var aggregator MetricsAggregator = METRICSAGGREGATOR_AVG
	this.Aggregator = aggregator
	var dataSource MetricsDataSource = METRICSDATASOURCE_METRICS
	this.DataSource = dataSource
	return &this
}

// GetAggregator returns the Aggregator field value.
func (o *MetricsScalarQuery) GetAggregator() MetricsAggregator {
	if o == nil {
		var ret MetricsAggregator
		return ret
	}
	return o.Aggregator
}

// GetAggregatorOk returns a tuple with the Aggregator field value
// and a boolean to check if the value has been set.
func (o *MetricsScalarQuery) GetAggregatorOk() (*MetricsAggregator, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Aggregator, true
}

// SetAggregator sets field value.
func (o *MetricsScalarQuery) SetAggregator(v MetricsAggregator) {
	o.Aggregator = v
}

// GetDataSource returns the DataSource field value.
func (o *MetricsScalarQuery) GetDataSource() MetricsDataSource {
	if o == nil {
		var ret MetricsDataSource
		return ret
	}
	return o.DataSource
}

// GetDataSourceOk returns a tuple with the DataSource field value
// and a boolean to check if the value has been set.
func (o *MetricsScalarQuery) GetDataSourceOk() (*MetricsDataSource, bool) {
	if o == nil {
		return nil, false
	}
	return &o.DataSource, true
}

// SetDataSource sets field value.
func (o *MetricsScalarQuery) SetDataSource(v MetricsDataSource) {
	o.DataSource = v
}

// GetName returns the Name field value if set, zero value otherwise.
func (o *MetricsScalarQuery) GetName() string {
	if o == nil || o.Name == nil {
		var ret string
		return ret
	}
	return *o.Name
}

// GetNameOk returns a tuple with the Name field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MetricsScalarQuery) GetNameOk() (*string, bool) {
	if o == nil || o.Name == nil {
		return nil, false
	}
	return o.Name, true
}

// HasName returns a boolean if a field has been set.
func (o *MetricsScalarQuery) HasName() bool {
	return o != nil && o.Name != nil
}

// SetName gets a reference to the given string and assigns it to the Name field.
func (o *MetricsScalarQuery) SetName(v string) {
	o.Name = &v
}

// GetQuery returns the Query field value.
func (o *MetricsScalarQuery) GetQuery() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Query
}

// GetQueryOk returns a tuple with the Query field value
// and a boolean to check if the value has been set.
func (o *MetricsScalarQuery) GetQueryOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Query, true
}

// SetQuery sets field value.
func (o *MetricsScalarQuery) SetQuery(v string) {
	o.Query = v
}

// MarshalJSON serializes the struct using spec logic.
func (o MetricsScalarQuery) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["aggregator"] = o.Aggregator
	toSerialize["data_source"] = o.DataSource
	if o.Name != nil {
		toSerialize["name"] = o.Name
	}
	toSerialize["query"] = o.Query

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *MetricsScalarQuery) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Aggregator *MetricsAggregator `json:"aggregator"`
		DataSource *MetricsDataSource `json:"data_source"`
		Query      *string            `json:"query"`
	}{}
	all := struct {
		Aggregator MetricsAggregator `json:"aggregator"`
		DataSource MetricsDataSource `json:"data_source"`
		Name       *string           `json:"name,omitempty"`
		Query      string            `json:"query"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Aggregator == nil {
		return fmt.Errorf("required field aggregator missing")
	}
	if required.DataSource == nil {
		return fmt.Errorf("required field data_source missing")
	}
	if required.Query == nil {
		return fmt.Errorf("required field query missing")
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
	if v := all.Aggregator; !v.IsValid() {
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
	o.Aggregator = all.Aggregator
	o.DataSource = all.DataSource
	o.Name = all.Name
	o.Query = all.Query
	return nil
}
