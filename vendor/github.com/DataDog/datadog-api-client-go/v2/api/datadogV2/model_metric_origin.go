// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// MetricOrigin Metric origin information.
type MetricOrigin struct {
	// The origin metric type code
	MetricType *int32 `json:"metric_type,omitempty"`
	// The origin product code
	Product *int32 `json:"product,omitempty"`
	// The origin service code
	Service *int32 `json:"service,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewMetricOrigin instantiates a new MetricOrigin object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewMetricOrigin() *MetricOrigin {
	this := MetricOrigin{}
	var metricType int32 = 0
	this.MetricType = &metricType
	var product int32 = 0
	this.Product = &product
	var service int32 = 0
	this.Service = &service
	return &this
}

// NewMetricOriginWithDefaults instantiates a new MetricOrigin object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewMetricOriginWithDefaults() *MetricOrigin {
	this := MetricOrigin{}
	var metricType int32 = 0
	this.MetricType = &metricType
	var product int32 = 0
	this.Product = &product
	var service int32 = 0
	this.Service = &service
	return &this
}

// GetMetricType returns the MetricType field value if set, zero value otherwise.
func (o *MetricOrigin) GetMetricType() int32 {
	if o == nil || o.MetricType == nil {
		var ret int32
		return ret
	}
	return *o.MetricType
}

// GetMetricTypeOk returns a tuple with the MetricType field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MetricOrigin) GetMetricTypeOk() (*int32, bool) {
	if o == nil || o.MetricType == nil {
		return nil, false
	}
	return o.MetricType, true
}

// HasMetricType returns a boolean if a field has been set.
func (o *MetricOrigin) HasMetricType() bool {
	return o != nil && o.MetricType != nil
}

// SetMetricType gets a reference to the given int32 and assigns it to the MetricType field.
func (o *MetricOrigin) SetMetricType(v int32) {
	o.MetricType = &v
}

// GetProduct returns the Product field value if set, zero value otherwise.
func (o *MetricOrigin) GetProduct() int32 {
	if o == nil || o.Product == nil {
		var ret int32
		return ret
	}
	return *o.Product
}

// GetProductOk returns a tuple with the Product field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MetricOrigin) GetProductOk() (*int32, bool) {
	if o == nil || o.Product == nil {
		return nil, false
	}
	return o.Product, true
}

// HasProduct returns a boolean if a field has been set.
func (o *MetricOrigin) HasProduct() bool {
	return o != nil && o.Product != nil
}

// SetProduct gets a reference to the given int32 and assigns it to the Product field.
func (o *MetricOrigin) SetProduct(v int32) {
	o.Product = &v
}

// GetService returns the Service field value if set, zero value otherwise.
func (o *MetricOrigin) GetService() int32 {
	if o == nil || o.Service == nil {
		var ret int32
		return ret
	}
	return *o.Service
}

// GetServiceOk returns a tuple with the Service field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MetricOrigin) GetServiceOk() (*int32, bool) {
	if o == nil || o.Service == nil {
		return nil, false
	}
	return o.Service, true
}

// HasService returns a boolean if a field has been set.
func (o *MetricOrigin) HasService() bool {
	return o != nil && o.Service != nil
}

// SetService gets a reference to the given int32 and assigns it to the Service field.
func (o *MetricOrigin) SetService(v int32) {
	o.Service = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o MetricOrigin) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.MetricType != nil {
		toSerialize["metric_type"] = o.MetricType
	}
	if o.Product != nil {
		toSerialize["product"] = o.Product
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
func (o *MetricOrigin) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		MetricType *int32 `json:"metric_type,omitempty"`
		Product    *int32 `json:"product,omitempty"`
		Service    *int32 `json:"service,omitempty"`
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
	o.MetricType = all.MetricType
	o.Product = all.Product
	o.Service = all.Service
	return nil
}
