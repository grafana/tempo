// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// UsageDataObject Usage data.
type UsageDataObject struct {
	// Usage attributes data.
	Attributes *UsageAttributesObject `json:"attributes,omitempty"`
	// Unique ID of the response.
	Id *string `json:"id,omitempty"`
	// Type of usage data.
	Type *UsageTimeSeriesType `json:"type,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewUsageDataObject instantiates a new UsageDataObject object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewUsageDataObject() *UsageDataObject {
	this := UsageDataObject{}
	var typeVar UsageTimeSeriesType = USAGETIMESERIESTYPE_USAGE_TIMESERIES
	this.Type = &typeVar
	return &this
}

// NewUsageDataObjectWithDefaults instantiates a new UsageDataObject object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewUsageDataObjectWithDefaults() *UsageDataObject {
	this := UsageDataObject{}
	var typeVar UsageTimeSeriesType = USAGETIMESERIESTYPE_USAGE_TIMESERIES
	this.Type = &typeVar
	return &this
}

// GetAttributes returns the Attributes field value if set, zero value otherwise.
func (o *UsageDataObject) GetAttributes() UsageAttributesObject {
	if o == nil || o.Attributes == nil {
		var ret UsageAttributesObject
		return ret
	}
	return *o.Attributes
}

// GetAttributesOk returns a tuple with the Attributes field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageDataObject) GetAttributesOk() (*UsageAttributesObject, bool) {
	if o == nil || o.Attributes == nil {
		return nil, false
	}
	return o.Attributes, true
}

// HasAttributes returns a boolean if a field has been set.
func (o *UsageDataObject) HasAttributes() bool {
	return o != nil && o.Attributes != nil
}

// SetAttributes gets a reference to the given UsageAttributesObject and assigns it to the Attributes field.
func (o *UsageDataObject) SetAttributes(v UsageAttributesObject) {
	o.Attributes = &v
}

// GetId returns the Id field value if set, zero value otherwise.
func (o *UsageDataObject) GetId() string {
	if o == nil || o.Id == nil {
		var ret string
		return ret
	}
	return *o.Id
}

// GetIdOk returns a tuple with the Id field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageDataObject) GetIdOk() (*string, bool) {
	if o == nil || o.Id == nil {
		return nil, false
	}
	return o.Id, true
}

// HasId returns a boolean if a field has been set.
func (o *UsageDataObject) HasId() bool {
	return o != nil && o.Id != nil
}

// SetId gets a reference to the given string and assigns it to the Id field.
func (o *UsageDataObject) SetId(v string) {
	o.Id = &v
}

// GetType returns the Type field value if set, zero value otherwise.
func (o *UsageDataObject) GetType() UsageTimeSeriesType {
	if o == nil || o.Type == nil {
		var ret UsageTimeSeriesType
		return ret
	}
	return *o.Type
}

// GetTypeOk returns a tuple with the Type field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageDataObject) GetTypeOk() (*UsageTimeSeriesType, bool) {
	if o == nil || o.Type == nil {
		return nil, false
	}
	return o.Type, true
}

// HasType returns a boolean if a field has been set.
func (o *UsageDataObject) HasType() bool {
	return o != nil && o.Type != nil
}

// SetType gets a reference to the given UsageTimeSeriesType and assigns it to the Type field.
func (o *UsageDataObject) SetType(v UsageTimeSeriesType) {
	o.Type = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o UsageDataObject) MarshalJSON() ([]byte, error) {
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
func (o *UsageDataObject) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Attributes *UsageAttributesObject `json:"attributes,omitempty"`
		Id         *string                `json:"id,omitempty"`
		Type       *UsageTimeSeriesType   `json:"type,omitempty"`
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
