// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// MetricIngestedIndexedVolume Object for a single metric's ingested and indexed volume.
type MetricIngestedIndexedVolume struct {
	// Object containing the definition of a metric's ingested and indexed volume.
	Attributes *MetricIngestedIndexedVolumeAttributes `json:"attributes,omitempty"`
	// The metric name for this resource.
	Id *string `json:"id,omitempty"`
	// The metric ingested and indexed volume type.
	Type *MetricIngestedIndexedVolumeType `json:"type,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewMetricIngestedIndexedVolume instantiates a new MetricIngestedIndexedVolume object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewMetricIngestedIndexedVolume() *MetricIngestedIndexedVolume {
	this := MetricIngestedIndexedVolume{}
	var typeVar MetricIngestedIndexedVolumeType = METRICINGESTEDINDEXEDVOLUMETYPE_METRIC_VOLUMES
	this.Type = &typeVar
	return &this
}

// NewMetricIngestedIndexedVolumeWithDefaults instantiates a new MetricIngestedIndexedVolume object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewMetricIngestedIndexedVolumeWithDefaults() *MetricIngestedIndexedVolume {
	this := MetricIngestedIndexedVolume{}
	var typeVar MetricIngestedIndexedVolumeType = METRICINGESTEDINDEXEDVOLUMETYPE_METRIC_VOLUMES
	this.Type = &typeVar
	return &this
}

// GetAttributes returns the Attributes field value if set, zero value otherwise.
func (o *MetricIngestedIndexedVolume) GetAttributes() MetricIngestedIndexedVolumeAttributes {
	if o == nil || o.Attributes == nil {
		var ret MetricIngestedIndexedVolumeAttributes
		return ret
	}
	return *o.Attributes
}

// GetAttributesOk returns a tuple with the Attributes field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MetricIngestedIndexedVolume) GetAttributesOk() (*MetricIngestedIndexedVolumeAttributes, bool) {
	if o == nil || o.Attributes == nil {
		return nil, false
	}
	return o.Attributes, true
}

// HasAttributes returns a boolean if a field has been set.
func (o *MetricIngestedIndexedVolume) HasAttributes() bool {
	return o != nil && o.Attributes != nil
}

// SetAttributes gets a reference to the given MetricIngestedIndexedVolumeAttributes and assigns it to the Attributes field.
func (o *MetricIngestedIndexedVolume) SetAttributes(v MetricIngestedIndexedVolumeAttributes) {
	o.Attributes = &v
}

// GetId returns the Id field value if set, zero value otherwise.
func (o *MetricIngestedIndexedVolume) GetId() string {
	if o == nil || o.Id == nil {
		var ret string
		return ret
	}
	return *o.Id
}

// GetIdOk returns a tuple with the Id field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MetricIngestedIndexedVolume) GetIdOk() (*string, bool) {
	if o == nil || o.Id == nil {
		return nil, false
	}
	return o.Id, true
}

// HasId returns a boolean if a field has been set.
func (o *MetricIngestedIndexedVolume) HasId() bool {
	return o != nil && o.Id != nil
}

// SetId gets a reference to the given string and assigns it to the Id field.
func (o *MetricIngestedIndexedVolume) SetId(v string) {
	o.Id = &v
}

// GetType returns the Type field value if set, zero value otherwise.
func (o *MetricIngestedIndexedVolume) GetType() MetricIngestedIndexedVolumeType {
	if o == nil || o.Type == nil {
		var ret MetricIngestedIndexedVolumeType
		return ret
	}
	return *o.Type
}

// GetTypeOk returns a tuple with the Type field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MetricIngestedIndexedVolume) GetTypeOk() (*MetricIngestedIndexedVolumeType, bool) {
	if o == nil || o.Type == nil {
		return nil, false
	}
	return o.Type, true
}

// HasType returns a boolean if a field has been set.
func (o *MetricIngestedIndexedVolume) HasType() bool {
	return o != nil && o.Type != nil
}

// SetType gets a reference to the given MetricIngestedIndexedVolumeType and assigns it to the Type field.
func (o *MetricIngestedIndexedVolume) SetType(v MetricIngestedIndexedVolumeType) {
	o.Type = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o MetricIngestedIndexedVolume) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Attributes != nil {
		toSerialize["attributes"] = o.Attributes
	}
	if o.Id != nil {
		toSerialize["id"] = o.Id
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
func (o *MetricIngestedIndexedVolume) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Attributes *MetricIngestedIndexedVolumeAttributes `json:"attributes,omitempty"`
		Id         *string                                `json:"id,omitempty"`
		Type       *MetricIngestedIndexedVolumeType       `json:"type,omitempty"`
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
	if v := all.Type; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	if all.Attributes != nil && all.Attributes.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Attributes = all.Attributes
	o.Id = all.Id
	o.Type = all.Type
	return nil
}
