// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// MetricSuggestedTagsAndAggregations Object for a single metric's actively queried tags and aggregations.
type MetricSuggestedTagsAndAggregations struct {
	// Object containing the definition of a metric's actively queried tags and aggregations.
	Attributes *MetricSuggestedTagsAttributes `json:"attributes,omitempty"`
	// The metric name for this resource.
	Id *string `json:"id,omitempty"`
	// The metric actively queried configuration resource type.
	Type *MetricActiveConfigurationType `json:"type,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewMetricSuggestedTagsAndAggregations instantiates a new MetricSuggestedTagsAndAggregations object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewMetricSuggestedTagsAndAggregations() *MetricSuggestedTagsAndAggregations {
	this := MetricSuggestedTagsAndAggregations{}
	var typeVar MetricActiveConfigurationType = METRICACTIVECONFIGURATIONTYPE_ACTIVELY_QUERIED_CONFIGURATIONS
	this.Type = &typeVar
	return &this
}

// NewMetricSuggestedTagsAndAggregationsWithDefaults instantiates a new MetricSuggestedTagsAndAggregations object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewMetricSuggestedTagsAndAggregationsWithDefaults() *MetricSuggestedTagsAndAggregations {
	this := MetricSuggestedTagsAndAggregations{}
	var typeVar MetricActiveConfigurationType = METRICACTIVECONFIGURATIONTYPE_ACTIVELY_QUERIED_CONFIGURATIONS
	this.Type = &typeVar
	return &this
}

// GetAttributes returns the Attributes field value if set, zero value otherwise.
func (o *MetricSuggestedTagsAndAggregations) GetAttributes() MetricSuggestedTagsAttributes {
	if o == nil || o.Attributes == nil {
		var ret MetricSuggestedTagsAttributes
		return ret
	}
	return *o.Attributes
}

// GetAttributesOk returns a tuple with the Attributes field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MetricSuggestedTagsAndAggregations) GetAttributesOk() (*MetricSuggestedTagsAttributes, bool) {
	if o == nil || o.Attributes == nil {
		return nil, false
	}
	return o.Attributes, true
}

// HasAttributes returns a boolean if a field has been set.
func (o *MetricSuggestedTagsAndAggregations) HasAttributes() bool {
	return o != nil && o.Attributes != nil
}

// SetAttributes gets a reference to the given MetricSuggestedTagsAttributes and assigns it to the Attributes field.
func (o *MetricSuggestedTagsAndAggregations) SetAttributes(v MetricSuggestedTagsAttributes) {
	o.Attributes = &v
}

// GetId returns the Id field value if set, zero value otherwise.
func (o *MetricSuggestedTagsAndAggregations) GetId() string {
	if o == nil || o.Id == nil {
		var ret string
		return ret
	}
	return *o.Id
}

// GetIdOk returns a tuple with the Id field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MetricSuggestedTagsAndAggregations) GetIdOk() (*string, bool) {
	if o == nil || o.Id == nil {
		return nil, false
	}
	return o.Id, true
}

// HasId returns a boolean if a field has been set.
func (o *MetricSuggestedTagsAndAggregations) HasId() bool {
	return o != nil && o.Id != nil
}

// SetId gets a reference to the given string and assigns it to the Id field.
func (o *MetricSuggestedTagsAndAggregations) SetId(v string) {
	o.Id = &v
}

// GetType returns the Type field value if set, zero value otherwise.
func (o *MetricSuggestedTagsAndAggregations) GetType() MetricActiveConfigurationType {
	if o == nil || o.Type == nil {
		var ret MetricActiveConfigurationType
		return ret
	}
	return *o.Type
}

// GetTypeOk returns a tuple with the Type field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MetricSuggestedTagsAndAggregations) GetTypeOk() (*MetricActiveConfigurationType, bool) {
	if o == nil || o.Type == nil {
		return nil, false
	}
	return o.Type, true
}

// HasType returns a boolean if a field has been set.
func (o *MetricSuggestedTagsAndAggregations) HasType() bool {
	return o != nil && o.Type != nil
}

// SetType gets a reference to the given MetricActiveConfigurationType and assigns it to the Type field.
func (o *MetricSuggestedTagsAndAggregations) SetType(v MetricActiveConfigurationType) {
	o.Type = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o MetricSuggestedTagsAndAggregations) MarshalJSON() ([]byte, error) {
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
func (o *MetricSuggestedTagsAndAggregations) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Attributes *MetricSuggestedTagsAttributes `json:"attributes,omitempty"`
		Id         *string                        `json:"id,omitempty"`
		Type       *MetricActiveConfigurationType `json:"type,omitempty"`
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
