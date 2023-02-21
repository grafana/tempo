// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// MetricBulkTagConfigResponse Wrapper for a single bulk tag configuration status response.
type MetricBulkTagConfigResponse struct {
	// The status of a request to bulk configure metric tags.
	// It contains the fields from the original request for reference.
	Data *MetricBulkTagConfigStatus `json:"data,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewMetricBulkTagConfigResponse instantiates a new MetricBulkTagConfigResponse object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewMetricBulkTagConfigResponse() *MetricBulkTagConfigResponse {
	this := MetricBulkTagConfigResponse{}
	return &this
}

// NewMetricBulkTagConfigResponseWithDefaults instantiates a new MetricBulkTagConfigResponse object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewMetricBulkTagConfigResponseWithDefaults() *MetricBulkTagConfigResponse {
	this := MetricBulkTagConfigResponse{}
	return &this
}

// GetData returns the Data field value if set, zero value otherwise.
func (o *MetricBulkTagConfigResponse) GetData() MetricBulkTagConfigStatus {
	if o == nil || o.Data == nil {
		var ret MetricBulkTagConfigStatus
		return ret
	}
	return *o.Data
}

// GetDataOk returns a tuple with the Data field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MetricBulkTagConfigResponse) GetDataOk() (*MetricBulkTagConfigStatus, bool) {
	if o == nil || o.Data == nil {
		return nil, false
	}
	return o.Data, true
}

// HasData returns a boolean if a field has been set.
func (o *MetricBulkTagConfigResponse) HasData() bool {
	return o != nil && o.Data != nil
}

// SetData gets a reference to the given MetricBulkTagConfigStatus and assigns it to the Data field.
func (o *MetricBulkTagConfigResponse) SetData(v MetricBulkTagConfigStatus) {
	o.Data = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o MetricBulkTagConfigResponse) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Data != nil {
		toSerialize["data"] = o.Data
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *MetricBulkTagConfigResponse) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Data *MetricBulkTagConfigStatus `json:"data,omitempty"`
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
	if all.Data != nil && all.Data.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Data = all.Data
	return nil
}
