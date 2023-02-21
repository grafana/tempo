// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// ScalarResponse A message containing the response to a scalar query.
type ScalarResponse struct {
	// The object describing a scalar response.
	Attributes *ScalarFormulaResponseAtrributes `json:"attributes,omitempty"`
	// The type of the resource. The value should always be scalar_response.
	Type *ScalarFormulaResponseType `json:"type,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewScalarResponse instantiates a new ScalarResponse object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewScalarResponse() *ScalarResponse {
	this := ScalarResponse{}
	var typeVar ScalarFormulaResponseType = SCALARFORMULARESPONSETYPE_SCALAR_RESPONSE
	this.Type = &typeVar
	return &this
}

// NewScalarResponseWithDefaults instantiates a new ScalarResponse object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewScalarResponseWithDefaults() *ScalarResponse {
	this := ScalarResponse{}
	var typeVar ScalarFormulaResponseType = SCALARFORMULARESPONSETYPE_SCALAR_RESPONSE
	this.Type = &typeVar
	return &this
}

// GetAttributes returns the Attributes field value if set, zero value otherwise.
func (o *ScalarResponse) GetAttributes() ScalarFormulaResponseAtrributes {
	if o == nil || o.Attributes == nil {
		var ret ScalarFormulaResponseAtrributes
		return ret
	}
	return *o.Attributes
}

// GetAttributesOk returns a tuple with the Attributes field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ScalarResponse) GetAttributesOk() (*ScalarFormulaResponseAtrributes, bool) {
	if o == nil || o.Attributes == nil {
		return nil, false
	}
	return o.Attributes, true
}

// HasAttributes returns a boolean if a field has been set.
func (o *ScalarResponse) HasAttributes() bool {
	return o != nil && o.Attributes != nil
}

// SetAttributes gets a reference to the given ScalarFormulaResponseAtrributes and assigns it to the Attributes field.
func (o *ScalarResponse) SetAttributes(v ScalarFormulaResponseAtrributes) {
	o.Attributes = &v
}

// GetType returns the Type field value if set, zero value otherwise.
func (o *ScalarResponse) GetType() ScalarFormulaResponseType {
	if o == nil || o.Type == nil {
		var ret ScalarFormulaResponseType
		return ret
	}
	return *o.Type
}

// GetTypeOk returns a tuple with the Type field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ScalarResponse) GetTypeOk() (*ScalarFormulaResponseType, bool) {
	if o == nil || o.Type == nil {
		return nil, false
	}
	return o.Type, true
}

// HasType returns a boolean if a field has been set.
func (o *ScalarResponse) HasType() bool {
	return o != nil && o.Type != nil
}

// SetType gets a reference to the given ScalarFormulaResponseType and assigns it to the Type field.
func (o *ScalarResponse) SetType(v ScalarFormulaResponseType) {
	o.Type = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o ScalarResponse) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Attributes != nil {
		toSerialize["attributes"] = o.Attributes
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
func (o *ScalarResponse) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Attributes *ScalarFormulaResponseAtrributes `json:"attributes,omitempty"`
		Type       *ScalarFormulaResponseType       `json:"type,omitempty"`
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
	o.Type = all.Type
	return nil
}
