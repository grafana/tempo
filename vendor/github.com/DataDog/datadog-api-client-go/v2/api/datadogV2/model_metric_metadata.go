// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// MetricMetadata Metadata for the metric.
type MetricMetadata struct {
	// Metric origin information.
	Origin *MetricOrigin `json:"origin,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewMetricMetadata instantiates a new MetricMetadata object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewMetricMetadata() *MetricMetadata {
	this := MetricMetadata{}
	return &this
}

// NewMetricMetadataWithDefaults instantiates a new MetricMetadata object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewMetricMetadataWithDefaults() *MetricMetadata {
	this := MetricMetadata{}
	return &this
}

// GetOrigin returns the Origin field value if set, zero value otherwise.
func (o *MetricMetadata) GetOrigin() MetricOrigin {
	if o == nil || o.Origin == nil {
		var ret MetricOrigin
		return ret
	}
	return *o.Origin
}

// GetOriginOk returns a tuple with the Origin field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MetricMetadata) GetOriginOk() (*MetricOrigin, bool) {
	if o == nil || o.Origin == nil {
		return nil, false
	}
	return o.Origin, true
}

// HasOrigin returns a boolean if a field has been set.
func (o *MetricMetadata) HasOrigin() bool {
	return o != nil && o.Origin != nil
}

// SetOrigin gets a reference to the given MetricOrigin and assigns it to the Origin field.
func (o *MetricMetadata) SetOrigin(v MetricOrigin) {
	o.Origin = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o MetricMetadata) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Origin != nil {
		toSerialize["origin"] = o.Origin
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *MetricMetadata) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Origin *MetricOrigin `json:"origin,omitempty"`
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
	if all.Origin != nil && all.Origin.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Origin = all.Origin
	return nil
}
